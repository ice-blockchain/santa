// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	_ "embed"
	"io"

	"github.com/framey-io/go-tarantool"

	"github.com/ice-blockchain/santa/achievements/internal/badges"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

var (
	ErrRelationNotFound = storage.ErrRelationNotFound
	ErrNotFound         = storage.ErrNotFound
	ErrDuplicate        = storage.ErrDuplicate
)

const (
	// Badge types.
	BadgeTypeSocial = badges.BadgeTypeSocial
	BadgeTypeIce    = badges.BadgeTypeIce
	BadgeTypeLevel  = badges.BadgeTypeLevel
)

type (
	TaskName         = string
	BadgeName        = string
	BadgeType        = string
	UserID           = string
	UserAchievements struct {
		Role   string           `json:"role" example:"AMBASSADOR"`
		Badges []*BadgeOverview `json:"badges,omitempty"`
		Tasks  []*TaskTODO      `json:"tasks,omitempty"`
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
	TaskTODO struct {
		Name     string `json:"name" example:"CLAIM_USERNAME"`
		Index    uint64 `json:"index" example:"0"`
		Achieved bool   `json:"achieved" example:"false"`
	}
	Task struct {
		CompletedAt *time.Time `json:"completedAt" swaggertype:"string" example:"2022-01-03T16:20:52.156534Z"`
		UserID      UserID     `json:"userId" uri:"-" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
		Name        TaskName   `uri:"taskName" json:"name" example:"TASK1"`
		Index       uint64     `json:"index" example:"0"`
	}
	Badge = badges.Badge

	BadgeProgressInterval = badges.ProgressInterval

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
	}
	WriteRepository interface {
		CompleteTask(context.Context, *Task) error
		UnCompleteTask(context.Context, *Task) error
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
	proxyProcessor struct {
		internalProcessors []messagebroker.Processor
		parallelProcessing bool
	}
)
