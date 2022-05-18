package achievementprocessor

import (
	"context"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewTaskProcessor(db tarantool.Connector) messagebroker.Processor {
	return &taskSourceProcessor{db: db}
}

func (u *taskSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	// We'll need to pass newly achieved tasks to the mb producer, get them here and
	// increment user's level here (each task += 1 level)
	return errors.Errorf("Implement me")
}
