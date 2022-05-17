// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.
type (
	UserID        = string
	LevelName     = string
	Repository    interface{}
	AchievedLevel struct {
		AchievedAt  *time.Time `json:"achievedAt"`
		UserID      UserID     `json:"userId"`
		LevelName   LevelName  `json:"levelName"`
		TotalLevels uint64     `json:"totalLevels"`
		Unachieved  bool       `json:"unachieved"`
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}

	// | taskSource is source processor to increment user's levels on task completion ( each task 1 level, Levels -> #7 ).
	taskSource struct {
		r *repository
	}
	// | userSource is source processor to increment user's levels on phone number confirmation (Levels -> #8).
	userSource struct {
		r *repository
	}
	// | progressSource is source processor to increment user's levels on consecutive mining sessions.
	progressSource struct {
		r *repository
	}

	// | agendaReferralsSource is source processor to increment user's levels on referred users from agenda (Levels -> 9-11).
	agendaReferralsSource struct {
		r *repository
	}

	config struct {
		Levels struct {
			AgendaReferrals           map[string]uint64 `yaml:"agendaReferrals"`
			ConsecutiveMiningSessions map[string]uint32 `yaml:"consecutiveMiningSessions"`
			TaskCompletion            map[string]string `yaml:"taskCompletion"`
		} `yaml:"levels"`

		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}

	// | currentUserLevels is internal type to handle database values (current_user_levels space).
	currentUserLevels struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Timestamp.
		UpdatedAt *time.Time
		// Link to user.
		UserID UserID
		// Total number of levels achieved to user (needed for badge achieving).
		Level uint64
	}
	// | achievedUserLevel is internal type to handle database values (ACHIEVED_USER_LEVELS space).
	achievedUserLevel struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{} `msgpack:",asArray"`
		UpdatedAt *time.Time
		UserID    UserID
		LevelName LevelName
	}
	level struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name LevelName
	}
	levelWithAchieved struct {
		level
		// Flag if level was achieved by user (to understand if we need to achieve/unachieve it).
		Achieved bool
	}
)

const (
	currentUsersLevelsSpace = "CURRENT_USER_LEVELS"
	achievedUserLevelsSpace = "ACHIEVED_USER_LEVELS"
	levelsSpace             = "LEVELS"

	levelForPhoneNumberConfirmation = "L13"
	fieldCurrentUserLevelsLevel     = 2
	fieldCurrentUserLevelsUpdatedAt = 0
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var (
	cfg                config
	errAlreadyAchieved = storage.ErrDuplicate
	errNotAchieved     = storage.ErrNotFound
)
