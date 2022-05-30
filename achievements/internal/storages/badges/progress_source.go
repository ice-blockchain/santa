// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"

	"cosmossdk.io/math"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &progressSource{r: newRepository(db, mb)}
}

func (p *progressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "badges/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}

	return errors.Wrapf(p.achieveBadgesForUserProgress(ctx, userProgress),
		"badges/progressSource: failed to achieve completed user's badges for userID:%v", userProgress.UserID)
}

func (p *progressSource) achieveBadgesForUserProgress(ctx context.Context, userProgress *progress.UserProgress) error {
	allBadges, err := p.r.getUnachievedBadges(ctx, userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "badges/progressSource: failed to fetch all badges")
	}
	for _, badge := range allBadges {
		// Because of 256 bit balance using math.Uint here instead of uint64.
		var criteriaValue math.Uint
		switch badge.Type {
		case BadgeTypeSocial:
			criteriaValue = math.NewUint(userProgress.T1Referrals)
		case BadgeTypeIce:
			criteriaValue = userProgress.Balance.Uint
		case BadgeTypeLevel:
			continue // Level is not stored at userProgress, so process it in level_source.
		}
		if err = p.checkAndAchieveBadge(ctx, userProgress.UserID, badge, criteriaValue); err != nil {
			return errors.Wrapf(err, "failed to check if badge %v achieved for user %v", badge.Name, userProgress.UserID)
		}
	}

	return nil
}

func (p *progressSource) checkAndAchieveBadge(ctx context.Context, userID UserID, badge *Badge, criteriaValue math.Uint) error {
	if criteriaValue.GTE(math.NewUint(badge.Interval.Left)) && criteriaValue.LTE(math.NewUint(badge.Interval.Right)) {
		if err := p.r.achieveBadge(ctx, userID, badge.Name); err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err, "failed to achieve badge %v to userID %v", badge.Name, userID)
		}
	}

	return nil
}
