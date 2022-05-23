package achievementprocessor

import (
	"context"
	"encoding/json"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/messages"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewBadgeProcessor(db tarantool.Connector) messagebroker.Processor {
	return &badgeSourceProcessor{db: db}
}

func (b *badgeSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	badge := new(messages.AchievedBadgeMessage)
	if err := json.Unmarshal(message.Value, badge); err != nil {
		return errors.Wrapf(err, "badgeSourceProcessor: cannot unmarshal %v into %#v", string(message.Value), badge)
	}

	return errors.Wrapf(b.updateTotalBadgesCount(badge.Name, 1), "badgeSourceProcessor: failed to update total badge count for badge:%#v", badge)
}

func (b *badgeSourceProcessor) updateTotalBadgesCount(badgeName BadgeName, diff int64) error {
	key := "TOTAL_BADGES_" + badgeName
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: 1, Arg: diff},
	}

	return errors.Wrapf(b.db.UpsertAsync("GLOBAL", &global{Key: key, Value: 1}, incrementOps).GetTyped(&[]*global{}),
		"failed to update global record the KEY = '%v'", key)
}
