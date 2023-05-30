// SPDX-License-Identifier: ice License 1.0

package badges

import (
	"context"
	"sort"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/santa/tasks"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/ice-blockchain/wintr/log"
)

func (r *repository) achieveBadges(ctx context.Context, userID string) error { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	pr, err := r.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, ErrRelationNotFound) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	if pr == nil {
		pr = new(progress)
		pr.UserID = userID
	}
	if pr.AchievedBadges != nil && len(*pr.AchievedBadges) == len(&AllTypes) {
		return nil
	}
	achievedBadges := pr.reEvaluateAchievedBadges(r)
	if achievedBadges != nil && pr.AchievedBadges != nil && len(*pr.AchievedBadges) == len(*achievedBadges) {
		return nil
	}
	sql := `INSERT INTO badge_progress(achieved_badges, user_id) VALUES($1, $2) 
				ON CONFLICT (user_id) DO UPDATE
									SET achieved_badges = $1
				WHERE COALESCE(badge_progress.achieved_badges, ARRAY[]::TEXT[]) = COALESCE($3, ARRAY[]::TEXT[])`
	rowsUpserted, err := storage.Exec(ctx, r.db, sql, achievedBadges, userID, pr.AchievedBadges)
	if err != nil || rowsUpserted == 0 {
		if rowsUpserted == 0 && err == nil {
			return r.achieveBadges(ctx, userID)
		}

		return errors.Wrapf(err, "failed to insert BADGE_PROGRESS userID:%v, achievedBadges:%v", userID, achievedBadges)
	}
	//nolint:nestif // .
	if achievedBadges != nil && len(*achievedBadges) > 0 && (pr.AchievedBadges == nil || len(*pr.AchievedBadges) < len(*achievedBadges)) {
		achievedBadgesCount := make(map[GroupType]uint64, len(AllGroups))
		for _, achievedBadge := range *achievedBadges {
			achievedBadgesCount[GroupTypeForEachType[achievedBadge]]++
		}
		newlyAchievedBadges := make([]*AchievedBadge, 0, len(&AllTypes))
	outer:
		for _, achievedBadge := range *achievedBadges {
			if pr.AchievedBadges != nil {
				for _, previouslyAchievedBadge := range *pr.AchievedBadges {
					if achievedBadge == previouslyAchievedBadge {
						continue outer
					}
				}
			}
			groupType := GroupTypeForEachType[achievedBadge]
			newlyAchievedBadges = append(newlyAchievedBadges, &AchievedBadge{
				UserID:         userID,
				Type:           achievedBadge,
				Name:           AllNames[groupType][achievedBadge],
				GroupType:      groupType,
				AchievedBadges: achievedBadgesCount[groupType],
			})
		}
		if cErr := runConcurrently(ctx, r.sendAchievedBadgeMessage, newlyAchievedBadges); cErr != nil {
			sErr := errors.Wrapf(err, "failed to sendAchievedBadgeMessages for userID:%v,achievedBadges:%#v", userID, newlyAchievedBadges)
			sql = `UPDATE badge_progress 
						SET achieved_badges = $1
						WHERE user_id = $2 AND
							  COALESCE(badge_progress.achieved_badges, ARRAY[]::TEXT[]) = COALESCE($3, ARRAY[]::TEXT[])`
			rowsUpdated, uErr := storage.Exec(ctx, r.db, sql, pr.AchievedBadges, userID, achievedBadges)
			if uErr != nil || rowsUpdated == 0 {
				if rowsUpdated == 0 && uErr == nil {
					log.Error(errors.Wrapf(sErr, "[sendAchievedBadgeMessages]rollback race condition"))

					return r.achieveBadges(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(uErr, "[sendAchievedBadgeMessages][rollback] failed to update badge_progress.achieved_badges for achieved badges:%v and userID:%v", pr.AchievedBadges, userID), //nolint:lll // .
				).ErrorOrNil()
			}

			return sErr
		}
	}

	return nil
}

func (p *progress) reEvaluateAchievedBadges(repo *repository) *users.Enum[Type] { //nolint:funlen,gocognit,revive // .
	if p.AchievedBadges != nil && len(*p.AchievedBadges) == len(&AllTypes) {
		return p.AchievedBadges
	}
	alreadyAchievedBadges := make(map[Type]any, len(&AllTypes))
	if p.AchievedBadges != nil {
		for _, badge := range *p.AchievedBadges {
			alreadyAchievedBadges[badge] = struct{}{}
		}
	}
	achievedBadges := make(users.Enum[Type], 0, len(&AllTypes))
	for _, badgeType := range &AllTypes {
		if _, alreadyAchieved := alreadyAchievedBadges[badgeType]; alreadyAchieved {
			achievedBadges = append(achievedBadges, badgeType)

			continue
		}
		var achieved bool
		switch GroupTypeForEachType[badgeType] {
		case LevelGroupType:
			achieved = p.CompletedLevels >= repo.cfg.Milestones[badgeType].FromInclusive
		case CoinGroupType:
			if !p.Balance.IsZero() {
				achieved = p.Balance.GTE(coin.NewAmountUint64(repo.cfg.Milestones[badgeType].FromInclusive).MultiplyUint64(uint64(coin.Denomination)).Uint)
			}
		case SocialGroupType:
			achieved = p.FriendsInvited >= repo.cfg.Milestones[badgeType].FromInclusive
		}
		if achieved {
			achievedBadges = append(achievedBadges, badgeType)
		}
	}
	if len(achievedBadges) == 0 {
		return nil
	}
	sort.SliceStable(achievedBadges, func(i, j int) bool {
		return AllTypeOrder[achievedBadges[i]] < AllTypeOrder[achievedBadges[j]]
	})

	return &achievedBadges
}

