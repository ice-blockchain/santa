// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.
type (
	UserID     = string
	LevelName  = string
	Repository interface {
		achieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error
		insertCurrentUserLevel(ctx context.Context, userID UserID) error
	}
	AchievedLevel struct {
		AchievedAt  time.Time `json:"achievedAt"`
		UserID      UserID    `json:"userId"`
		LevelName   LevelName `json:"levelName"`
		TotalLevels uint64    `json:"totalLevels"`
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
		r Repository
	}
	// | userSource is source processor to increment user's levels on phone number confirmation (Levels -> #8).
	userSource struct {
		r Repository
	}
	// | progressSource is source processor to increment user's levels on consecutive mining sessions.
	progressSource struct {
		r Repository
	}

	// | agendaReferralsSource is source processor to increment user's levels on referred users from agenda (Levels -> 9-11).
	agendaReferralsSource struct {
		r Repository
	}

	config struct {
		Levels struct {
			AgendaReferrals           map[uint64]string `yaml:"agendaReferrals"`
			ConsecutiveMiningSessions map[uint32]string `yaml:"consecutiveMiningSessions"`
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
		UserID   UserID
		// Total number of levels achieved to user (needed for badge achieving).
		Level uint64
		// Timestamp.
		UpdatedAt uint64
	}
)

const (
	currentUsersLevelsSpace = "CURRENT_USER_LEVELS"

	levelForPhoneNumberConfirmation = "L13"
	fieldCurrentUserLevelsLevel     = 1
	fieldCurrentUserLevelsUpdatedAt = 2
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var (
	cfg                config
	errAlreadyAchieved = storage.ErrDuplicate
)
