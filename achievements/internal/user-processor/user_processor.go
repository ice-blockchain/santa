// SPDX-License-Identifier: BUSL-1.1

package userprocessor

import (
	"context"
	"encoding/json"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func New(db tarantool.Connector, repository WriteRepository) messagebroker.Processor {
	return &userSourceProcessor{
		db: db,
		r:  repository,
	}
}

func (u *userSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	// Process user messages here (from eskimo).
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "userSourceProcessor: cannot unmarshal %v into %#v", string(message.Value), user)
	}
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
	if err := u.achieveTaskAndLevels(ctx, user); err != nil {
		return errors.Wrapf(err, "failed to achieve task && levels on user message %#v", user)
	}

	return nil
}

func (u *userSourceProcessor) handleUserDeletion(user *users.UserSnapshot) error {
	if err := u.deleteUserAchievements(user.Before.ID); err != nil {
		return errors.Wrapf(err, "failed to deleteUserAchievements")
	}
	if err := u.updateTotalUsersCount(-1); err != nil {
		return errors.Wrapf(err, "failed to update total_users counter")
	}

	if user.Before.ReferredBy != "" {
		if err := u.updateT1ReferralsCount(user.Before.ReferredBy, -1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}

	return nil
}

func (u *userSourceProcessor) handleUserCreation(user *users.UserSnapshot) error {
	_, err := u.getUserAchievements(user.ID)
	if errors.Is(err, storage.ErrNotFound) {
		// User's achievements record does not exists, so it is a new user - increment counter.
		if err = u.updateTotalUsersCount(1); err != nil {
			return errors.Wrapf(err, "failed to update total_users counter")
		}
		if err = u.insertUserAchievements(user.ID); err != nil {
			return errors.Wrapf(err, "failed to insert user achievements record")
		}
	}
	// Next we need to check if we need to update T1 referrals count (userID = referredBy, count +=1).
	if user.ReferredBy != "" {
		if err = u.updateT1ReferralsCount(user.ReferredBy, 1); err != nil {
			return errors.Wrapf(err, "failed to update t1 referrals counter")
		}
	}

	return nil
}

func (u *userSourceProcessor) deleteUserAchievements(userID users.UserID) error {
	sql := `DELETE FROM user_achievements WHERE user_id = :userID`
	params := map[string]interface{}{
		"userID": userID,
	}

	return errors.Wrapf(storage.CheckSQLDMLErr(u.db.PrepareExecute(sql, params)),
		"failed to delete user achievements record for user.ID:%v", userID)
}

func (u *userSourceProcessor) updateTotalUsersCount(diff int64) error {
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

func (u *userSourceProcessor) getUserAchievements(userID users.UserID) (*userAchievements, error) {
	res := new(userAchievements)
	if err := u.db.GetTyped(userAchievementsSpace, "pk_unnamed_USER_ACHIEVEMENTS_1", tarantool.StringKey{S: userID}, res); err != nil {
		return nil, errors.Wrapf(err, "unable to get user_achievements record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no user achievements record for userID:%v", userID)
	}

	return res, nil
}

func (u *userSourceProcessor) insertUserAchievements(userID users.UserID) error {
	ua := &userAchievements{
		UserID: userID,
		// Other fields have DEFAULT values on DDL.
	}

	return errors.Wrapf(u.db.InsertTyped(userAchievementsSpace, ua, &[]*userAchievements{}),
		"failed to insert user achievements record for user.ID:%v", userID)
}

func (u *userSourceProcessor) updateT1ReferralsCount(userID users.UserID, diff int64) error {
	key := tarantool.StringKey{S: userID}
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldT1Referrals, Arg: diff},
	}

	return errors.Wrapf(u.db.UpdateTyped(userAchievementsSpace, "pk_unnamed_USER_ACHIEVEMENTS_1", key, incrementOps, &[]*userAchievements{}),
		"failed to update %v record with the new count of T1 referals for userID:%v", userAchievementsSpace, userID)
}

func (u *userSourceProcessor) achieveTaskAndLevels(ctx context.Context, user *users.UserSnapshot) error {
	userAchievementState, err := u.getUserAchievements(user.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to get user state to achieve tasks")
	}
	achievedTask := u.getCompletedTask(user, userAchievementState)
	//nolint:godox,nolintlint // TODO: think about how to achieve social sharing (endpoint call after sharing?), join twitter, etc.
	if achievedTask != "" {
		err = u.r.AchieveTask(ctx, user.ID, achievedTask)
		if err != nil {
			return errors.Wrapf(err, "failed to achieve task %#v for userID:%v", achievedTask, user.ID)
		}
	}

	return errors.Wrapf(u.achieveLevels(ctx, user), "failed to achieve user's level")
}

func (u *userSourceProcessor) achieveLevels(ctx context.Context, user *users.UserSnapshot) error {
	// New level for user - 8. Confirm phone number
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
