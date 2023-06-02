// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	friendsinvited "github.com/ice-blockchain/santa/friends-invited"
	"github.com/ice-blockchain/santa/tasks"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func (s *miningSessionSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	type (
		miningSession struct {
			UserID       string `json:"userId,omitempty" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
			MiningStreak uint64 `json:"miningStreak,omitempty" example:"11"`
		}
	)
	var ms miningSession
	if err := json.UnmarshalContext(ctx, msg.Value, &ms); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &ms)
	}
	if ms.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, ms.MiningStreak, ms.UserID), "failed to upsertProgress for miningSession:%#v", ms)
}

func (s *miningSessionSource) upsertProgress(ctx context.Context, miningStreak uint64, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if pr, err := s.getProgress(ctx, userID); err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.CompletedLevels != nil &&
			(len(*pr.CompletedLevels) == len(&AllLevelTypes) ||
				AreLevelsCompleted(pr.CompletedLevels, Level1Type, Level2Type, Level3Type, Level4Type, Level5Type))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	insertTuple := &progress{UserID: userID, MiningStreak: miningStreak}
	_, err := storage.Exec(ctx, s.db, `
		INSERT INTO levels_and_roles_progress(user_id, mining_streak) VALUES ($1,$2)
		ON CONFLICT (user_id) DO UPDATE 
			SET mining_streak = EXCLUDED.mining_streak
		WHERE levels_and_roles_progress.mining_streak != EXCLUDED.mining_streak`, insertTuple.UserID, insertTuple.MiningStreak)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, userID),
			"failed to sendTryCompleteLevelsCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *completedTasksSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	var ct tasks.CompletedTask
	if err := json.UnmarshalContext(ctx, msg.Value, &ct); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &ct)
	}
	if ct.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, ct.CompletedTasks, ct.UserID), "failed to upsertProgress for completedTask:%#v", ct)
}

func (s *completedTasksSource) upsertProgress(ctx context.Context, completedTasks uint64, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	pr, err := s.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.CompletedLevels != nil && (len(*pr.CompletedLevels) == len(&AllLevelTypes))) ||
		(pr != nil && (pr.CompletedTasks == uint64(len(&tasks.AllTypes)) ||
			AreLevelsCompleted(pr.CompletedLevels, Level6Type, Level7Type, Level8Type, Level9Type, Level10Type, Level11Type))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	insertTuple := &progress{UserID: userID, CompletedTasks: completedTasks}
	_, err = storage.Exec(ctx, s.db, `
		INSERT INTO levels_and_roles_progress(user_id, completed_tasks) VALUES ($1,$2)
		ON CONFLICT(user_id) DO UPDATE 
			SET completed_tasks = EXCLUDED.completed_tasks
		WHERE levels_and_roles_progress.completed_tasks != EXCLUDED.completed_tasks`, insertTuple.UserID, insertTuple.CompletedTasks)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, userID),
			"failed to sendTryCompleteLevelsCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *userPingsSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	type (
		userPing struct {
			LastPingCooldownEndedAt *time.Time `json:"lastPingCooldownEndedAt,omitempty" example:"2022-01-03T16:20:52.156534Z"`
			UserID                  string     `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
			PingedBy                string     `json:"pingedBy,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		}
	)
	ping := new(userPing)
	if err := json.UnmarshalContext(ctx, msg.Value, ping); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), ping)
	}
	if ping.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, ping.UserID, ping.PingedBy, ping.LastPingCooldownEndedAt), "failed to upsertProgress for ping:%#v", ping)
}

func (s *userPingsSource) upsertProgress(ctx context.Context, userID, pingedBy string, lastCooldown *time.Time) error {
	if pr, err := s.getProgress(ctx, userID); err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.CompletedLevels != nil &&
			(len(*pr.CompletedLevels) == len(&AllLevelTypes) ||
				AreLevelsCompleted(pr.CompletedLevels, Level16Type, Level17Type, Level18Type, Level19Type, Level20Type, Level21Type))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	sql := `INSERT INTO pings(user_id, pinged_by,last_ping_cooldown_ended_at) VALUES ($1,$2, $3)`
	params := []any{
		userID,
		pingedBy,
		lastCooldown.Time,
	}
	if _, err := storage.Exec(ctx, s.db, sql, params...); err != nil && storage.IsErr(err, storage.ErrDuplicate) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to insert pings, params:%#v", params...)
	}
	sql = `INSERT INTO levels_and_roles_progress (user_id, pings_sent)
				VALUES ($1, 1)
				ON CONFLICT(user_id) DO UPDATE 
		   			SET pings_sent = levels_and_roles_progress.pings_sent +1`
	if _, err := storage.Exec(ctx, s.db, sql, pingedBy); err != nil {
		return errors.Wrapf(err, "failed to set levels_and_roles_progress.pings_sent, params:%#v", params...)
	}

	return errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, pingedBy),
		"failed to sendTryCompleteLevelsCommandMessage, userID:%v,pingedBy:%v", userID, pingedBy)
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

func (s *userTableSource) upsertProgress(ctx context.Context, us *users.UserSnapshot) error { //nolint:funlen // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if us.PhoneNumberHash == us.ID {
		us.PhoneNumberHash = ""
	}
	var hideLevel, hideRole bool
	if us.HiddenProfileElements != nil {
		for _, hiddenElement := range *us.HiddenProfileElements {
			switch hiddenElement { //nolint:exhaustive // We only care about those.
			case users.LevelHiddenProfileElement:
				hideLevel = true
			case users.RoleHiddenProfileElement:
				hideRole = true
			}
		}
	}
	insertTuple := &progress{
		UserID:          us.ID,
		PhoneNumberHash: &us.PhoneNumberHash,
		HideLevel:       hideLevel,
		HideRole:        hideRole,
	}
	_, err := storage.Exec(ctx, s.db, `
		INSERT INTO levels_and_roles_progress(user_id, phone_number_hash, hide_level, hide_role)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET 
			    phone_number_hash = EXCLUDED.phone_number_hash,
			    hide_level = EXCLUDED.hide_level,
			    hide_role = EXCLUDED.hide_role
			WHERE levels_and_roles_progress.phone_number_hash != EXCLUDED.phone_number_hash
			   OR levels_and_roles_progress.hide_level != EXCLUDED.hide_level
			   OR levels_and_roles_progress.hide_role != EXCLUDED.hide_role`,
		insertTuple.UserID, insertTuple.PhoneNumberHash, insertTuple.HideLevel, insertTuple.HideRole)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, us.ID), "failed to sendTryCompleteLevelsCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

func (f *friendsInvitedSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	friends := new(friendsinvited.Count)
	if err := json.UnmarshalContext(ctx, msg.Value, friends); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), friends)
	}

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(f.updateFriendsInvited(ctx, friends), "failed to update levels friends invited count for %#v", friends),
		errors.Wrapf(f.sendTryCompleteLevelsCommandMessage(ctx, friends.UserID), "failed to sendTryCompleteLevelsCommandMessage for userID:%v", friends.UserID),
	).ErrorOrNil()
}

func (f *friendsInvitedSource) updateFriendsInvited(ctx context.Context, friends *friendsinvited.Count) error {
	sql := `INSERT INTO levels_and_roles_progress(user_id, friends_invited) VALUES ($1, $2)
		   	ON CONFLICT(user_id) DO UPDATE SET 
		   	    friends_invited = EXCLUDED.friends_invited
		   	WHERE levels_and_roles_progress.friends_invited != EXCLUDED.friends_invited`
	_, err := storage.Exec(ctx, f.db, sql, friends.UserID, friends.Count)

	return errors.Wrapf(err, "failed to set levels_and_roles_progress.friends_invited, params:%#v", friends)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	_, delProgressErr := storage.Exec(ctx, s.db, `DELETE FROM LEVELS_AND_ROLES_PROGRESS WHERE user_id = $1`, us.Before.ID)

	return errors.Wrapf(delProgressErr, "failed to delete LEVELS_AND_ROLES_PROGRESS for:%#v", us)
}

func (r *repository) sendTryCompleteLevelsCommandMessage(ctx context.Context, userID string) error {
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

func (s *tryCompleteLevelsCommandSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(s.completeLevels(ctx, msg.Key), "failed to completeLevels for userID:%v", msg.Key),
		errors.Wrapf(s.enableRoles(ctx, msg.Key), "failed to enableRoles for userID:%v", msg.Key),
	).ErrorOrNil()
}

func (r *repository) completeLevels(ctx context.Context, userID string) error { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	pr, err := r.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrRelationNotFound) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	if pr == nil {
		pr = new(progress)
		pr.UserID = userID
	}
	if pr.CompletedLevels != nil && len(*pr.CompletedLevels) == len(&AllLevelTypes) {
		return nil
	}
	completedLevels := pr.reEvaluateCompletedLevels(r)
	if completedLevels != nil && pr.CompletedLevels != nil && len(*pr.CompletedLevels) == len(*completedLevels) {
		return nil
	}
	sql := `INSERT INTO levels_and_roles_progress(user_id, completed_levels) VALUES ($1, $2)
				ON CONFLICT (user_id) DO UPDATE 
					SET completed_levels = EXCLUDED.completed_levels
				WHERE COALESCE(levels_and_roles_progress.completed_levels,ARRAY[]::TEXT[]) = COALESCE($3,ARRAY[]::TEXT[])`
	params := []any{
		pr.UserID,
		completedLevels,
		pr.CompletedLevels,
	}
	if rowsUpdated, uErr := storage.Exec(ctx, r.db, sql, params...); uErr == nil && rowsUpdated == 0 {
		return r.completeLevels(ctx, userID)
	} else if uErr != nil {
		return errors.Wrapf(uErr, "failed to update LEVELS_AND_ROLES_PROGRESS.completed_levels for params:%#v", params...)
	}
	if completedLevels != nil && len(*completedLevels) > 0 && (pr.CompletedLevels == nil || len(*pr.CompletedLevels) < len(*completedLevels)) { //nolint:nestif,lll // .
		newlyCompletedLevels := make([]*CompletedLevel, 0, len(&AllLevelTypes))
	outer:
		for _, completedLevel := range *completedLevels {
			if pr.CompletedLevels != nil {
				for _, previouslyCompletedLevel := range *pr.CompletedLevels {
					if completedLevel == previouslyCompletedLevel {
						continue outer
					}
				}
			}
			newlyCompletedLevels = append(newlyCompletedLevels, &CompletedLevel{
				UserID:          userID,
				Type:            completedLevel,
				CompletedLevels: uint64(len(*completedLevels)),
			})
		}
		if err = runConcurrently(ctx, r.sendCompletedLevelMessage, newlyCompletedLevels); err != nil {
			sErr := errors.Wrapf(err, "failed to sendCompletedLevelMessages for userID:%v,completedLevels:%#v", userID, newlyCompletedLevels)
			params[1] = pr.CompletedLevels
			params[2] = completedLevels
			if rowsUpdated, rErr := storage.Exec(ctx, r.db, sql, params...); rowsUpdated == 0 && rErr == nil {
				return r.completeLevels(ctx, userID)
			} else if rErr != nil {
				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[sendCompletedLevelMessages][rollback]failed to update LEVELS_AND_ROLES_PROGRESS.completed_levels, params:%#v", params...),
				).ErrorOrNil()
			}

			return sErr
		}
	}

	return nil
}

func (p *progress) reEvaluateCompletedLevels(repo *repository) *users.Enum[LevelType] { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if p.CompletedLevels != nil && len(*p.CompletedLevels) == len(&AllLevelTypes) {
		return p.CompletedLevels
	}
	alreadyCompletedLevels := make(map[LevelType]any, len(&AllLevelTypes))
	if p.CompletedLevels != nil {
		for _, level := range *p.CompletedLevels {
			alreadyCompletedLevels[level] = struct{}{}
		}
	}
	completedLevels := make(users.Enum[LevelType], 0, len(&AllLevelTypes))
	for _, levelType := range &AllLevelTypes {
		if _, alreadyCompleted := alreadyCompletedLevels[levelType]; alreadyCompleted {
			completedLevels = append(completedLevels, levelType)

			continue
		}
		var completed bool
		//nolint:nestif // .
		if milestone, found := repo.cfg.MiningStreakMilestones[levelType]; found {
			if p.MiningStreak >= milestone {
				completed = true
			}
		} else if milestone, found = repo.cfg.PingsSentMilestones[levelType]; found {
			if p.PingsSent >= milestone {
				completed = true
			}
		} else if milestone, found = repo.cfg.AgendaContactsJoinedMilestones[levelType]; found {
			if p.PhoneNumberHash != nil && *p.PhoneNumberHash != "" && p.AgendaContactsJoined >= milestone {
				completed = true
			}
		} else if milestone, found = repo.cfg.CompletedTasksMilestones[levelType]; found {
			if p.CompletedTasks >= milestone {
				completed = true
			}
		}
		if completed {
			completedLevels = append(completedLevels, levelType)
		}
	}
	if len(completedLevels) == 0 {
		return nil
	}

	return &completedLevels
}

func (r *repository) sendCompletedLevelMessage(ctx context.Context, completedLevel *CompletedLevel) error {
	valueBytes, err := json.MarshalContext(ctx, completedLevel)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", completedLevel)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     completedLevel.UserID,
		Topic:   r.cfg.MessageBroker.Topics[2].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (r *repository) enableRoles(ctx context.Context, userID string) error { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	pr, err := r.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrRelationNotFound) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	if pr == nil {
		pr = new(progress)
		pr.UserID = userID
	}
	if pr.EnabledRoles != nil && len(*pr.EnabledRoles) == len(&AllRoleTypesThatCanBeEnabled) {
		return nil
	}
	enabledRoles := pr.reEvaluateEnabledRoles(r)
	if enabledRoles == nil || (pr.EnabledRoles != nil && len(*pr.EnabledRoles) == len(*enabledRoles)) {
		return nil
	}
	sql := `INSERT INTO levels_and_roles_progress (user_id, enabled_roles) VALUES ($1, $2)
				ON CONFLICT (user_id) DO UPDATE 
					SET enabled_roles = EXCLUDED.enabled_roles
				WHERE COALESCE(levels_and_roles_progress.enabled_roles,ARRAY[]::TEXT[]) = COALESCE($3,ARRAY[]::TEXT[])`
	params := []any{
		pr.UserID,
		enabledRoles,
		pr.EnabledRoles,
	}
	if rowsUpdated, uErr := storage.Exec(ctx, r.db, sql, params...); uErr == nil && rowsUpdated == 0 {
		return r.enableRoles(ctx, userID)
	} else if uErr != nil {
		return errors.Wrapf(uErr, "failed to insert LEVELS_AND_ROLES_PROGRESS.enabled_roles for params:%#v", params...)
	}
	if len(*enabledRoles) > 0 && (pr.EnabledRoles == nil || len(*pr.EnabledRoles) < len(*enabledRoles)) { //nolint:nestif // .
		newlyEnabledRoles := make([]*EnabledRole, 0, len(&AllRoleTypesThatCanBeEnabled))
	outer:
		for _, enabledRole := range *enabledRoles {
			if pr.EnabledRoles != nil {
				for _, previouslyEnabledRole := range *pr.EnabledRoles {
					if enabledRole == previouslyEnabledRole {
						continue outer
					}
				}
			}
			newlyEnabledRoles = append(newlyEnabledRoles, &EnabledRole{
				UserID: userID,
				Type:   enabledRole,
			})
		}
		if err = runConcurrently(ctx, r.sendEnabledRoleMessage, newlyEnabledRoles); err != nil {
			sErr := errors.Wrapf(err, "failed to sendEnabledRoleMessages for userID:%v,enabledRoles:%#v", userID, newlyEnabledRoles)
			params[1] = pr.EnabledRoles
			params[2] = enabledRoles
			if rowsUpdated, rErr := storage.Exec(ctx, r.db, sql, params...); rowsUpdated == 0 && rErr == nil {
				log.Error(errors.Wrapf(sErr, "[sendEnabledRoleMessages]rollback race condition"))

				return r.enableRoles(ctx, userID)
			} else if rErr != nil {
				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(rErr, "[sendEnabledRoleMessages][rollback]failed to update LEVELS_AND_ROLES_PROGRESS.enabled_roles, params:%#v", params...),
				).ErrorOrNil()
			}

			return sErr
		}
	}

	return nil
}

func (p *progress) reEvaluateEnabledRoles(repo *repository) *users.Enum[RoleType] {
	if p.EnabledRoles != nil && len(*p.EnabledRoles) == len(&AllRoleTypesThatCanBeEnabled) {
		return p.EnabledRoles
	}
	if p.FriendsInvited >= repo.cfg.RequiredInvitedFriendsToBecomeAmbassador {
		completedLevels := append(make(users.Enum[RoleType], 0, len(&AllRoleTypesThatCanBeEnabled)), AmbassadorRoleType)

		return &completedLevels
	}

	return nil
}

func (r *repository) sendEnabledRoleMessage(ctx context.Context, enabledRole *EnabledRole) error {
	valueBytes, err := json.MarshalContext(ctx, enabledRole)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", enabledRole)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     enabledRole.UserID,
		Topic:   r.cfg.MessageBroker.Topics[3].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}
