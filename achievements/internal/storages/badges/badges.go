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
	// Check if such badge exists before achieve it.
	badgeObject, err := r.GetBadge(ctx, badgeName)
	if err != nil {
		return errors.Wrapf(err, "failed to read badge for badgeName:%v", badgeName)
	}
	now := uint64(time.Now().UTC().UnixNano())
	achievedBadgeByUser := &achievedBadge{
		UserID:     userID,
		BadgeName:  badgeObject.Name,
		AchievedAt: now,
	}
	if err := r.db.InsertTyped("ACHIEVED_USER_BADGES", achievedBadgeByUser, &[]*achievedBadge{}); err != nil {
		return errors.Wrapf(err, "failed to achieve badge %#v for user %v", badgeObject, userID)
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, userID, badgeObject, now), "failed to send achieved badge %#v to message broker", badgeObject)
}

// TODO: move to internal -> badges
func (r *repository) sendAchievedBadge(ctx context.Context, userID UserID, badge *Badge, achievedTime uint64) error {
	// Send achieved badges to message broker because of we need to calculate total count of achieved badges (in achievementprocessor.totalBadgesSource).
	m := AchievedBadgeMessage{
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
		Topic:   r.publishAchievedBadgesTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-badge] failed to send message to broker")
}

// TODO: move to internal -> badges
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
