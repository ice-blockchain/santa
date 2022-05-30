// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/levels"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewLevelProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &levelSource{r: newRepository(db, mb)}
}

func (l *levelSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userLevel := new(levels.AchievedLevel)
	if err := json.Unmarshal(message.Value, userLevel); err != nil {
		return errors.Wrapf(err, "badges/levelSource: cannot unmarshall %v into %#v", string(message.Value), userLevel)
	}

	return errors.Wrapf(l.achieveBadgesForUserLevel(ctx, userLevel),
		"badges/levelSource: failed to achieve completed user's badges for userID:%v", userLevel.UserID)
}

func (l *levelSource) achieveBadgesForUserLevel(ctx context.Context, userLevel *levels.AchievedLevel) error {
	allBadges, err := l.r.getUnachievedBadges(ctx, userLevel.UserID)
	if err != nil {
		return errors.Wrapf(err, "badges/levelSource: failed to fetch all badges")
	}
	for _, badge := range allBadges {
		var criteriaValue uint64
		switch badge.Type {
		case BadgeTypeSocial:
			continue // In progress_source.
		case BadgeTypeIce:
			continue // In progress_source.
		case BadgeTypeLevel:
			criteriaValue = userLevel.TotalLevels
		}
		if err = l.checkAndAchieveBadge(ctx, userLevel.UserID, badge, criteriaValue); err != nil {
			return errors.Wrapf(err, "failed to check if badge %v achieved for user %v", badge.Name, userLevel.UserID)
		}
	}

	return nil
}

func (l *levelSource) checkAndAchieveBadge(ctx context.Context, userID UserID, badge *Badge, criteriaValue uint64) error {
	if criteriaValue >= badge.Interval.Left && criteriaValue <= badge.Interval.Right {
		if err := l.r.achieveBadge(ctx, userID, badge.Name); err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err, "failed to achieve badge %v to userID %v", badge.Name, userID)
		}
	}

	return nil
}
