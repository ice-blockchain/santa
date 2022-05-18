package economyprocessor

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
	// Start mining message will be processed here
	// and economy updated messages ( update balance at user_achievements).
	return errors.Errorf("Implement me")
}
