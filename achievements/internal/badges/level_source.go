// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/levels"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewLevelProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &levelSource{r: newRepository(db, mb).(*repository)}
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
	allBadges, err := l.r.getUserBadges(ctx, userLevel.UserID)
	if err != nil {
		return errors.Wrapf(err, "badges/levelSource: failed to fetch all badges")
	}
	for _, currentBadge := range allBadges {
		switch currentBadge.BadgeType {
		case BadgeTypeSocial:
			continue // In progress_source.go .
		case BadgeTypeIce:
			continue // In progress_source.go .
		case BadgeTypeLevel:
			fitsBadge := func(b *badge) bool { return isUnit64FitsBadge(userLevel.TotalLevels, b) }
			lessThanBadge := func(b *badge) bool { return isUnit64LessThanBadge(userLevel.TotalLevels, b) }
			if err = checkAndAchieveBadge(ctx, l.r, userLevel.UserID, currentBadge, fitsBadge, lessThanBadge); err != nil {
				return errors.Wrapf(err, "failed to check if badge %v achieved for user %v", currentBadge.Name, userLevel.UserID)
			}
		}
	}

	return nil
}
