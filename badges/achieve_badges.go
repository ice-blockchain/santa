// SPDX-License-Identifier: ice License 1.0

package badges

import (
	"context"
	"fmt"
	"sort"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/go-tarantool-client"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
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
	sql := `UPDATE badge_progress
			SET achieved_badges = :achieved_badges
			WHERE user_id = :user_id
              AND IFNULL(achieved_badges,'') = IFNULL(:old_achieved_badges,'')`
	params := make(map[string]any, 1+1+1)
	params["user_id"] = pr.UserID
	params["achieved_badges"] = achievedBadges
	params["old_achieved_badges"] = pr.AchievedBadges
	//nolint:nestif // .
	if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			tuple := &progress{AchievedBadges: achievedBadges, UserID: pr.UserID}
			if err = storage.CheckNoSQLDMLErr(r.db.InsertTyped("BADGE_PROGRESS", tuple, &[]*progress{})); err != nil && errors.Is(err, storage.ErrDuplicate) {
				return r.achieveBadges(ctx, userID)
			}
			if err != nil {
				return errors.Wrapf(err, "failed to insert BADGE_PROGRESS %#v", tuple)
			}
		}
		if err != nil {
			return errors.Wrapf(err, "failed to update badge_progress.achieved_badges for params:%#v", params)
		}
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
		if err = runConcurrently(ctx, r.sendAchievedBadgeMessage, newlyAchievedBadges); err != nil {
			sErr := errors.Wrapf(err, "failed to sendAchievedBadgeMessages for userID:%v,achievedBadges:%#v", userID, newlyAchievedBadges)
			params["achieved_badges"] = pr.AchievedBadges
			params["old_achieved_badges"] = achievedBadges
			if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Error(errors.Wrapf(sErr, "[sendAchievedBadgeMessages]rollback race condition"))

					return r.achieveBadges(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[sendAchievedBadgeMessages][rollback] failed to update badge_progress.achieved_badges for params:%#v", params),
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
	insertTuple := &progress{
		UserID:     us.ID,
		HideBadges: hideBadges,
	}
	const hideBadgesColumnIndex = 5
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: hideBadgesColumnIndex, Arg: insertTuple.HideBadges})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("BADGE_PROGRESS", insertTuple, ops, &[]*progress{})), "failed to upsert progress for %#v", us),
		errors.Wrapf(s.updateFriendsInvited(ctx, us), "failed to updateFriendsInvited for user:%#v", us),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, us.ID), "failed to sendTryAchieveBadgesCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

//nolint:gocognit // .
func (s *userTableSource) updateFriendsInvited(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil || us.User == nil || us.User.ReferredBy == "" || us.User.ReferredBy == us.User.ID || (us.Before != nil && us.Before.ID != "" && us.User.ReferredBy == us.Before.ReferredBy) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `REPLACE INTO referrals(user_id,referred_by) VALUES (:user_id,:referred_by)`
	params := make(map[string]any, 1+1)
	params["user_id"] = us.User.ID
	params["referred_by"] = us.User.ReferredBy
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		return errors.Wrapf(err, "failed to REPLACE INTO referrals, params:%#v", params)
	}
	delete(params, "user_id")
	sql = `UPDATE badge_progress 
		   SET friends_invited = (SELECT COUNT(*) FROM referrals WHERE referred_by = :referred_by)
		   WHERE user_id = :referred_by`
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.InsertTyped("BADGE_PROGRESS", &progress{UserID: us.User.ReferredBy, FriendsInvited: 1}, &[]*progress{})), "failed to insert progress for referredBy:%v, from :%#v", us.User.ReferredBy, us) //nolint:lll // .
		}
		if err != nil && !errors.Is(err, storage.ErrDuplicate) {
			return errors.Wrapf(err, "failed to set badge_progress.friends_invited, params:%#v", params)
		}
	}

	return errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, us.User.ReferredBy),
		"failed to sendTryAchieveBadgesCommandMessage, userID:%v,referredBy:%v", us.User.ID, us.User.ReferredBy)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := map[string]any{"user_id": us.Before.ID}

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM badge_progress WHERE user_id = :user_id`, params)),
			"failed to delete badge_progress for:%#v", us),
	).ErrorOrNil()
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
	const completedLevelsColumnIndex = 4
	insertTuple := &progress{UserID: userID, CompletedLevels: completedLevels}
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: completedLevelsColumnIndex, Arg: insertTuple.CompletedLevels})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("BADGE_PROGRESS", insertTuple, ops, &[]*progress{})),
			"failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, userID),
			"failed to sendTryAchieveBadgesCommandMessage for userID:%v", userID),
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
	const balanceColumnIndex = 1
	insertTuple := &progress{UserID: userID, Balance: balance}
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: balanceColumnIndex, Arg: insertTuple.Balance})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("BADGE_PROGRESS", insertTuple, ops, &[]*progress{})),
			"failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryAchieveBadgesCommandMessage(ctx, userID),
			"failed to sendTryAchieveBadgesCommandMessage for userID:%v", userID),
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
	sql := fmt.Sprintf(`UPDATE badge_statistics
						SET achieved_by = :achieved_by
						WHERE badge_type IN ('%v','%v','%v')`, LevelGroupType, CoinGroupType, SocialGroupType)
	params := make(map[string]any, 1)
	params["achieved_by"] = val.Value

	return errors.Wrapf(storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)), "failed to update badge_statistics from global unsigned value:%#v", &val)
}

func (s *achievedBadgesSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	var badge AchievedBadge
	if err := json.UnmarshalContext(ctx, msg.Value, &badge); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &badge)
	}
	const achievedByColumnIndex = 2
	incOp := []tarantool.Op{{Op: "+", Field: achievedByColumnIndex, Arg: 1}}
	tuple := &statistics{Type: badge.Type}

	return errors.Wrapf(s.db.UpsertTyped("BADGE_STATISTICS", tuple, incOp, &[]*statistics{}),
		"error increasing badge statistics for badge:%v", badge.Type)
}
