package economy_processor

import (
	"context"
	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func New(db tarantool.Connector) messagebroker.Processor {
	return &economySourceProcessor{db: db}
}

func (u *economySourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	// start mining message will be processed here
	return errors.Errorf("Implement me")
}
