package levels

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewUserSource(db tarantool.Connector) messagebroker.Processor {
	return &userSource{
		r: &repository{db: db},
	}
}

func (u *userSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "levels/userSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}

	return errors.Wrapf(u.achieveLevels(ctx, user), "levels/userSource: failed to increment user's level for the phone number confirmation")
}

func (u *userSource) achieveLevels(ctx context.Context, user *users.UserSnapshot) error {
	// New level for user (Levels -> 8 Confirm phone number)
	// it seems eskimo can send unconfirmed number at initial user creation for now
	// but in case of user modification (before != nil) it sends confirmed number, catch it here.
	if user.PhoneNumber != "" && user.Before != nil && user.Before.PhoneNumber == "" {
		err := u.r.IncrementUserLevel(ctx, user.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to increment user's level for the phone number confirmation userID:%v", user.ID)
		}
	}

	return nil
}
