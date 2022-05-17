package achievements

import (
	"context"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"time"
)

func (r *repository) GetAchievedUserBadges(ctx context.Context, userID UserID, badgeType BadgeType) ([]*BadgeInventory, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "failed to get referrals because of context failed")
	}

	var queryResult []*badgeInventory
	sql := `
select
    BADGES.*,
    ACHIEVED_USER_BADGES.ACHIEVED_AT > 0 as achieved,
    (count(ALL_USERS.ACHIEVED_AT)*100.0/(SELECT VALUE from TOTAL_USERS where key = 'TOTAL_USERS')) as achieved_percentage
from badges
    left join ACHIEVED_USER_BADGES on BADGES.NAME = ACHIEVED_USER_BADGES.BADGE_NAME and
    ACHIEVED_USER_BADGES.USER_ID = :userId
    left join ACHIEVED_USER_BADGES ALL_USERS on BADGES.NAME = ALL_USERS.BADGE_NAME
where
    BADGES.TYPE = :badgeType
group by BADGES.NAME`
	params := map[string]interface{}{
		"userId":    userID,
		"badgeType": badgeType,
		// TODO: limit, offset? It seems we won't have a lot of badges for the pagination.
	}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil {
		return nil, errors.Wrapf(err, "failed to get achieved badges for type %v and user %v", badgeType, userID)
	}
	result := make([]*BadgeInventory, 0, len(queryResult))
	// We cannot use public BadgeInventory here because of struct embedding, so convert them.
	for _, badge := range queryResult {
		result = append(result, badge.BadgeInventory())
	}

	return result, nil
}

func (r *repository) MarkBadgeAchieved(ctx context.Context, userID UserID, badge *Badge) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}

	sql := `INSERT INTO achieved_user_badges (USER_ID,  BADGE_NAME,  ACHIEVED_AT)
                                      VALUES (:user_id, :badge_name, :achieved_at)`

	params := map[string]interface{}{
		"user_id":     userID,
		"badge_name":  badge.Name,
		"achieved_at": time.Now().UTC().UnixNano(),
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve badge %#v for user %v", badge, userID)
	}
	return nil
}

func (b *badgeInventory) BadgeInventory() *BadgeInventory {
	return &BadgeInventory{
		Badge: Badge{
			Name: b.Name,
			Type: b.BadgeType,
			ProgressInterval: struct {
				Left  uint64 `json:"left" example:"11"`
				Right uint64 `json:"right" example:"22"`
			}{
				Left:  b.FromInclusive,
				Right: b.ToInclusive,
			},
		},
		Achieved:                    b.Achieved,
		GlobalAchievementPercentage: b.GlobalAchievementPercentage,
	}
}