func (r *repository) sendAchievedBadgeMessage(ctx context.Context, achievedBadge *AchievedBadge) error {
	valueBytes, err := json.MarshalContext(ctx, achievedBadge)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", achievedBadge)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     achievedBadge.UserID,
		Topic:   r.cfg.MessageBroker.Topics[2].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (r *repository) sendTryAchieveBadgesCommandMessage(ctx context.Context, userID string) error {
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   r.cfg.MessageBroker.Topics[1].Name,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (s *tryAchievedBadgesCommandSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}

	return errors.Wrapf(s.achieveBadges(ctx, msg.Key), "failed to achieveBadges for userID:%v", msg.Key)
}

func (s *userTableSource) Process(ctx context.Context, msg *messagebroker.Message) error { //nolint:gocognit // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	snapshot := new(users.UserSnapshot)
	if err := json.UnmarshalContext(ctx, msg.Value, snapshot); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), snapshot)
	}
	if (snapshot.Before == nil || snapshot.Before.ID == "") && (snapshot.User == nil || snapshot.User.ID == "") {
		return nil
	}
	if snapshot.Before != nil && snapshot.Before.ID != "" && (snapshot.User == nil || snapshot.User.ID == "") {
		return errors.Wrapf(s.deleteProgress(ctx, snapshot), "failed to delete progress for:%#v", snapshot)
	}
	if err := s.upsertProgress(ctx, snapshot); err != nil {
		return errors.Wrapf(err, "failed to upsert progress for:%#v", snapshot)
	}

	return nil
}

func (s *userTableSource) upsertProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	var hideBadges bool
	if us.HiddenProfileElements != nil {
		for _, hiddenElement := range *us.HiddenProfileElements {
			if users.BadgesHiddenProfileElement == hiddenElement {
				hideBadges = true

				break
			}
		}
	}
	sql := `INSERT INTO badge_progress(user_id, hide_badges) VALUES($1, $2)
				   ON CONFLICT(user_id)
				   DO UPDATE
					   SET hide_badges = EXCLUDED.hide_badges
				   WHERE COALESCE(badge_progress.hide_badges, FALSE) != COALESCE(EXCLUDED.hide_badges, FALSE)`
	_, err := storage.Exec(ctx, s.db, sql, us.ID, hideBadges)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", us),
		errors.Wrapf(s.insertReferrals(ctx, us), "failed to insertReferrals for user:%#v", us),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, us.ID), "failed to sendTryAchieveBadgesCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

//nolint:gocognit // .
func (s *userTableSource) insertReferrals(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil || us.User == nil || us.User.ReferredBy == "" || us.User.ReferredBy == us.User.ID || (us.Before != nil && us.Before.ID != "" && us.User.ReferredBy == us.Before.ReferredBy) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := []any{
		us.User.ID,
		us.User.ReferredBy,
	}
	sql := `INSERT INTO referrals(user_id,referred_by) VALUES($1, $2)`
	if _, err := storage.Exec(ctx, s.db, sql, params...); err != nil {
		if storage.IsErr(err, storage.ErrDuplicate) {
			return nil
		}

		return errors.Wrapf(err, "failed to insert referrals, params:%#v", params...)
	}
	if sErr := s.sendReferralsCountUpdate(ctx, us.User.ReferredBy); sErr != nil {
		sErr = errors.Wrapf(sErr, "failed to send referral counts update")
		sql = `DELETE FROM referrals WHERE user_id = $1 AND referred_by = $2`
		if _, err := storage.Exec(ctx, s.db, sql, params...); err != nil {
			return multierror.Append(sErr, //nolint:wrapcheck // Not needed.
				errors.Wrapf(err, "failed to delete referrals, params:%#v", params...)).ErrorOrNil()
		}

		return sErr
	}

	return errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, us.User.ReferredBy),
		"failed to sendTryAchieveBadgesCommandMessage, userID:%v,referredBy:%v", us.User.ID, us.User.ReferredBy)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `DELETE FROM badge_progress WHERE user_id = $1`
	_, err := storage.Exec(ctx, s.db, sql, us.Before.ID)

	return errors.Wrapf(err, "failed to delete badge_progress for:%#v", us)
}

