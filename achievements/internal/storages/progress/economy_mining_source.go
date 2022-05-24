package progress

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
	"time"
)

func NewEcomonyMiningSource(db tarantool.Connector) messagebroker.Processor {
	return &economyMiningSource{
		r: &repository{db: db},
	}
}

func (e *economyMiningSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	miningEvent := new(economy.MiningStarted)
	if err := json.Unmarshal(message.Value, miningEvent); err != nil {
		return errors.Wrapf(err, "achievements/economyMiningSource: cannot unmarshall %v into %#v", string(message.Value), miningEvent)
	}

	err := e.handleMiningSessionStart(userID, miningEvent.TS)
	if err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}
	return nil
}

func (m *economyMiningSource) handleMiningSessionStart(userID UserID, lastStartedTS time.Time) error {
	userProgress, err := m.r.GetUserProgress(userID)
	timeStartedNano := uint64(lastStartedTS.UTC().UnixNano())
	if err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: failed handle MiningStarted message")
	}
	// Session count found - update it.
	return m.handleConsecutiveSessionsUpdate(userProgress, userID, timeStartedNano)
}

func (m *economyMiningSource) handleConsecutiveSessionsUpdate(
	userProgress *UserProgress,
	userID UserID,
	timeStartedNano uint64,
) error {
	// First we need to check if 10 hours passed from the previous session, if passed - reset counter to 0.
	if time.Since(time.Unix(0, int64(userProgress.LastMiningStartedAt))) >= maxTimeBetweenConsecutiveMiningSessions {
		if err := m.r.ResetConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
			return errors.Wrapf(err, "failed to reset consecutive mining sessions for userID:%v", userID)
		}

		return nil
	}
	// And if not - increment counter of consecutive sessions.
	if err := m.r.UpdateConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
		return errors.Wrapf(err, "failed to update consecutive mining sessions for userID:%v", userID)
	}

	return nil
}
