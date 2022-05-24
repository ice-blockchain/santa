// SPDX-License-Identifier: BUSL-1.1

package global

import (
	"context"
	"encoding/json"
	"github.com/ice-blockchain/santa/achievements/internal/storages/badges"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func NewTotalBadgesProcessor(db tarantool.Connector) messagebroker.Processor {
	return &totalBadgesSource{db: db}
}

func (b *totalBadgesSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	achievedBadge := new(badges.AchievedBadgeMessage)
	if err := json.Unmarshal(message.Value, achievedBadge); err != nil {
		return errors.Wrapf(err, "totalBadgesSource: cannot unmarshall %v into %#v", string(message.Value), achievedBadge)
	}
	return errors.Wrapf(b.updateTotalBadgesCount(achievedBadge.Name, 1), "totalBadgesSource: failed to update total badge count for badge:%v", achievedBadge.Name)
}

func (b *totalBadgesSource) updateTotalBadgesCount(badgeName string, diff int64) error {
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
