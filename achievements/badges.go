// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"encoding/json"
	achievementprocessor "github.com/ice-blockchain/santa/achievements/internal/achievement-processor"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"time"

	"github.com/pkg/errors"
)

//nolint:funlen // because on long inline sql
func (r *repository) GetUserBadges(ctx context.Context, userID UserID, badgeType BadgeType) ([]*BadgeInventory, error) {
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
    inner join USER_PROGRESS USER_EXISTS 
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
	now := uint64(time.Now().UTC().UnixNano())
	achievedBadgeByUser := &achievedBadge{
		UserID:     userID,
		BadgeName:  badge.Name,
		AchievedAt: now,
	}
	if err := r.db.InsertTyped("achieved_user_badges", achievedBadgeByUser, &[]*achievedBadge{}); err != nil {
		return errors.Wrapf(err, "failed to achieve badge %#v for user %v", badge, userID)
	}
	return errors.Wrapf(r.sendAchievedBadge(ctx, userID, badge, now), "failed to send achieved badge %#v to message broker", badge)
}

func (r *repository) sendAchievedBadge(ctx context.Context, userID UserID, badge *Badge, achievedTime uint64) error {
	// Send achieved badges to message broker because of we need to calculate total count of achieved badges (in achievementprocessor.badgeSourceProcessor)
	m := achievementprocessor.AchievedBadgeMessage{
		Name:          badge.Name,
		BadgeType:     badge.Type,
		FromInclusive: badge.Interval.Right,
		ToInclusive:   badge.Interval.Left,
		UserID:        userID,
		AchievedAt:    achievedTime,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-badge] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   cfg.MessageBroker.Topics[1].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-badge] failed to send message to broker")
}
