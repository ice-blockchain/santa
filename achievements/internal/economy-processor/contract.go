package economyprocessor

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
)

type (
	UserID          = string
	BadgeName       = string
	TaskName        = string
	WriteRepository interface {
		AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
		AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error
		IncrementUserLevel(ctx context.Context, userID UserID) error
	}

	miningEventSourceProcessor struct {
		db tarantool.Connector
		r  WriteRepository
	}

	//nolint:deadcode,unused // TODO: economy updates (balance at user_achievements) when they'll be implemented in freezer.
	balanceUpdateSourceProcessor struct {
		db tarantool.Connector
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
	consecutiveUserMiningSessionsSpace = "consecutive_user_mining_sessions"
	//nolint:gomnd,nolintlint // 24 hour is session duration, and up to 10 hours between sessions
	maxTimeBetweenConsecutiveMiningSessions = (24 + 10) * time.Hour
	// Database fields for tarantol oprations, we   keep them in sync with DDL.
	lastMinintStartedAtField = 1
	maxCountField            = 2
)
