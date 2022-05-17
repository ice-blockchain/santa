// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	_ "embed"
	"io"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.
var (
	ErrNotFound = storage.ErrNotFound
)

type (
	BadgeName        = string
	BadgeType        = string
	UserID           = string
	UserAchievements struct {
		Role   string           `json:"role" example:"AMBASSADOR"`
		Badges []*BadgeOverview `json:"badges,omitempty"`
		Tasks  []*Task          `json:"tasks,omitempty"`
		Level  uint64           `json:"level" example:"11"`
	}
	BadgeInventory struct {
		Badge
		Achieved bool `json:"achieved" example:"false"`
		// The percentage of all the users that have this badge.
		GlobalAchievementPercentage float64 `json:"globalAchievementPercentage" example:"25.5"`
	}
	BadgeOverview struct {
		Badge
		Position struct {
			X     uint64 `json:"x" example:"3"`
			OutOf uint64 `json:"outOf" example:"10"`
		} `json:"position"`
	}
	Task struct {
		Name     string `json:"name" example:"CLAIM_USERNAME"`
		Index    uint64 `json:"index" example:"0"`
		Achieved bool   `json:"achieved" example:"false"`
	}
	Badge struct {
		Name             string    `json:"name" example:"ICE Breaker"`
		Type             BadgeType `json:"type" example:"SOCIAL"`
		ProgressInterval struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		} `json:"interval"`
	}
	Repository interface {
		io.Closer
		ReadBadgesRepository
	}
	Processor interface {
		io.Closer
		WriteBadgesRepository
		CheckHealth(context.Context) error
	}

	ReadBadgesRepository interface {
		GetAchievedUserBadges(ctx context.Context, userID UserID, badgeType BadgeType) ([]*BadgeInventory, error)
	}
	WriteBadgesRepository interface {
		AddBadge(ctx context.Context, badge *Badge) error
		MarkBadgeAchieved(ctx context.Context, userID UserID, badge *Badge) error
	}
)

// Private API.
const (
	applicationYamlKey = "achievements"
)

var (
	//go:embed DDL.lua
	ddl string
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg config
)

type (
	repository struct {
		db tarantool.Connector
	}

	processor struct {
		close func() error
		WriteBadgesRepository
	}

	config struct{}
	// We need this struct to deserialize db response from ReadBadgesRepository.GetAchievedUserBadges because of API struct uses struct embedding.
	badgeInventory struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name BadgeName
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// Min-max range of the certain value (based on badgeType) to achieve the badge.
		FromInclusive uint64
		ToInclusive   uint64
		// If the badge was achieved by user.
		Achieved bool `json:"achieved" example:"false"`
		// The percentage of all the users that have this badge.
		GlobalAchievementPercentage float64 `json:"globalAchievementPercentage" example:"25.5"`
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
