// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/go-tarantool-client"
	"github.com/ice-blockchain/santa/tasks"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/log"
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
	const miningStreakColumnIndex = 4
	insertTuple := &progress{UserID: userID, MiningStreak: miningStreak}
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: miningStreakColumnIndex, Arg: insertTuple.MiningStreak})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("LEVELS_AND_ROLES_PROGRESS", insertTuple, ops, &[]*progress{})),
			"failed to upsert progress for %#v", insertTuple),
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
	const completedTasksColumnIndex = 8
	insertTuple := &progress{UserID: userID, CompletedTasks: completedTasks}
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: completedTasksColumnIndex, Arg: insertTuple.CompletedTasks})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("LEVELS_AND_ROLES_PROGRESS", insertTuple, ops, &[]*progress{})),
			"failed to upsert progress for %#v", insertTuple),
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
			UserID   string `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
			PingedBy string `json:"pingedBy,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		}
	)
	ping := new(userPing)
	if err := json.UnmarshalContext(ctx, msg.Value, ping); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), ping)
	}
	if ping.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, ping.UserID, ping.PingedBy), "failed to upsertProgress for ping:%#v", ping)
}

func (s *userPingsSource) upsertProgress(ctx context.Context, userID, pingedBy string) error { //nolint:gocognit // .
	if pr, err := s.getProgress(ctx, userID); err != nil && !errors.Is(err, storage.ErrRelationNotFound) ||
		(pr != nil && pr.CompletedLevels != nil &&
			(len(*pr.CompletedLevels) == len(&AllLevelTypes) ||
				AreLevelsCompleted(pr.CompletedLevels, Level16Type, Level17Type, Level18Type, Level19Type, Level20Type, Level21Type))) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	sql := `REPLACE INTO pings(user_id,pinged_by) VALUES (:user_id,:pinged_by)`
	params := make(map[string]any, 1+1)
	params["user_id"] = userID
	params["pinged_by"] = pingedBy
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		return errors.Wrapf(err, "failed to REPLACE INTO pings, params:%#v", params)
	}
	delete(params, "user_id")
	sql = `UPDATE levels_and_roles_progress 
		   SET pings_sent = (SELECT COUNT(*) FROM pings WHERE pinged_by = :pinged_by)
		   WHERE user_id = :pinged_by`
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.InsertTyped("LEVELS_AND_ROLES_PROGRESS", &progress{UserID: pingedBy, PingsSent: 1}, &[]*progress{})), "failed to insert progress for userID:%v,pingedBy:%v", userID, pingedBy) //nolint:lll // .
		}
		if err != nil && !errors.Is(err, storage.ErrDuplicate) {
			return errors.Wrapf(err, "failed to set levels_and_roles_progress.pings_sent, params:%#v", params)
		}
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
		PhoneNumberHash: us.PhoneNumberHash,
		HideLevel:       hideLevel,
		HideRole:        hideRole,
	}
	const phoneNumberHashColumnIndex, hideLevelColumnIndex, hideRoleColumnIndex = 3, 9, 10
	ops := append(make([]tarantool.Op, 0, 1+1+1),
		tarantool.Op{Op: "=", Field: phoneNumberHashColumnIndex, Arg: insertTuple.PhoneNumberHash},
		tarantool.Op{Op: "=", Field: hideLevelColumnIndex, Arg: insertTuple.HideLevel},
		tarantool.Op{Op: "=", Field: hideRoleColumnIndex, Arg: insertTuple.HideRole})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("LEVELS_AND_ROLES_PROGRESS", insertTuple, ops, &[]*progress{})), "failed to upsert progress for %#v", insertTuple), //nolint:lll // .
		errors.Wrapf(s.updateAgendaContactsJoined(ctx, us), "failed to updateAgendaContactsJoined for user:%#v", us),
		errors.Wrapf(s.updateFriendsInvited(ctx, us), "failed to updateFriendsInvited for user:%#v", us),
		errors.Wrapf(s.insertAgendaPhoneNumberHashes(ctx, us), "failed to insertAgendaPhoneNumberHashes for user:%#v", us),
		errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, us.ID), "failed to sendTryCompleteLevelsCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

