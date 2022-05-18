// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

//nolint:funlen // because on long inline sql
func (r *repository) GetAchievedUserBadges(ctx context.Context, userID UserID, badgeType BadgeType) ([]*BadgeInventory, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "failed to get referrals because of context failed")
	}
	var queryResult []*badgeInventory
	sql := `
select
    BADGES.*,
    ACHIEVED_USER_BADGES.ACHIEVED_AT > 0 as achieved,
    ((SELECT VALUE from GLOBAL where key = 'TOTAL_BADGES_'||BADGES.NAME)*100.0/(SELECT VALUE from GLOBAL where key = 'TOTAL_USERS')) as achieved_percentage
from badges
    left join ACHIEVED_USER_BADGES 
		on BADGES.NAME = ACHIEVED_USER_BADGES.BADGE_NAME and ACHIEVED_USER_BADGES.USER_ID = :userId
    inner join USER_ACHIEVEMENTS USER_EXISTS 
		on USER_EXISTS.USER_ID = :userId
where
    BADGES.TYPE = :badgeType
order by BADGES.FROM_INCLUSIVE`
	params := map[string]interface{}{
		"userId":    userID,
		"badgeType": badgeType,
	}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil {
		return nil, errors.Wrapf(err, "failed to get achieved badges for type %v and user %v", badgeType, userID)
	}
	if len(queryResult) == 0 {
		return nil, errors.Wrapf(ErrRelationNotFound, "no user found with id: %v", userID)
	}
	result := make([]*BadgeInventory, 0, len(queryResult))
	// We cannot use public BadgeInventory here because of struct embedding, so convert them.
	for _, badge := range queryResult {
		result = append(result, badge.BadgeInventory())
	}

	return result, nil
}

func (b *badgeInventory) BadgeInventory() *BadgeInventory {
	return &BadgeInventory{
		Badge: Badge{
			Name: b.Name,
			Type: b.BadgeType,
			Interval: ProgressInterval{
				Left:  b.FromInclusive,
				Right: b.ToInclusive,
			},
		},
		Achieved:                    b.Achieved,
		GlobalAchievementPercentage: b.GlobalAchievementPercentage,
	}
}

func (r *repository) AchieveBadge(ctx context.Context, userID UserID, badge *Badge) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}
	achievedBadgeByUser := &achievedBadge{
		UserID:     userID,
		BadgeName:  badge.Name,
		AchievedAt: uint64(time.Now().UTC().UnixNano()),
	}

	return errors.Wrapf(r.db.InsertTyped("achieved_user_badges", achievedBadgeByUser, &[]*achievedBadge{}),
		"failed to achieve badge %#v for user %v", badge, userID)
}
