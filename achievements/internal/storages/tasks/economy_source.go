package tasks

import (
	"context"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewEconomyMiningSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &economyMiningSource{
		r: newRepository(db, mb),
	}
}

func (m *economyMiningSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	err := m.r.AchieveTask(ctx, userID, taskFirstMiningSession)
	if err != nil && !errors.Is(err, ErrAlreadyAchieved) {
		return errors.Wrapf(err, "tasks/economyMiningSource: Failed to achieve task for the first mining session for userID:%v", userID)
	}

	return nil
}