//nolint:gocognit,funlen // .
func (s *userTableSource) updateAgendaContactsJoined(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil || us.User == nil || us.User.PhoneNumberHash == us.User.ID || us.User.PhoneNumberHash == "" || (us.Before != nil && us.Before.PhoneNumberHash == us.User.PhoneNumberHash) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `SELECT user_id
		   FROM agenda_phone_number_hashes
		   WHERE agenda_phone_number_hash = :phone_number_hash`
	params := make(map[string]any, 1)
	params["phone_number_hash"] = us.User.PhoneNumberHash
	resp := make([]*struct {
		_msgpack struct{} `msgpack:",asArray"` //nolint:tagliatelle,revive,nosnakecase // To insert we need asArray
		UserID   string
	}, 0)
	if err := s.db.PrepareExecuteTyped(sql, params, &resp); err != nil || len(resp) == 0 {
		return errors.Wrapf(err, "failed to select for userID's that have this phone_number_hash in their phone's agenda, params:%#v", params)
	}
	params = make(map[string]any, len(resp))
	userIDs, values, conditions := make([]string, 0, len(resp)), make([]string, 0, len(resp)), make([]string, 0, len(resp))
	for ix, row := range resp {
		userIDs = append(userIDs, row.UserID)
		params[fmt.Sprintf("user_id%[1]v", ix)] = row.UserID
		values = append(values, fmt.Sprintf(":user_id%[1]v", ix))
		conditions = append(conditions, fmt.Sprintf(`WHEN user_id = :user_id%[1]v 
																THEN (SELECT COUNT(*) 
																	  FROM agenda_phone_number_hashes h 
																			JOIN levels_and_roles_progress p 
																			  ON p.phone_number_hash = h.agenda_phone_number_hash
																	  WHERE h.user_id = :user_id%[1]v)`, ix))
	}
	sql = fmt.Sprintf(`UPDATE levels_and_roles_progress
					   SET agenda_contacts_joined = (CASE %[2]v ELSE agenda_contacts_joined END)
					   WHERE user_id IN (%[1]v)`, strings.Join(values, ","), strings.Join(conditions, ","))
	if _, err := storage.CheckSQLDMLResponse(s.db.PrepareExecute(sql, params)); err != nil {
		return errors.Wrapf(err, "failed to update levels_and_roles_progress.agenda_contacts_joined, params:%#v", params)
	}

	return errors.Wrapf(runConcurrently(ctx, s.sendTryCompleteLevelsCommandMessage, userIDs), "failed to runConcurrently[sendTryCompleteLevelsCommandMessage], userIDs:%#v", userIDs) //nolint:lll // .
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
	sql = `UPDATE levels_and_roles_progress 
		   SET friends_invited = (SELECT COUNT(*) FROM referrals WHERE referred_by = :referred_by)
		   WHERE user_id = :referred_by`
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.InsertTyped("LEVELS_AND_ROLES_PROGRESS", &progress{UserID: us.User.ReferredBy, FriendsInvited: 1}, &[]*progress{})), "failed to insert progress for referredBy:%v, from :%#v", us.User.ReferredBy, us) //nolint:lll // .
		}
		if err != nil && !errors.Is(err, storage.ErrDuplicate) {
			return errors.Wrapf(err, "failed to set levels_and_roles_progress.friends_invited, params:%#v", params)
		}
	}

	return errors.Wrapf(s.sendTryCompleteLevelsCommandMessage(ctx, us.User.ReferredBy),
		"failed to sendTryCompleteLevelsCommandMessage, userID:%v,referredBy:%v", us.User.ID, us.User.ReferredBy)
}

