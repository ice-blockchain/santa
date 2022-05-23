// SPDX-License-Identifier: BUSL-1.1

package economyprocessor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func NewMiningEventProcessor(db tarantool.Connector, repository WriteRepository) messagebroker.Processor {
	return &miningEventSourceProcessor{
		db: db,
		r:  repository,
	}
}

func (m *miningEventSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	miningEvent := new(economy.MiningStarted)
	userID := message.Key
	if err := json.Unmarshal(message.Value, miningEvent); err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot unmarshal %v into %#v", string(message.Value), miningEvent)
	}
	updatedSessions, err := m.handleMiningSessionStart(userID, miningEvent.TS)
	if err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}
	if err = m.achieveTaskAndLevels(ctx, updatedSessions); err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

func (m *miningEventSourceProcessor) updateConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS}, // | last_mining_started_at = lastStartedTS.
		{Op: "+", Field: maxCountField, Arg: 1},                        // | max_count +=1.
	}

	return errors.Wrapf(m.db.UpdateTyped(consecutiveUserMiningSessionsSpace, "pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1",
		key, ops, &[]*consecutiveUserMiningSessions{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", consecutiveUserMiningSessionsSpace, userID)
}

func (m *miningEventSourceProcessor) resetConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS}, // | last_mining_started_at = lastStartedTS.
		{Op: "=", Field: maxCountField, Arg: 1},                        // | max_count = 1 (lastest session just started).
	}

	return errors.Wrapf(m.db.UpdateTyped(consecutiveUserMiningSessionsSpace, "pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1",
		key, ops, &[]*consecutiveUserMiningSessions{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", consecutiveUserMiningSessionsSpace, userID)
}

func (m *miningEventSourceProcessor) insertConsecutiveMiningSessions(userID UserID, lastStartedTS uint64) error {
	ua := &consecutiveUserMiningSessions{
		UserID:              userID,
		LastMiningStartedAt: lastStartedTS,
		MaxCount:            1, // Initial session = 1 .
	}

	return errors.Wrapf(m.db.InsertTyped(consecutiveUserMiningSessionsSpace, ua, &[]*consecutiveUserMiningSessions{}),
		"failed to insert consecutive user mining sessions for userID:%v", userID)
}

func (m *miningEventSourceProcessor) getConsecutiveMiningSessions(userID UserID) (*consecutiveUserMiningSessions, error) {
	var res consecutiveUserMiningSessions
	if err := m.db.GetTyped(consecutiveUserMiningSessionsSpace,
		"pk_unnamed_CONSECUTIVE_USER_MINING_SESSIONS_1", tarantool.StringKey{S: userID}, &res); err != nil {
		return nil, errors.Wrapf(err, "unable to get consecutive user mining sessions record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no consecutive user mining sessions record for userID:%v", userID)
	}

	return &res, nil
}

func (m *miningEventSourceProcessor) handleMiningSessionStart(userID UserID, lastStartedTS time.Time) (*consecutiveUserMiningSessions, error) {
	userMiningSessions, err := m.getConsecutiveMiningSessions(userID)
	timeStartedNano := uint64(lastStartedTS.UTC().UnixNano())
	if errors.Is(err, storage.ErrNotFound) {
		// User's mining sessions not found - so it is his first session.
		if err = m.insertConsecutiveMiningSessions(userID, timeStartedNano); err != nil {
			return nil, errors.Wrapf(err, "failed to insert user achievements record")
		}

		return &consecutiveUserMiningSessions{
			UserID:              userID,
			LastMiningStartedAt: timeStartedNano,
			MaxCount:            1,
		}, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "miningEventSourceProcessor: failed handle MiningStarted message")
	}
	// Session count found - update it.
	return m.handleConsecutiveSessionsUpdate(userMiningSessions, userID, timeStartedNano)
}

func (m *miningEventSourceProcessor) handleConsecutiveSessionsUpdate(
	userMiningSessions *consecutiveUserMiningSessions,
	userID UserID,
	timeStartedNano uint64,
) (*consecutiveUserMiningSessions, error) {
	// First we need to check if 10 hours passed from the previous session, if passed - reset counter to 0.
	if time.Since(time.Unix(0, int64(userMiningSessions.LastMiningStartedAt))) >= maxTimeBetweenConsecutiveMiningSessions {
		if err := m.resetConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
			return nil, errors.Wrapf(err, "failed to reset consecutive mining sessions for userID:%v", userID)
		}

		return &consecutiveUserMiningSessions{
			UserID:              userID,
			LastMiningStartedAt: timeStartedNano,
			MaxCount:            1,
		}, nil
	}
	// And if not - increment counter of consecutive sessions.
	if err := m.updateConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
		return nil, errors.Wrapf(err, "failed to update consecutive mining sessions for userID:%v", userID)
	}

	return &consecutiveUserMiningSessions{
		UserID:              userID,
		LastMiningStartedAt: timeStartedNano,
		MaxCount:            userMiningSessions.MaxCount + 1,
	}, nil
}

func (m *miningEventSourceProcessor) achieveTaskAndLevels(ctx context.Context, sessions *consecutiveUserMiningSessions) error {
	//nolint:godot,nolintlint // FIXME: handle decrement in case when user spends >10h and continie mining,
	// counter resets to 0, and we'll be here again with the same value.
	switch sessions.MaxCount {
	case 90, 60, 30, 10, 5: //nolint:gomnd,nolintlint // Consecutive mining sessions increments level (Levels -> #2-6).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to ")
		}
	case 1: //nolint:gomnd,nolintlint // First mining - increment user's level (Levels -> #1) and achieve task (Tasks #2).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to first mining session for userID:%v", sessions.UserID)
		}
		if err := m.r.AchieveTask(ctx, sessions.UserID, "TASK2"); err != nil {
			return errors.Wrapf(err, "failed to achieve task due to first mining session for userID:%v", sessions.UserID)
		}
	}

	return nil
}
