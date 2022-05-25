package progress

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewEconomyMiningSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &economyMiningSource{
		r: newRepository(db, mb),
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

	err := e.handleConsecutiveSessionsUpdate(ctx, userID, uint64(miningEvent.TS.UTC().UnixNano()))
	if err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

func (e *economyMiningSource) handleConsecutiveSessionsUpdate(
	ctx context.Context,
	userID UserID,
	timeStartedNano uint64,
) error {
	userProgress, err := e.r.GetUserProgress(userID)
	if err != nil {
		return errors.Wrapf(err, "progress/economyMiningSource: cannot get user progress for userID:%v", userID)
	}
	// First we need to check if 10 hours passed from the previous session, if passed - reset counter to 0.
	if time.Since(time.Unix(0, int64(userProgress.LastMiningStartedAt))) >= maxTimeBetweenConsecutiveMiningSessions {
		if err := e.r.ResetConsecutiveMiningSessionsCount(ctx, userID, timeStartedNano); err != nil {
			return errors.Wrapf(err, "failed to reset consecutive mining sessions for userID:%v", userID)
		}

		return nil
	}
	// And if not - increment counter of consecutive sessions.
	if err := e.r.UpdateConsecutiveMiningSessionsCount(ctx, userID, timeStartedNano); err != nil {
		return errors.Wrapf(err, "failed to update consecutive mining sessions for userID:%v", userID)
	}

	return nil
}