func (s *userTableSource) insertAgendaPhoneNumberHashes(ctx context.Context, us *users.UserSnapshot) error {
	contacts := s.newlyAddedAgendaContacts(us)
	if len(contacts) == 0 {
		return nil
	}
	var (
		jx                     = 0
		allContactsBatches     = make([][]string, 0, (len(contacts)/agendaPhoneNumberHashesBatchSize)+1)
		currentContactsBatches = make([]string, 0, int(math.Min(float64(len(contacts)), agendaPhoneNumberHashesBatchSize)))
	)
	for contact := range contacts {
		if jx != 0 && jx%agendaPhoneNumberHashesBatchSize == 0 {
			allContactsBatches = append(allContactsBatches, currentContactsBatches)
			currentContactsBatches = currentContactsBatches[:0]
		}
		currentContactsBatches = append(currentContactsBatches, contact)
		jx++
	}

	return errors.Wrap(runConcurrently(ctx, func(ctx context.Context, contactsBatch []string) error {
		values := make([]string, 0, len(contactsBatch))
		params := make(map[string]any, len(contactsBatch)+1)
		params["user_id"] = us.ID
		for ix, contact := range contactsBatch {
			values = append(values, fmt.Sprintf("(:user_id,:c%v)", ix))
			params[fmt.Sprintf("c%v", ix)] = contact
		}
		sql := fmt.Sprintf(`REPLACE INTO agenda_phone_number_hashes(user_id, agenda_phone_number_hash) VALUES %v`, strings.Join(values, ","))

		return errors.Wrapf(storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)), "failed to REPLACE INTO agenda_phone_number_hashes, params:%#v", params)
	}, allContactsBatches), "at least one REPLACE INTO agenda_phone_number_hashes batch failed")
}

