// SPDX-License-Identifier: ice License 1.0

package friendsinvited

import (
	"context"
	stdlibtime "time"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage/v2"
)

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
		return errors.Wrapf(s.deleteReferrals(ctx, snapshot), "failed to delete progress for:%#v", snapshot)
	}

	return s.insertReferrals(ctx, snapshot, msg.Timestamp)
}

//nolint:gocognit // Transaction in one place.
func (s *userTableSource) insertReferrals(ctx context.Context, us *users.UserSnapshot, msgTimestamp stdlibtime.Time) error {
	if ctx.Err() != nil || us.User == nil || us.User.ReferredBy == "" || us.User.ReferredBy == us.User.ID || (us.Before != nil && us.Before.ID != "" && (us.User.ReferredBy == us.Before.ReferredBy || us.Before.ReferredBy != "")) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `INSERT INTO referrals(user_id,referred_by, processed_at) VALUES ($1,$2,$3)`
	params := []any{
		us.User.ID,
		us.User.ReferredBy,
		msgTimestamp,
	}

	return errors.Wrapf(storage.DoInTransaction(ctx, s.db, func(conn storage.QueryExecer) error {
		if _, err := storage.Exec(ctx, s.db, sql, params...); err != nil {
			if storage.IsErr(err, storage.ErrDuplicate) {
				return nil
			}

			return errors.Wrapf(err, "failed to insert referrals, params:%#v", params...)
		}
		sql = `
		INSERT INTO friends_invited(user_id,invited_count) VALUES ($1, 1)
		ON CONFLICT(user_id) DO UPDATE SET
		   invited_count = friends_invited.invited_count + 1
		RETURNING *`
		friends, err := storage.ExecOne[friendsInvited](ctx, s.db, sql, us.User.ReferredBy)
		if err != nil {
			return errors.Wrapf(err, "failed to increment friends_invited for userID:%v (ref:%v)", us.User.ReferredBy, us.User.ID)
		}

		return s.sendFriendsInvitedCountUpdate(ctx, friends)
	}), "insertReferrals: transaction failed for %#v", us)
}

func (s *userTableSource) deleteReferrals(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	_, errDelUser := storage.Exec(ctx, s.db, `DELETE FROM friends_invited WHERE user_id = $1`, us.Before.ID)

	return errors.Wrapf(errDelUser, "failed to delete task_progress for:%#v", us)
}

func (r *repository) sendFriendsInvitedCountUpdate(ctx context.Context, friends *friendsInvited) error {
	refCount := &Count{
		UserID: friends.UserID,
		Count:  uint64(friends.InvitedCount),
	}
	valueBytes, err := json.MarshalContext(ctx, refCount)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", refCount)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     friends.UserID,
		Topic:   r.cfg.MessageBroker.Topics[1].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}
