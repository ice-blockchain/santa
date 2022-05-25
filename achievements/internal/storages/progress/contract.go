package progress

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"time"
)

// Public API.
type (
	UserID     = string
	Repository interface {
		InsertUserProgress(ctx context.Context, user *users.User) error
		DeleteUserProgress(userID users.UserID) error
		UpdateTotalUsersCount(diff int64) error
		UpdateT1ReferralsCount(ctx context.Context, userID users.UserID, diff int64) error
		UpdateAgendaPhoneNumbersHashes(ctx context.Context, userID users.UserID, agendaPhoneHashes string) error

		ResetConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error
		UpdateConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error
	}
	ReadRepository interface {
		Repository
		GetUserProgress(userID users.UserID) (*UserProgress, error)
	}
	UserProgress struct {
		UserID UserID
		// Agenda phone numbers hashes we store to see if users are in agenda for each other.
		AgendaPhoneNumbersHashes string
		// User's balance (iceflakes).
		Balance uint64
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
		// Timestamp.
		LastMiningStartedAt uint64
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxConsecutiveMiningSessionsCount uint32
		TotalUserReferalPings             uint32
	}
)

// Private API.
type (
	repository struct {
		db                          tarantool.Connector
		mb                          messagebroker.Client
		publishUpdatedProgressTopic string
	}
	// | userSource is a source processor to insert/update user's state at USER_ACHIEVEMENTS space and to count total users
	userSource struct {
		r ReadRepository
	}
	// | economyMiningSource is a source processor to count user's consecutive mining sessions.
	economyMiningSource struct {
		r ReadRepository
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

	// | userProgress  is an internal type to store user achievements badges in database (USER_ACHIEVEMENTS space).
	userProgress struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// AgendaPhoneNumbersHashes we store to see if users are in agenda for each other.
		AgendaPhoneNumbersHashes string
		// User's balance (iceflakes).
		Balance uint64
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
		// Timestamp.
		LastMiningStartedAt uint64
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxConsecutiveMiningSessionsCount uint32
		TotalUserReferalPings             uint32
	}
)

const (
	userProgressSpace = "USER_PROGRESS"

	//nolint:gomnd,nolintlint // 24 hour is session duration, and up to 10 hours between sessions
	maxTimeBetweenConsecutiveMiningSessions = (24 + 10) * time.Hour
	// Database fields for tarantol oprations, we   keep them in sync with DDL.
	fieldAgendaPhoneNumbersHashes          = 1
	fieldT1Referrals                       = 3
	lastMinintStartedAtField               = 4
	maxConsecutiveMiningSessionsCountField = 5
)
