package badges

import (
	"context"
	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.

var (
	ErrAlreadyAchieved = storage.ErrDuplicate
)

type (
	UserID    = string
	BadgeName = string
	BadgeType = string

	Repository interface {
		AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
	}

	Badge struct {
		Name     string           `json:"name" example:"ICE Breaker"`
		Type     BadgeType        `json:"type" example:"SOCIAL"`
		Interval ProgressInterval `json:"interval"`
	}
	ProgressInterval struct {
		Left  uint64 `json:"left" example:"11"`
		Right uint64 `json:"right" example:"22"`
	}

	// | AchievedBadgeMessage is a message broker notification event when user achieves a new badge.
	AchievedBadgeMessage struct {
		// Primary key.
		Name string
		// User.
		UserID UserID
		// Time when badge was achieved.
		AchievedAt uint64
	}
)

// Private API.
type (
	repository struct {
		db                         tarantool.Connector
		mb                         messagebroker.Client
		publishAchievedBadgesTopic string
	}

	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		// For now we're saving only integer, but scalar may be one of
		// boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary,
		// but I cant find golang mapping in docs (interface{}?).
		Value uint64
	}

	totalBadgesSource struct {
		db tarantool.Connector
	}
	badge struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name BadgeName
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// Min-max range of the certain value (based on badgeType) to achieve the badge.
		FromInclusive uint64
		ToInclusive   uint64
	}

	// `achievedBadge` is an internal type to store user's achieved badges in database.
	achievedBadge struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{} `msgpack:",asArray"`
		UserID     UserID
		BadgeName  string
		AchievedAt uint64
	}
)

const (
	badgesSpace = "BADGES"
)
