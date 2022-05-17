// SPDX-License-Identifier: BUSL-1.1

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

func NewEconomyMiningProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &economyMiningSource{
		r: newRepository(db, mb).(*repository),
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

	userProgress, err := e.r.getUserProgress(userID)
	if err != nil {
		return errors.Wrapf(err, "progress/economyMiningSource: cannot get user progress for userID:%v", userID)
	}
	// First we need to check if 10 hours passed from the previous session, if passed - reset counter to 0
	// and if not - increment counter of consecutive sessions.
	reset := time.Now().UTC().Sub(time.Unix(0, userProgress.LastMiningStartedAt.UTC().UnixNano())) >= maxTimeBetweenConsecutiveMiningSessions

	return errors.Wrapf(e.r.updateConsecutiveMiningSessionsCount(ctx, userID, miningEvent.TS, reset),
		"failed to update consecutive mining sessions for userID:%v", userID)
}
