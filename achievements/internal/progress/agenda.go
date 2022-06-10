// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

func (r *repository) updateAgendaPhoneNumbersHashes(ctx context.Context, userID UserID, agendaHashes string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed to update agenda phone number hashes because of context failed")
	}
	if agendaHashes == "" {
		return nil
	}
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: fieldAgendaPhoneNumbersHashes, Arg: agendaHashes}, // | agenda_phone_number_hashes = new value.
	}
	var res []*UserProgress
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, ops, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the agenda phone numbers hashes for userID:%v", userProgressSpace, userID)
	}

	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0]),
		"progress/mining sessions: failed to send updated progress message for UserID:%v", userID)
}

func (r *repository) insertAgendaReferrals(ctx context.Context, agendaOwnerID, userIDInAgenda UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed to insert agenda referrals because of context failed")
	}
	agendaReferral := &agendaReferrals{
		UserID:       userIDInAgenda,
		AgendaUserID: agendaOwnerID,
	}
	if err := r.db.InsertTyped(agendaReferralsSpace, agendaReferral, &[]*agendaReferrals{}); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_FOUND {
			return nil
		}

		return errors.Wrapf(err,
			"failed to insert agenda referrals record for user.ID:%v", agendaOwnerID)
	}
	count, err := r.getCountOfAgendaReferrals(ctx, agendaOwnerID)
	if err != nil {
		return errors.Wrapf(err, "failed to count agenda referrals for userID:%v", agendaOwnerID)
	}

	return errors.Wrapf(r.sendAgendaReferralsCountUpdate(ctx, agendaOwnerID, count),
		"progress: failed to send updated agenda referrals count for UserID:%v", agendaOwnerID)
}

func (r *repository) deleteAgendaReferrals(ctx context.Context, agendaOwnerID, userIDInAgenda UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed to delete referrals because of context failed")
	}
	agendaReferral := &agendaReferrals{
		UserID:       userIDInAgenda,
		AgendaUserID: agendaOwnerID,
	}
	var res []*agendaReferrals
	if err := r.db.DeleteTyped(agendaReferralsSpace, "pk_unnamed_AGENDA_REFERRALS_1", agendaReferral, &res); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_NOT_FOUND {
			return nil
		}

		return errors.Wrapf(err, "failed to delete agenda referrals")
	}
	count, err := r.getCountOfAgendaReferrals(ctx, agendaOwnerID)
	if err != nil {
		return errors.Wrapf(err, "failed to count agenda referrals for userID:%v", agendaOwnerID)
	}

	return errors.Wrapf(r.sendAgendaReferralsCountUpdate(ctx, agendaOwnerID, count),
		"progress: failed to send updated agenda referrals count for UserID:%v", agendaOwnerID)
}

func (r *repository) getCountOfAgendaReferrals(ctx context.Context, userID UserID) (uint64, error) {
	if ctx.Err() != nil {
		return 0, errors.Wrap(ctx.Err(), "failed to get count of users in agenda because of context failed")
	}
	var queryResult []*withCount
	sql := `select count(1) as c from agenda_referrals where agenda_user_id = :userId`
	params := map[string]interface{}{
		"userId": userID,
	}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil {
		return 0, errors.Wrapf(err, "failed to get count of users in agenda for userID:%v", userID)
	}
	if len(queryResult) == 0 {
		return 0, nil
	}

	return queryResult[0].Count, nil
}

func (r *repository) sendAgendaReferralsCountUpdate(ctx context.Context, userID UserID, countOfAgendaReferrals uint64) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed send agenda referrals count update because of context failed")
	}
	c := AgendaReferralsCount{
		UserID:               userID,
		AgendaReferralsCount: countOfAgendaReferrals,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return errors.Wrapf(err, "[user-progress] failed to marshal %#v", c)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   cfg.MessageBroker.Topics[3].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[user-progress] failed to send message to broker")
}