func (*userTableSource) newlyAddedAgendaContacts(us *users.UserSnapshot) map[string]struct{} { //nolint:gocognit,gocyclo,revive,cyclop // .
	if us.User == nil || us.User.ID == "" || us.User.AgendaPhoneNumberHashes == "" || us.User.AgendaPhoneNumberHashes == us.User.ID {
		return nil
	}
	after := strings.Split(us.User.AgendaPhoneNumberHashes, ",")
	newlyAdded := make(map[string]struct{}, len(after))
	if us.Before == nil || us.Before.ID == "" || us.Before.AgendaPhoneNumberHashes == "" || us.Before.AgendaPhoneNumberHashes == us.User.ID {
		for _, agendaPhoneNumberHash := range after {
			if agendaPhoneNumberHash == "" {
				continue
			}
			newlyAdded[agendaPhoneNumberHash] = struct{}{}
		}

		return newlyAdded
	}
	before := strings.Split(us.Before.AgendaPhoneNumberHashes, ",")
outer:
	for _, afterAgendaPhoneNumberHash := range after {
		if afterAgendaPhoneNumberHash == "" || strings.EqualFold(afterAgendaPhoneNumberHash, us.User.PhoneNumberHash) {
			continue
		}
		for _, beforeAgendaPhoneNumberHash := range before {
			if strings.EqualFold(beforeAgendaPhoneNumberHash, afterAgendaPhoneNumberHash) {
				continue outer
			}
		}
		newlyAdded[afterAgendaPhoneNumberHash] = struct{}{}
	}

	return newlyAdded
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := map[string]any{"user_id": us.Before.ID}

	var mErr *multierror.Error
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM LEVELS_AND_ROLES_PROGRESS WHERE user_id = :user_id`, params)); err != nil {
		mErr = multierror.Append(errors.Wrapf(err, "failed to delete LEVELS_AND_ROLES_PROGRESS for:%#v", us))
	}
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM AGENDA_PHONE_NUMBER_HASHES WHERE user_id = :user_id`, params)); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			mErr = multierror.Append(errors.Wrapf(err, "failed to delete AGENDA_PHONE_NUMBER_HASHES for:%#v", us))
		}
	}
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM PINGS WHERE user_id = :user_id`, params)); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			mErr = multierror.Append(errors.Wrapf(err, "failed to delete PINGS for:%#v", us))
		}
	}

	return mErr.ErrorOrNil() //nolint:wrapcheck // Not needed.
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
	sql := `UPDATE levels_and_roles_progress
			SET completed_levels = :completed_levels
			WHERE user_id = :user_id
              AND IFNULL(completed_levels,'') = IFNULL(:old_completed_levels,'')`
	params := make(map[string]any, 1+1+1)
	params["user_id"] = pr.UserID
	params["completed_levels"] = completedLevels
	params["old_completed_levels"] = pr.CompletedLevels
	//nolint:nestif // .
	if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			tuple := &progress{CompletedLevels: completedLevels, UserID: pr.UserID}
			err = storage.CheckNoSQLDMLErr(r.db.InsertTyped("LEVELS_AND_ROLES_PROGRESS", tuple, &[]*progress{}))
			if err != nil && errors.Is(err, storage.ErrDuplicate) {
				return r.completeLevels(ctx, userID)
			}

			if err != nil {
				return errors.Wrapf(err, "failed to insert LEVELS_AND_ROLES_PROGRESS %#v", tuple)
			}
		}
		if err != nil {
			return errors.Wrapf(err, "failed to update LEVELS_AND_ROLES_PROGRESS.completed_levels for params:%#v", params)
		}
	}
	//nolint:nestif // .
	if completedLevels != nil && len(*completedLevels) > 0 && (pr.CompletedLevels == nil || len(*pr.CompletedLevels) < len(*completedLevels)) {
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
			params["completed_levels"] = pr.CompletedLevels
			params["old_completed_levels"] = completedLevels
			if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Error(errors.Wrapf(sErr, "[sendCompletedLevelMessages]rollback race condition"))

					return r.completeLevels(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[sendCompletedLevelMessages][rollback]failed to update LEVELS_AND_ROLES_PROGRESS.completed_levels, params:%#v", params),
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
			if p.PhoneNumberHash != "" && p.AgendaContactsJoined >= milestone {
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
	sql := `UPDATE levels_and_roles_progress
			SET enabled_roles = :enabled_roles
			WHERE user_id = :user_id
              AND IFNULL(enabled_roles,'') = IFNULL(:old_enabled_roles,'')`
	params := make(map[string]any, 1+1+1)
	params["user_id"] = pr.UserID
	params["enabled_roles"] = enabledRoles
	params["old_enabled_roles"] = pr.EnabledRoles
	//nolint:nestif // .
	if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			tuple := &progress{EnabledRoles: enabledRoles, UserID: pr.UserID}
			err = storage.CheckNoSQLDMLErr(r.db.InsertTyped("LEVELS_AND_ROLES_PROGRESS", tuple, &[]*progress{}))
			if err != nil && errors.Is(err, storage.ErrDuplicate) {
				return r.enableRoles(ctx, userID)
			}

			if err != nil {
				return errors.Wrapf(err, "failed to insert LEVELS_AND_ROLES_PROGRESS %#v", tuple)
			}
		}
		if err != nil {
			return errors.Wrapf(err, "failed to update LEVELS_AND_ROLES_PROGRESS.enabled_roles for params:%#v", params)
		}
	}
	//nolint:nestif // .
	if len(*enabledRoles) > 0 && (pr.EnabledRoles == nil || len(*pr.EnabledRoles) < len(*enabledRoles)) {
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
			params["enabled_roles"] = pr.EnabledRoles
			params["old_enabled_roles"] = enabledRoles
			if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Error(errors.Wrapf(sErr, "[sendEnabledRoleMessages]rollback race condition"))

					return r.enableRoles(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[sendEnabledRoleMessages][rollback]failed to update LEVELS_AND_ROLES_PROGRESS.enabled_roles, params:%#v", params),
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
