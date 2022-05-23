// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/santa/achievements/internal"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func New(db tarantool.Connector) internal.UserSource {
	return &userSource{
		r: &repository{db: db},
	}
}

func (u *userSource) ProcessUser(ctx context.Context, user *users.UserSnapshot) error {
	// User deletion, we need to handle it, update total_users in GLOBAL and delete him from USER_ACHIEVEMENTS
	// and decrement t1 referrals count for its parent if user was referred by another user.
	if user.User == nil && user.Before != nil {
		if err := u.handleUserDeletion(user); err != nil {
			return errors.Wrapf(err, "failed to handle user deletion event")
		}

		return nil // We're complete here with deletion.
	}
	if err := u.handleUserCreation(user); err != nil {
		return errors.Wrap(err, "failed to handle user creation/modification event")
	}

	// MOVE to tasks and levels packages
	//if err := u.achieveTaskAndLevels(ctx, user); err != nil {
	//	return errors.Wrapf(err, "failed to achieve task && levels on user message %#v", user)
	//}

	return nil
}

func (u *userSource) handleUserDeletion(user *users.UserSnapshot) error {
	if err := u.r.DeleteUserAchievements(user.Before.ID); err != nil {
		return errors.Wrapf(err, "failed to deleteUserAchievements")
	}
	if err := u.r.UpdateTotalUsersCount(-1); err != nil {
		return errors.Wrapf(err, "failed to update total_users counter")
	}

	if user.Before.ReferredBy != "" {
		if err := u.r.UpdateT1ReferralsCount(user.Before.ReferredBy, -1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}

	return nil
}

func (u *userSource) handleUserCreation(user *users.UserSnapshot) error {
	_, err := u.r.GetUserAchievements(user.ID)
	if errors.Is(err, storage.ErrNotFound) {
		// User's achievements record does not exists, so it is a new user - increment counter.
		if err = u.r.UpdateTotalUsersCount(1); err != nil {
			return errors.Wrapf(err, "failed to update total_users counter")
		}
		if err = u.r.InsertUserAchievements(user.ID); err != nil {
			return errors.Wrapf(err, "failed to insert user achievements record")
		}
	}
	// Next we need to check if we need to update T1 referrals count (userID = referredBy, count +=1).
	if user.ReferredBy != "" {
		if err = u.r.UpdateT1ReferralsCount(user.ReferredBy, 1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}

	return nil
}
