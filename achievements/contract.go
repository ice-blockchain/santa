// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	_ "embed"
	"github.com/framey-io/go-tarantool"
	"io"
)

// Public API.

type (
	BadgeName        = string
	BadgeType        = string
	UserID           = string
	UserAchievements struct {
		Level  uint64           `json:"level" example:"11"`
		Role   string           `json:"role" example:"AMBASSADOR"`
		Badges []*BadgeOverview `json:"badges,omitempty"`
		Tasks  []*Task          `json:"tasks,omitempty"`
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
		Achieved bool   `json:"achieved" example:"false"`
		Name     string `json:"Name" example:"CLAIM_USERNAME"`
		Index    uint64 `json:"index" example:"0"`
	}
	Badge struct {
		Name             string    `json:"Name" example:"ICE Breaker"`
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

	// TODO split one achievements package to 3 packages for badges, tasks and achievements? Or remove Badges from Name if it'll be implemented all in one
	ReadBadgesRepository interface {
		GetAchievedUserBadges(ctx context.Context, userId UserID, badgeType BadgeType) ([]*BadgeInventory, error)
	}
	WriteBadgesRepository interface {
		AddBadge(ctx context.Context, badge *Badge) error
		MarkBadgeAchieved(ctx context.Context, userId UserID, badge *Badge) error
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
	// TODO split for 3 packages like badges, tasks and achievements?
	repository struct {
		db tarantool.Connector
	}

	processor struct {
		close func() error
		WriteBadgesRepository
	}

	config struct {
	}
	// badge is an internal type to store badges in database
	badge struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name BadgeName
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// Min-max range of the certain value (based on BadgeType) to achieve the badge
		From_inclusive uint64
		To_inclusive   uint64
	}
	// achievedBadge is an internal type to store user's achieved badges in database
	achievedBadge struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{} `msgpack:",asArray"`
		UserID     UserID
		BadgeName  string
		AchievedAt uint64
	}
	// we nned this srtuct to delesialize db responce from ReadBadgesRepository.GetAchievedUserBadges because of API struct uses struct embedding
	badgeInventory struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name BadgeName
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// Min-max range of the certain value (based on badgeType) to achieve the badge
		From_inclusive uint64
		To_inclusive   uint64
		// if the badge was achieved by user
		Achieved bool `json:"achieved" example:"false"`
		// The percentage of all the users that have this badge.
		GlobalAchievementPercentage float64 `json:"globalAchievementPercentage" example:"25.5"`
	}
)
