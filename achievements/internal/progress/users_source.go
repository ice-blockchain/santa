// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
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
		return errors.Wrapf(err, "achievements/userSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}

	if user.User == nil && user.Before != nil {
		return errors.Wrapf(u.handleUserDeletion(ctx, user), "failed to handle user deletion event")
	}

	return errors.Wrap(u.handleUserCreateOrUpdate(ctx, user), "failed to handle user creation/modification event")
}

func (u *userSource) handleUserDeletion(ctx context.Context, user *users.UserSnapshot) error {
	if err := u.r.incrementOrDecrementTotalUsersCount(-1); err != nil {
		return errors.Wrapf(err, "failed to update total_users counter")
	}
	if user.Before.ReferredBy != "" {
		if err := u.r.updateT1ReferralsCount(ctx, user.Before.ReferredBy, -1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}
	if err := u.r.deleteUserProgress(user.Before.ID); err != nil {
		return errors.Wrapf(err, "failed to deleteUserAchievements")
	}

	return nil
}

func (u *userSource) handleUserCreateOrUpdate(ctx context.Context, user *users.UserSnapshot) error {
	_, err := u.r.getUserProgress(user.ID)
	if errors.Is(err, storage.ErrNotFound) {
		if err = u.handleUserCreation(ctx, user); err != nil {
			return errors.Wrapf(err, "failed to handle user creation userID:%v", user.ID)
		}
	}

	return errors.Wrapf(u.handleUserModification(ctx, user), "failed to handle user moditication userID: %v", user.ID)
}

func (u *userSource) handleUserCreation(ctx context.Context, user *users.UserSnapshot) error {
	// User's achievements record does not exists, so it is a new user - increment counter.
	if err := u.r.incrementOrDecrementTotalUsersCount(1); err != nil {
		return errors.Wrapf(err, "failed to update total_users counter")
	}
	// And insert a new record into user_progress for a new user.

	return errors.Wrapf(u.r.insertUserProgress(ctx, user.User), "failed to insert user progress record")
}

func (u *userSource) handleUserModification(ctx context.Context, user *users.UserSnapshot) error {
	// In case of modification - update agenda hashes.
	if user.Before != nil && user.AgendaPhoneNumberHashes != "" {
		if err := u.r.updateAgendaPhoneNumbersHashes(ctx, user.ID, user.AgendaPhoneNumberHashes); err != nil {
			return errors.Wrapf(err, "progress/userSource: failed to update agenda phone number hashes")
		}
	}

	// Next we need to check if we need to update T1 referrals count (userID = referredBy, count +=1).
	if user.ReferredBy != "" {
		if err := u.updateUserReferrals(ctx, user.User); err != nil {
			return errors.Wrapf(err, "failed to update user's referrals for userID:%v (referredBy=%v)", user.ID, user.ReferredBy)
		}
	}

	return nil
}

func (u *userSource) updateUserReferrals(ctx context.Context, user *users.User) error {
	if user.PhoneNumberHash != "" {
		if err := u.checkAndUpdateAgendaReferrals(ctx, user.ReferredBy, user); err != nil {
			return errors.Wrapf(err, "progress/userSource: failed to update agenda referrals")
		}
	}

	return errors.Wrapf(u.r.updateT1ReferralsCount(ctx, user.ReferredBy, 1), "progress/userSource: failed to update t1 referrals counter")
}

func (u *userSource) checkAndUpdateAgendaReferrals(ctx context.Context, referredByID users.UserID, user *users.User) error {
	// Check here if user's phone number is in referredBy's agenda.
	referredByUser, err := u.r.getUserProgress(referredByID)
	if err != nil {
		return errors.Wrapf(err, "failed to read referedBy user's progress (%v) for agenda update (%v user added)", referredByID, user.ID)
	}
	if referredByUser.AgendaPhoneNumbersHashes == "" {
		return nil
	}
	if !strings.Contains(referredByUser.AgendaPhoneNumbersHashes, user.PhoneNumberHash) {
		// Maybe the user removed that contact from agenda.
		return errors.Wrapf(u.r.deleteAgendaReferrals(ctx, referredByUser.UserID, user.ID),
			"failed to delete agenda referrals for userID:%v (failed to remove user %v)", referredByID, user.ID)
	}

	return errors.Wrapf(u.r.insertAgendaReferrals(ctx, referredByUser.UserID, user.ID),
		"failed to update agenda referrals for userID:%v (failed to add user %v)", referredByID, user.ID)
}
