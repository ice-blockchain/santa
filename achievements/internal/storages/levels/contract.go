package levels

import (
	"context"
	"github.com/framey-io/go-tarantool"
)

// Public API.
type (
	UserID = string

	Repository interface {
		IncrementUserLevel(ctx context.Context, userID UserID) error
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
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

	// | achievedUserLevels stores achieved user's level in database (achieved_user_levels space).
	achievedUserLevels struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{} `msgpack:",asArray"`
		UserID    UserID
		LevelName string
		// Timestamp.
		AchievedAt uint64
	}
)

const (
	achievedUserLevelsSpace = "ACHIEVED_USER_LEVELS"
)
