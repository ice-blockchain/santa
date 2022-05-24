package global

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"math"
)

func NewTotalUsersProcessor(db tarantool.Connector) messagebroker.Processor {
	return &totalUsersSource{
		db: db,
		p:  progress.NewRepository(db),
	}
}

func (u *totalUsersSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "achievements/totalUsersSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}

	// User deletion, we need to handle it, update total_users in GLOBAL and delete him from USER_ACHIEVEMENTS
	// and decrement t1 referrals count for its parent if user was referred by another user.
	if user.User == nil && user.Before != nil {
		if err := u.updateTotalUsersCount(-1); err != nil {
			return errors.Wrapf(err, "failed to update total_users counter")
		}

		return nil // We're complete here with deletion.
	}
	_, err := u.p.GetUserProgress(user.ID)
	if errors.Is(err, storage.ErrNotFound) {
		// User's achievements record does not exists, so it is a new user - increment counter.
		if err = u.updateTotalUsersCount(1); err != nil {
			return errors.Wrapf(err, "failed to update total_users counter")
		}
	}

	return nil
}

func (u *totalUsersSource) updateTotalUsersCount(diff int64) error {
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: 1, Arg: diff},
	}

	return errors.Wrap(u.db.UpsertAsync("GLOBAL", &global{Key: "TOTAL_USERS", Value: 1}, incrementOps).GetTyped(&[]*global{}),
		"failed to update global record the KEY = 'TOTAL_USERS'")
}
