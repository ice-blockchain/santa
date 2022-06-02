// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"github.com/framey-io/go-tarantool"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

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

	Repository interface{}

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
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{}   `msgpack:",asArray"`
		AchievedAt *time.Time `json:"achievedAt"`
		UserID     UserID     `json:"userId"`
		Name       string     `json:"name"`
	}

	AchievedBadgeMessage struct {
		*AchievedBadge
		// True in case if badge was unachieved.
		Unachieved bool `json:"unachieved"`
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
		r *repository
	}
	levelSource struct {
		r *repository
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
	badgeWithAchieved struct {
		badge
		// Flag if badge was achieved by user (to understand if we need to achieve/unachieve it).
		Achieved bool
	}

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
	achievedBadgeKey struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{} `msgpack:",asArray"`
		UserID    UserID
		BadgeName BadgeName
	}
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var (
	cfg                config
	errAlreadyAchieved = storage.ErrDuplicate
	errNotAchieved     = storage.ErrNotFound
)

const (
	fieldGlobalValue = 0

	achievedBadgesSpace = "ACHIEVED_USER_BADGES"
)
