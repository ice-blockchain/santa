// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	return &repository{db: db, mb: mb}
}

func achieveLevelAndHandleError(
	ctx context.Context,
	r *repository,
	userID UserID,
	levelName LevelName,
) error {
	if err := r.achieveUserLevel(ctx, userID, levelName); err != nil && !errors.Is(err, errAlreadyAchieved) {
		return errors.Wrapf(err, "failed to achieve users level %v for userID:%v", levelName, userID)
	}

	return nil
}

func unachieveLevelAndHandleError(
	ctx context.Context,
	r *repository,
	userID UserID,
	levelName LevelName,
) error {
	if err := r.unachieveUserLevel(ctx, userID, levelName); err != nil && !errors.Is(err, errNotAchieved) {
		return errors.Wrapf(err, "failed to achieve users level %v for userID:%v", levelName, userID)
	}

	return nil
}

func (r *repository) achieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve level failed because context failed")
	}
	now := time.Now()
	lvl := &achievedUserLevel{
		UpdatedAt: now,
		UserID:    userID,
		LevelName: levelName,
	}
	if err := r.db.InsertTyped(achievedUserLevelsSpace, lvl, &[]*achievedUserLevel{}); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_FOUND {
			return errors.Wrapf(errAlreadyAchieved, "level %v already achieved for userID %v", levelName, userID)
		}

		return errors.Wrapf(err, "failed to insert achieved user level %v user.ID:%v", levelName, userID)
	}

	return errors.Wrapf(r.incrementOrDecrementCurrentLevel(ctx, userID, levelName, now, 1), "failed to increment user's level %v for userID:%v", levelName, userID)
}

func (r *repository) unachieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve level failed because context failed")
	}
	now := time.Now()
	key := &achievedUserLevelKey{ // Cannot use achievedUserLevel here because of 3 fields (2 fields in key).
		UserID:    userID,
		LevelName: levelName,
	}
	res := []*achievedUserLevel{}
	if err := r.db.DeleteTyped(achievedUserLevelsSpace, "pk_unnamed_ACHIEVED_USER_LEVELS_1", key, &res); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_NOT_FOUND {
			return errors.Wrapf(errNotAchieved, "level %v unachieved for userID %v yet", levelName, userID)
		}

		return errors.Wrapf(err, "failed to delete achieved record for user.ID:%v", userID)
	}

	return errors.Wrapf(r.incrementOrDecrementCurrentLevel(ctx, userID, levelName, now, -1),
		"failed to decrement user's level %v for userID:%v", levelName, userID)
}

func (r *repository) insertCurrentUserLevel(ctx context.Context, userID UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed to get insert agenda referrals because of context failed")
	}
	lvl := &currentUserLevels{
		UserID: userID,
		Level:  0,
	}
	if err := r.db.InsertTyped(currentUsersLevelsSpace, lvl, &[]*currentUserLevels{}); err != nil && !errors.Is(err, storage.ErrDuplicate) {
		return errors.Wrapf(err, "failed to insert current user level record for user.ID:%v", userID)
	}

	return nil
}

func (r *repository) incrementOrDecrementCurrentLevel(ctx context.Context, userID UserID, achievedLevel LevelName, achievedAt *time.Time, diff int64) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "increment level failed because context failed")
	}
	key := tarantool.StringKey{S: userID}
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldCurrentUserLevelsLevel, Arg: 1},
		{Op: "=", Field: fieldCurrentUserLevelsUpdatedAt, Arg: achievedAt},
	}

	res := []*currentUserLevels{}
	// Updated here (not upsert) to get result with new value, upsert does not return anything.
	if err := r.db.UpdateTyped(currentUsersLevelsSpace, "pk_unnamed_CURRENT_USER_LEVELS_1", key, incrementOps, &res); err != nil {
		return errors.Wrapf(err, "failed to update current user level (%v) for user %v", achievedLevel, userID)
	}

	return errors.Wrapf(r.sendAchievedLevel(ctx, &AchievedLevel{
		AchievedAt:  achievedAt,
		UserID:      userID,
		LevelName:   achievedLevel,
		TotalLevels: res[0].Level,
		Unachieved:  op == "-",
	}),
		"failed to send incremented level to message broker: %v for userID:%v", achievedLevel, userID)
}

func (r *repository) sendAchievedLevel(ctx context.Context, level *AchievedLevel) error {
	b, err := json.Marshal(level)
	if err != nil {
		return errors.Wrapf(err, "[achieve-level] failed to marshal %#v", level)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     level.UserID,
		Topic:   cfg.MessageBroker.Topics[4].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-level] failed to send message to broker")
}

func (r *repository) getUserLevels(ctx context.Context, userID UserID) ([]*levelWithAchieved, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "fetching achieved levels failed because context failed")
	}

	sql := `
select
    LEVELS.NAME,
    ACHIEVED_USER_LEVELS.ACHIEVED_AT is not NULL as achieved
from LEVELS
         left join ACHIEVED_USER_LEVELS
                   on LEVELS.NAME = ACHIEVED_USER_LEVELS.LEVEL_NAME and ACHIEVED_USER_LEVELS.USER_ID = :userID
`
	params := map[string]interface{}{
		"userID": userID,
	}
	queryResult := []*levelWithAchieved{}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil {
		return nil, errors.Wrapf(err, "failed to get already achieved levels for userID:%v", userID)
	}

	return queryResult, nil
}
