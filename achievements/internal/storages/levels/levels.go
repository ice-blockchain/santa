// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	return &repository{db: db, mb: mb}
}

func (r *repository) achieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve level failed because context failed")
	}
	now := time.Now()
	sql := `INSERT INTO achieved_user_levels(USER_ID, LEVEL_NAME,   ACHIEVED_AT)
                                                   VALUES(:userID,  :levelName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"levelName":  levelName,
		"achievedAt": uint64(now.UTC().UnixNano()),
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's level %v for userID:%v", levelName, userID)
	}

	return errors.Wrapf(r.incrementCurrentLevel(ctx, userID, levelName, now), "failed to increment user's level %v for userID:%v", levelName, userID)
}

func (r *repository) insertCurrentUserLevel(ctx context.Context, userID UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "failed to get insert agenda referrals because of context failed")
	}
	lvl := &currentUserLevels{
		UserID:    userID,
		Level:     0,
		UpdatedAt: 0,
	}
	if err := r.db.InsertTyped(currentUsersLevelsSpace, lvl, &[]*currentUserLevels{}); err != nil && !errors.Is(err, storage.ErrDuplicate) {
		return errors.Wrapf(err, "failed to current user level record for user.ID:%v", userID)
	}

	return nil
}

func (r *repository) incrementCurrentLevel(ctx context.Context, userID UserID, achievedLevel LevelName, achievedAt time.Time) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "increment level failed because context failed")
	}
	now := uint64(achievedAt.UTC().UnixNano())
	key := tarantool.StringKey{S: userID}
	incrementOps := []tarantool.Op{
		{Op: "+", Field: fieldCurrentUserLevelsLevel, Arg: 1},
		{Op: "=", Field: fieldCurrentUserLevelsUpdatedAt, Arg: now},
	}

	res := []*currentUserLevels{}
	if err := r.db.UpdateTyped(currentUsersLevelsSpace, "pk_unnamed_CURRENT_USER_LEVELS_1", key, incrementOps, &res); err != nil {
		return errors.Wrapf(err, "failed to update current user level (%v) for user %v", achievedLevel, userID)
	}

	return errors.Wrapf(r.sendAchievedLevel(ctx, userID, achievedLevel, achievedAt, res[0].Level),
		"failed to send incremented level to message broker: %v for userID:%v", achievedLevel, userID)
}

func (r *repository) sendAchievedLevel(ctx context.Context, userID UserID, achievedLevel LevelName, achievedAt time.Time, totalLevels uint64) error {
	m := AchievedLevel{
		UserID:      userID,
		LevelName:   achievedLevel,
		AchievedAt:  achievedAt,
		TotalLevels: totalLevels,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-level] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   cfg.MessageBroker.Topics[4].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-level] failed to send message to broker")
}