func (r *repository) sendReferralsCountUpdate(ctx context.Context, userID string) error {
	pr, err := r.getProgress(ctx, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to get progress while sending referrals count update")
	}
	refCount := &tasks.ReferralsCount{
		UserID:    userID,
		Referrals: pr.FriendsInvited + 1,
	}
	valueBytes, err := json.MarshalContext(ctx, refCount)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", refCount)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   r.cfg.MessageBroker.Topics[3].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (r *referralCountsSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	refCount := new(tasks.ReferralsCount)
	if err := json.UnmarshalContext(ctx, msg.Value, refCount); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), refCount)
	}

	return errors.Wrapf(r.updateFriendsInvited(ctx, refCount), "failed to update")
}

func (r *referralCountsSource) updateFriendsInvited(ctx context.Context, refCount *tasks.ReferralsCount) error {
	sql := `INSERT INTO badge_progress(user_id, friends_invited) VALUES ($1, $2)
		   		ON CONFLICT(user_id) DO UPDATE  
		   			SET friends_invited = $2
		   		WHERE COALESCE(badge_progress.friends_invited, 0) != COALESCE(EXCLUDED.friends_invited, 0)`
	_, err := storage.Exec(ctx, r.db, sql, refCount.UserID, refCount.Referrals)

	return errors.Wrapf(err, "failed to set badge_progress.friends_invited, params:%#v", refCount)
}

func (s *completedLevelsSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	var cl levelsandroles.CompletedLevel
	if err := json.UnmarshalContext(ctx, msg.Value, &cl); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &cl)
	}
	if cl.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, cl.CompletedLevels, cl.UserID), "failed to upsertProgress for CompletedLevel:%#v", cl)
}

func (s *completedLevelsSource) upsertProgress(ctx context.Context, completedLevels uint64, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	pr, err := s.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.AchievedBadges != nil && (len(*pr.AchievedBadges) == len(&AllTypes))) ||
		(pr != nil && (pr.CompletedLevels == uint64(len(&levelsandroles.AllLevelTypes)) || IsBadgeGroupAchieved(pr.AchievedBadges, LevelGroupType))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	sql := `INSERT INTO badge_progress(user_id, completed_levels) VALUES($1, $2)
				ON CONFLICT(user_id)
				DO UPDATE
					SET completed_levels = EXCLUDED.completed_levels
				WHERE COALESCE(badge_progress.completed_levels, 0) != COALESCE(EXCLUDED.completed_levels, 0)`
	_, err = storage.Exec(ctx, s.db, sql, userID, int64(completedLevels))

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to insert/update progress for userID:%v, completedLevels:%v", userID, completedLevels),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, userID), "failed to sendTryAchieveBadgesCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *balancesTableSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	type (
		Balances struct {
			Standard   *coin.ICEFlake `json:"standard,omitempty" example:"124302"`
			PreStaking *coin.ICEFlake `json:"preStaking,omitempty" example:"124302"`
			UserID     string         `json:"userId,omitempty" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
		}
	)
	var bal Balances
	if err := json.UnmarshalContext(ctx, msg.Value, &bal); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &bal)
	}
	if bal.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, bal.Standard.Add(bal.PreStaking), bal.UserID), "failed to upsertProgress for Balances:%#v", bal)
}

func (s *balancesTableSource) upsertProgress(ctx context.Context, balance *coin.ICEFlake, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	pr, err := s.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.AchievedBadges != nil && (len(*pr.AchievedBadges) == len(&AllTypes) || IsBadgeGroupAchieved(pr.AchievedBadges, CoinGroupType))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	sql := `INSERT INTO badge_progress(user_id, balance) VALUES($1, $2)
				ON CONFLICT(user_id)
				DO UPDATE
					SET balance = $2
				WHERE COALESCE(badge_progress.balance, '') != COALESCE(EXCLUDED.balance, '')`
	_, err = storage.Exec(ctx, s.db, sql, userID, balance)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to insert/update progress balance:%v for userID:%v", balance, userID),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, userID), "failed to sendTryAchieveBadgesCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *globalTableSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	var val users.GlobalUnsigned
	if err := json.UnmarshalContext(ctx, msg.Value, &val); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &val)
	}
	if val.Key != "TOTAL_USERS" {
		return nil
	}
	sql := `UPDATE badge_statistics SET achieved_by = $1 WHERE badge_type IN ($2, $3, $4)`
	_, err := storage.Exec(ctx, s.db, sql, int64(val.Value), string(LevelGroupType), string(CoinGroupType), string(SocialGroupType))

	return errors.Wrapf(err, "failed to update badge_statistics from global unsigned value:%#v", &val)
}

func (s *achievedBadgesSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	var badge AchievedBadge
	if err := json.UnmarshalContext(ctx, msg.Value, &badge); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &badge)
	}
	sql := `INSERT INTO badge_statistics(badge_type, badge_group_type, achieved_by) VALUES($1, $2, 1)
				ON CONFLICT(badge_type)
				DO UPDATE
					SET achieved_by = badge_statistics.achieved_by+1`
	_, err := storage.Exec(ctx, s.db, sql, badge.Type, badge.GroupType)

	return errors.Wrapf(err, "error increasing badge statistics for badge:%v", badge.Type)
}
