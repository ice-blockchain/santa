package achievements

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"time"
)

// Public API.
type (
	UserID                     = string
	UserAchievementsRepository interface {
		UpdateT1ReferralsCount(userID users.UserID, diff int64) error
		InsertUserAchievements(userID users.UserID) error
		GetUserAchievements(userID users.UserID) (*UserAchievements, error)
		DeleteUserAchievements(userID users.UserID) error
		UpdateTotalUsersCount(diff int64) error
	}
	ConsecutiveMiningSessionRepository interface {
		GetConsecutiveMiningSessions(userID UserID) (*ConsecutiveMiningSessions, error)
		InsertConsecutiveMiningSessions(userID UserID, lastStartedTS uint64) error
		ResetConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error
		UpdateConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error
	}

	ConsecutiveMiningSessions struct {
		UserID UserID
		// Timestamp.
		LastMiningStartedAt uint64
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxCount uint32
	}

	UserAchievements struct {
		UserID UserID
		// User's role, one of PIONEER, ABMASSADOR, ...3rd...
		Role string
		// User's balance (ICE).
		Balance uint64
		Level   uint32
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
	}
	// | userSource is a source processor to insert/update user's state at USER_ACHIEVEMENTS space and to count total users
	userSource struct {
		r UserAchievementsRepository
	}
	// | economyMiningSource is a source processor to count user's consecutive mining sessions.
	economyMiningSource struct {
		r ConsecutiveMiningSessionRepository
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		// For now we're saving only integer, but scalar may be one of
		// boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary,
		// but I cant find golang mapping in docs (interface{}?).
		Value uint64
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

	// | consecutiveUserMiningSessions is an internal type to store count of user mining sessions in database.
	consecutiveUserMiningSessions struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// Timestamp.
		LastMiningStartedAt uint64
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxCount uint32
	}
)

const (
	userAchievementsSpace = "USER_ACHIEVEMENTS"
	fieldT1Referrals      = 4

	consecutiveUserMiningSessionsSpace = "CONSECUTIVE_USER_MINING_SESSIONS"
	//nolint:gomnd,nolintlint // 24 hour is session duration, and up to 10 hours between sessions
	maxTimeBetweenConsecutiveMiningSessions = (24 + 10) * time.Hour
	// Database fields for tarantol oprations, we   keep them in sync with DDL.
	lastMinintStartedAtField = 1
	maxCountField            = 2
)
