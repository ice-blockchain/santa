package achievementprocessor

import (
	"context"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewBadgeProcessor(db tarantool.Connector) messagebroker.Processor {
	return &badgeSourceProcessor{db: db}
}

func (u *badgeSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	// We'll need to pass newly achieved badges to the mb producer, get them here and update
	// GLOBAL.TOTAL_BADGES_<BADGE_NAME> with counter (like total users).
	return errors.Errorf("Implement me")
}
