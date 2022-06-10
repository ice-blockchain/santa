// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db: db,
		mb: mb,
	}
}

func achieveBadgeAndHandleError(ctx context.Context, r *repository, userID UserID, badgeName BadgeName) error {
	if err := r.achieveBadge(ctx, userID, badgeName); err != nil && !errors.Is(err, errAlreadyAchieved) {
		return errors.Wrapf(err, "failed to achieve badge %v to userID %v", badgeName, userID)
	}

	return nil
}

func unachieveBadgeAndHandleError(ctx context.Context, r *repository, userID UserID, badgeName BadgeName) error {
	if err := r.unachieveBadge(ctx, userID, badgeName); err != nil && !errors.Is(err, errNotAchieved) {
		return errors.Wrapf(err, "failed to achieve badge %v to userID %v", badgeName, userID)
	}

	return nil
}

func (r *repository) achieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve badge failed because context failed")
	}
	ab := &AchievedBadge{
		AchievedAt: time.Now(),
		Name:       badgeName,
		UserID:     userID,
	}
	if err := r.db.InsertTyped(achievedBadgesSpace, ab, &[]*AchievedBadge{}); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_FOUND {
			return errors.Wrapf(errAlreadyAchieved, "badge %v already achieved to user %v", badgeName, userID)
		}

		return errors.Wrapf(err, "Failed to insert achieved badge %v to userID %v", badgeName, userID)
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, &AchievedBadgeMessage{
		AchievedBadge: ab,
	}), "failed to send achieved badge %v to message broker for userId:%v", badgeName, userID)
}

func (r *repository) unachieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve badge failed because context failed")
	}
	now := time.Now()
	key := &achievedBadgeKey{
		UserID:    userID,
		BadgeName: badgeName,
	}
	res := []*AchievedBadge{}
	if err := r.db.DeleteTyped(achievedBadgesSpace, "pk_unnamed_ACHIEVED_USER_BADGES_1", key, &res); err != nil {
		tErr := new(tarantool.Error)
		if errors.As(err, tErr) && tErr.Code == tarantool.ER_TUPLE_NOT_FOUND {
			return errors.Wrapf(errNotAchieved, "badge %v already achieved to user %v", badgeName, userID)
		}

		return errors.Wrapf(err, "Failed to delete achieved badge %v for userID %v", badgeName, userID)
	}
	message := &AchievedBadgeMessage{
		AchievedBadge: &AchievedBadge{
			AchievedAt: now,
			Name:       badgeName,
			UserID:     userID,
		},
		Unachieved: true,
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, message), "failed to send achieved badge %v to message broker for userId:%v", badgeName, userID)
}

func (r *repository) sendAchievedBadge(ctx context.Context, m *AchievedBadgeMessage) error {
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-badge] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     m.UserID,
		Topic:   cfg.MessageBroker.Topics[1].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-badge] failed to send message to broker")
}

func (r *repository) getUserBadges(ctx context.Context, userID UserID) ([]*badgeWithAchieved, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "fetching badges failed because context failed")
	}
	sql := `
select
    BADGES.*,
    ACHIEVED_USER_BADGES.ACHIEVED_AT is not NULL as achieved
from BADGES
    left join ACHIEVED_USER_BADGES
on BADGES.NAME = ACHIEVED_USER_BADGES.BADGE_NAME and ACHIEVED_USER_BADGES.USER_ID = :userID
order by BADGES.TYPE, BADGES.FROM_INCLUSIVE
`
	params := map[string]interface{}{
		"userID": userID,
	}
	queryResult := []*badgeWithAchieved{}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil && !errors.Is(err, storage.ErrRelationNotFound) {
		return nil, errors.Wrapf(err, "failed to fetch all badges")
	} else if err != nil && errors.Is(err, storage.ErrRelationNotFound) {
		return nil, nil
	}

	return queryResult, nil
}

func (b *badge) Badge() *Badge {
	return &Badge{
		Name: b.Name,
		Type: b.BadgeType,
		Interval: ProgressInterval{
			Left:  b.FromInclusive,
			Right: b.ToInclusive,
		},
	}
}
