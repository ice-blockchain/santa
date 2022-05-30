// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewUsersProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &userSource{
		r: newRepository(db, mb).(*repository),
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
	if user.User != nil {
		if user.Before == nil {
			// Insert zero value (level = 0) to update it later when first level will be achieved.
			if err := u.r.insertCurrentUserLevel(ctx, user.ID); err != nil {
				return errors.Wrapf(err, "levels/userSource: failed to insert current user level for userID: %v", user.ID)
			}
		}
		if err := u.achieveLevelForPhoneNumberConfirmation(ctx, user); err != nil {
			return errors.Wrapf(err, "levels/userSource: failed to increment user's level for the phone number confirmation")
		}
	}

	return nil
}

func (u *userSource) achieveLevelForPhoneNumberConfirmation(ctx context.Context, user *users.UserSnapshot) error {
	// New level for user (Levels -> 8 Confirm phone number).
	if user.PhoneNumber != "" && user.Before != nil && user.Before.PhoneNumber == "" {
		err := u.r.achieveUserLevel(ctx, user.ID, levelForPhoneNumberConfirmation)
		if err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err, "failed to increment user's level for the phone number confirmation userID:%v", user.ID)
		}
	}

	return nil
}
