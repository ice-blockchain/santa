package tasks

import (
	"context"
	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewEcomonyMiningSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &economyMiningSource{
		r: New(db, mb),
	}
}

func (m *economyMiningSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	// TODO check if task is not achieved yet
	return errors.Wrapf(m.r.AchieveTask(ctx, userID, "TASK2"), "tasks/economyMiningSource: Failed to achieve task for the first mining session for userID:%v", userID)
}
