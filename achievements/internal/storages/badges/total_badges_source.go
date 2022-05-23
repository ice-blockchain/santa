// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"github.com/ice-blockchain/santa/achievements/internal"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func NewBadgeProcessor(db tarantool.Connector) internal.AchievedBadgesSource {
	return &totalBadgesSource{db: db}
}

func (b *totalBadgesSource) ProcessAchievedBadge(ctx context.Context, userId UserID, achievedBadge *AchievedBadgeMessage) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}

	return errors.Wrapf(b.updateTotalBadgesCount(achievedBadge.Name, 1), "totalBadgesSource: failed to update total badge count for badge:%#v", achievedBadge.Name)
}

func (b *totalBadgesSource) updateTotalBadgesCount(badgeName BadgeName, diff int64) error {
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
