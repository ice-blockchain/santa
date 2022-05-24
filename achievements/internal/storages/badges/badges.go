package badges

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"time"
)

func New(db tarantool.Connector, mb messagebroker.Client) Repository {
	var config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
	appCfg.MustLoadFromKey("achievements", &config)
	return &repository{
		db:                         db,
		mb:                         mb,
		publishAchievedBadgesTopic: config.MessageBroker.Topics[1].Name,
	}
}

func (r *repository) AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve badge failed because context failed")
	}
	now := uint64(time.Now().UTC().UnixNano())
	sql := `INSERT INTO achieved_user_badges(USER_ID, badge_name,   ACHIEVED_AT)
                                                   VALUES(:userID,  :badgeName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"badgeName":  badgeName,
		"achievedAt": now,
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's level for userID:%v", userID)
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, userID, badgeName, now), "failed to send achieved badge %v to message broker for userId:%v", badgeName, userID)
}

func (r *repository) sendAchievedBadge(ctx context.Context, userID UserID, badgeName BadgeName, achievedTime uint64) error {
	m := AchievedBadgeMessage{
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
		Topic:   r.publishAchievedBadgesTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-badge] failed to send message to broker")
}

func (r *repository) GetBadge(ctx context.Context, badgeName BadgeName) (*Badge, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "get badge failed because context failed")
	}
	var res badge
	if err := r.db.GetTyped(badgesSpace, "pk_unnamed_BADGES_1", tarantool.StringKey{S: badgeName}, &res); err != nil {
		return nil, errors.Wrapf(err, "unable to get badges record for badgeName:%v", badgeName)
	}
	if res.Name == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no badge record for name:%v", badgeName)
	}

	return res.Badge(), nil
}

func (b *badge) Badge() *Badge {
	return &Badge{
		Name:     b.Name,
		Type:     b.BadgeType,
		Interval: ProgressInterval{b.FromInclusive, b.ToInclusive},
	}
}
