// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"

	"cosmossdk.io/math"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/progress"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &progressSource{r: newRepository(db, mb).(*repository)}
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
	allBadges, err := p.r.getUserBadges(ctx, userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "badges/progressSource: failed to fetch all badges")
	}
	for _, currentBadge := range allBadges {
		switch currentBadge.BadgeType {
		case BadgeTypeSocial:
			fitsBadge := func(b *badge) bool { return isUnit64FitsBadge(userProgress.T1Referrals, b) }
			lessThanBadge := func(b *badge) bool { return isUnit64LessThanBadge(userProgress.T1Referrals, b) }
			if err = checkAndAchieveBadge(ctx, p.r, userProgress.UserID, currentBadge, fitsBadge, lessThanBadge); err != nil {
				return errors.Wrapf(err, "failed to achieve badge %v for user %v", currentBadge.Name, userProgress.UserID)
			}
		case BadgeTypeIce:
			fitsBadge := func(b *badge) bool { return isICEFlakesFitsBadge(userProgress.Balance, b) }
			lessThanBadge := func(b *badge) bool { return isICEFlakesLessThanBadge(userProgress.Balance, b) }
			if err = checkAndAchieveBadge(ctx, p.r, userProgress.UserID, currentBadge, fitsBadge, lessThanBadge); err != nil {
				return errors.Wrapf(err, "failed to achieve badge %v for user %v", currentBadge.Name, userProgress.UserID)
			}
		case BadgeTypeLevel:
			continue // Level is not stored at userProgress, so process it in level_source.
		}
	}

	return nil
}

func checkAndAchieveBadge(ctx context.Context, r *repository, userID UserID, badge *badgeWithAchieved, fitsBadge, lessThanBadge func(b *badge) bool) error {
	if !badge.Achieved && fitsBadge(&badge.badge) {
		if err := r.achieveBadge(ctx, userID, badge.Name); err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err, "failed to achieve badge %v to userID %v", badge.Name, userID)
		}
	} else if badge.Achieved { // Already achieved - check if we need to unachieve it.
		if lessThanBadge(&badge.badge) { // Dont compare for > badge.ToInclusive, cuz we store previously achieved badges.
			if err := r.unachieveBadge(ctx, userID, badge.Name); err != nil && !errors.Is(err, errNotAchieved) {
				return errors.Wrapf(err, "failed to unachieve badge %v to userID %v", badge.Name, userID)
			}
		}
	}

	return nil
}

func isUnit64FitsBadge(criteria uint64, b *badge) bool {
	return criteria >= b.FromInclusive && criteria <= b.ToInclusive
}

func isUnit64LessThanBadge(criteria uint64, b *badge) bool {
	return criteria < b.FromInclusive
}

func isICEFlakesFitsBadge(criteria *coin.ICEFlake, b *badge) bool {
	return criteria.GTE(math.NewUint(b.FromInclusive)) && criteria.LTE(math.NewUint(b.ToInclusive))
}

func isICEFlakesLessThanBadge(criteria *coin.ICEFlake, b *badge) bool {
	return criteria.LT(math.NewUint(b.FromInclusive))
}
