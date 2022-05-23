package achievements

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func (r *repository) UpdateConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS}, // | last_mining_started_at = lastStartedTS.
		{Op: "+", Field: maxCountField, Arg: 1},                        // | max_count +=1.
	}

	return errors.Wrapf(r.db.UpdateTyped(consecutiveUserMiningSessionsSpace, "pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1",
		key, ops, &[]*consecutiveUserMiningSessions{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", consecutiveUserMiningSessionsSpace, userID)
}

func (r *repository) ResetConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS}, // | last_mining_started_at = lastStartedTS.
		{Op: "=", Field: maxCountField, Arg: 1},                        // | max_count = 1 (lastest session just started).
	}

	return errors.Wrapf(r.db.UpdateTyped(consecutiveUserMiningSessionsSpace, "pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1",
		key, ops, &[]*consecutiveUserMiningSessions{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", consecutiveUserMiningSessionsSpace, userID)
}

func (r *repository) InsertConsecutiveMiningSessions(userID UserID, lastStartedTS uint64) error {
	ua := &consecutiveUserMiningSessions{
		UserID:              userID,
		LastMiningStartedAt: lastStartedTS,
		MaxCount:            1, // Initial session = 1 .
	}

	return errors.Wrapf(r.db.InsertTyped(consecutiveUserMiningSessionsSpace, ua, &[]*consecutiveUserMiningSessions{}),
		"failed to insert consecutive user mining sessions for userID:%v", userID)
}

func (r *repository) GetConsecutiveMiningSessions(userID UserID) (*ConsecutiveMiningSessions, error) {
	var res consecutiveUserMiningSessions
	if err := r.db.GetTyped(consecutiveUserMiningSessionsSpace,
		"pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1", tarantool.StringKey{S: userID}, &res); err != nil {
		return nil, errors.Wrapf(err, "unable to get consecutive user mining sessions record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no consecutive user mining sessions record for userID:%v", userID)
	}

	return res.ConsecutiveMiningSessions(), nil
}

func (c *consecutiveUserMiningSessions) ConsecutiveMiningSessions() *ConsecutiveMiningSessions {
	return &ConsecutiveMiningSessions{
		UserID:              c.UserID,
		LastMiningStartedAt: c.LastMiningStartedAt,
		MaxCount:            c.MaxCount,
	}
}
