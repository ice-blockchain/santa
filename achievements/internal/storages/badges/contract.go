// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.

var ErrAlreadyAchieved = storage.ErrDuplicate

const (
	// Badge types.
	BadgeTypeSocial = "SOCIAL"
	BadgeTypeIce    = "ICE"
	BadgeTypeLevel  = "LEVEL"
)

type (
	UserID    = string
	BadgeName = string
	BadgeType = string

	Repository interface {
		achieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
		getUnachievedBadges(ctx context.Context, userID UserID) ([]*Badge, error)
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

	// | AchievedBadge is a message broker notification event when user achieves a new badge.
	AchievedBadge struct {
		// Time when badge was achieved.
		AchievedAt time.Time `json:"achievedAt"`
		// Primary key.
		Name string `json:"name"`
		// User.
		UserID UserID `json:"userId"`
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}
	totalBadgesSource struct {
		db tarantool.Connector
	}

	progressSource struct {
		r Repository
	}
	levelSource struct {
		r Repository
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Value    interface{}
		Key      string
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

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

const (
	fieldGlobalValue = 0
)
