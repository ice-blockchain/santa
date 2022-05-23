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

	// | taskSource is source processor to increment user's levels on task completion ( each task 1 level, Levels -> #7
	taskSource struct {
		r Repository
	}
	// | userSource is source processor to increment user's levels on phone number confirmation (Levels -> #8)
	userSource struct {
		r Repository
	}
	// | economyMiningSource is source processor to increment user's levels on consecutive mining sessions
	economyMiningSource struct {
		r Repository
	}

	// `userAchievements` is an internal type to store user achievements badges in database (USER_ACHIEVEMENTS space).
	userAchievements struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// User's role, one of PIONEER, ABMASSADOR, ...3rd...
		Role string
		// User's balance (ICE).
		Balance uint64
		Level   uint32
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
	}
)

const (
	userLevelField        = 3
	userAchievementsSpace = "USER_ACHIEVEMENTS"
)
