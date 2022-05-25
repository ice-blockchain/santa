// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"strings"
)

func New(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &userSource{
		r: newRepository(db, mb),
	}
}

func (u *userSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "achievements/userSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}

	// User deletion, we need to handle it, update total_users in GLOBAL and delete him from USER_ACHIEVEMENTS
	// and decrement t1 referrals count for its parent if user was referred by another user.
	if user.User == nil && user.Before != nil {
		if err := u.handleUserDeletion(ctx, user); err != nil {
			return errors.Wrapf(err, "failed to handle user deletion event")
		}

		return nil // We're complete here with deletion.
	}
	if err := u.handleUserCreation(ctx, user); err != nil {
		return errors.Wrap(err, "failed to handle user creation/modification event")
	}

	return nil
}

func (u *userSource) handleUserDeletion(ctx context.Context, user *users.UserSnapshot) error {
	if err := u.r.DeleteUserProgress(user.Before.ID); err != nil {
		return errors.Wrapf(err, "failed to deleteUserAchievements")
	}
	if err := u.r.UpdateTotalUsersCount(-1); err != nil {
		return errors.Wrapf(err, "failed to update total_users counter")
	}

	if user.Before.ReferredBy != "" {
		if err := u.r.UpdateT1ReferralsCount(ctx, user.Before.ReferredBy, -1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}

	return nil
}

func (u *userSource) handleUserCreation(ctx context.Context, user *users.UserSnapshot) error {
	_, err := u.r.GetUserProgress(user.ID)
	if errors.Is(err, storage.ErrNotFound) {
		// User's achievements record does not exists, so it is a new user - increment counter.
		if err = u.r.UpdateTotalUsersCount(1); err != nil {
			return errors.Wrapf(err, "failed to update total_users counter")
		}
		if err = u.r.InsertUserProgress(ctx, user.User); err != nil {
			return errors.Wrapf(err, "failed to insert user achievements record")
		}
	}
	// Next we need to check if we need to update T1 referrals count (userID = referredBy, count +=1).
	if user.ReferredBy != "" {
		if err = u.r.UpdateT1ReferralsCount(ctx, user.ReferredBy, 1); err != nil {
			return errors.Wrapf(err, "progress/userSource: failed to update t1 referrals counter")
		}
		if err = u.checkAndUpdateAgendaReferrals(user.ReferredBy, user.User); err != nil {
			return errors.Wrapf(err, "progress/userSource: failed to update agenda referrals")
		}
	}
	if user.Before != nil && user.AgendaPhoneNumberHashes != "" { // In case of modification - update agenda hashes.
		if err = u.r.UpdateAgendaPhoneNumbersHashes(ctx, user.ID, user.AgendaPhoneNumberHashes); err != nil {
			return errors.Wrapf(err, "progress/userSource: failed to update agenda phone number hashes")
		}
	}
	return nil
}

func (u *userSource) checkAndUpdateAgendaReferrals(referredByID users.UserID, user *users.User) error {
	// Check here if user's phone number is in referredBy's agenda.
	referredByUser, err := u.r.GetUserProgress(referredByID)
	if err != nil {
		return errors.Wrapf(err, "failed to read referedBy user's progress (%v) for user %v", referredByID, user.ID)
	}
	if strings.Contains(referredByUser.AgendaPhoneNumbersHashes, user.PhoneNumberHash) {
		// update agenda_referrals
	}

	return nil
}
