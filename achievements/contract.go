// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	_ "embed"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"io"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.
var (
	ErrRelationNotFound = storage.ErrRelationNotFound
)

type (
	BadgeName        = string
	TaskName         = string
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
		Name     string           `json:"name" example:"ICE Breaker"`
		Type     BadgeType        `json:"type" example:"SOCIAL"`
		Interval ProgressInterval `json:"interval"`
	}
	ProgressInterval struct {
		Left  uint64 `json:"left" example:"11"`
		Right uint64 `json:"right" example:"22"`
	}
	Repository interface {
		io.Closer
		ReadRepository
	}
	Processor interface {
		io.Closer
		WriteRepository
		CheckHealth(context.Context) error
	}

	ReadRepository interface {
		GetUserBadges(ctx context.Context, userID UserID, badgeType BadgeType) ([]*BadgeInventory, error)
		GetBadge(ctx context.Context, badgeName BadgeName) (*Badge, error)
		GetTask(ctx context.Context, taskName TaskName) (*Task, error)
	}
	WriteRepository interface {
		AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
		AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error
		IncrementUserLevel(ctx context.Context, userID UserID) error
	}
)

// Private API.
const (
	applicationYamlKey    = "achievements"
	userAchievementsSpace = "USER_ACHIEVEMENTS"
	badgesSpace           = "BADGES"
	tasksSpace            = "TASKS"
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
		mb messagebroker.Client
	}

	processor struct {
		close func() error
		WriteRepository
	}

	config struct {
		MessageBroker struct {
			ConsumingTopics []string `yaml:"consumingTopics"`
			Topics          []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
	// We need this struct to deserialize db response from ReadRepository.GetAchievedUserBadges because of API struct uses struct embedding.
	badgeInventory struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		badge
		// If the badge was achieved by user.
		Achieved bool `json:"achieved" example:"false"`
		// The percentage of all the users that have this badge.
		GlobalAchievementPercentage float64 `json:"globalAchievementPercentage" example:"25.5"`
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

	task struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name TaskName
		// index of the task ( they should be done in specific order)
		Index uint64
	}

	// `achievedBadge` is an internal type to store user's achieved badges in database.
	achievedBadge struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{} `msgpack:",asArray"`
		UserID     UserID
		BadgeName  string
		AchievedAt uint64
	}
	// `achievedTask` is an internal type to store user's achieved tasks in database.
	achievedTask struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{} `msgpack:",asArray"`
		UserID     UserID
		TaskName   string
		AchievedAt uint64
	}

	// `userAchievements` is an internal type to store user achievements badges in database (USER_ACHIEVEMENTS space)
	userAchievements struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// User's role, one of PIONEER, ABMASSADOR, ...3rd...
		Role string
		// User's balance (ICE).
		Balance uint64 // FIXME: it is float64/DOUBLE in freezer for now, how to convert them? Or refactor freezer to uint64.
		Level   uint32
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
	}
)
