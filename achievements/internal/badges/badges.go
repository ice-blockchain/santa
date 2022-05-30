// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db: db,
		mb: mb,
	}
}

func (r *repository) achieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve badge failed because context failed")
	}
	now := time.Now().UTC()
	sql := `INSERT INTO achieved_user_badges(USER_ID, badge_name,   ACHIEVED_AT)
                                                   VALUES(:userID,  :badgeName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"badgeName":  badgeName,
		"achievedAt": uint64(now.UnixNano()),
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's badge %v for userID:%v", badgeName, userID)
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, userID, badgeName, now), "failed to send achieved badge %v to message broker for userId:%v", badgeName, userID)
}

func (r *repository) sendAchievedBadge(ctx context.Context, userID UserID, badgeName BadgeName, achievedTime time.Time) error {
	m := AchievedBadge{
		Name:       badgeName,
		UserID:     userID,
		AchievedAt: achievedTime,
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

func (r *repository) getUnachievedBadges(ctx context.Context, userID UserID) ([]*Badge, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "fetching badges failed because context failed")
	}
	sql := `
SELECT badges.* FROM badges
LEFT JOIN achieved_user_badges
        ON badges.name = achieved_user_badges.badge_name 
        AND ACHIEVED_USER_BADGES.USER_ID = :userID
where ACHIEVED_AT is null
`
	params := map[string]interface{}{
		"userID": userID,
	}
	queryResult := []*badge{}
	if err := r.db.PrepareExecuteTyped(sql, params, &queryResult); err != nil && !errors.Is(err, storage.ErrRelationNotFound) {
		return nil, errors.Wrapf(err, "failed to fetch all badges")
	} else if err != nil && errors.Is(err, storage.ErrRelationNotFound) {
		return nil, nil
	}
	result := make([]*Badge, 0, len(queryResult))
	for _, badge := range queryResult {
		result = append(result, badge.Badge())
	}

	return result, nil
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
