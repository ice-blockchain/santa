package user_processor

import (
	"context"
	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func New(db tarantool.Connector) messagebroker.Processor {
	return &userSourceProcessor{db: db}
}

func (u *userSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	//user messages will be processed here ( user creation / deletion)
	return errors.Errorf("Implement me")
}
