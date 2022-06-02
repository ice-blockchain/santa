// SPDX-License-Identifier: BUSL-1.1

package tasks

import (
	"context"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

func NewEconomyMiningProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &economyMiningSource{
		r: NewRepository(db, mb),
	}
}

func (m *economyMiningSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	err := m.r.CompleteTask(ctx, userID, taskFirstMiningSession)
	if err != nil && !errors.Is(err, errAlreadyAchieved) {
		return errors.Wrapf(err, "tasks/economyMiningSource: Failed to achieve task for the first mining session for userID:%v", userID)
	}

	return nil
}
