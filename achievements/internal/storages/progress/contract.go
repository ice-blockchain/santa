// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
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
		InsertAgendaReferrals(ctx context.Context, agendaOwnerID, userIDInAgenda UserID) error

		ResetConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error
		UpdateConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error
	}
	ReadRepository interface {
		Repository
		GetUserProgress(userID users.UserID) (*UserProgress, error)
	}
	UserProgress struct {
		// User's balance.
		Balance *coin.ICEFlake `json:"balance"`
		// Primary key.
		UserID UserID `json:"userID"`
		// Agenda phone numbers hashes we store to see if users are in agenda for each other.
		AgendaPhoneNumbersHashes string `json:"agendaPhoneNumbersHashes"`
		// Count of user's referrals on Tier 1.
		T1Referrals uint64 `json:"t1Referrals"`
		// Timestamp.
		LastMiningStartedAt uint64 `json:"lastMiningStartedAt"`
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxConsecutiveMiningSessionsCount uint32 `json:"maxConsecutiveMiningSessionsCount"`
		TotalUserReferralPings            uint32 `json:"totalUserReferralPings"`
	}
	AgendaReferralsCount struct {
		UserID               UserID `json:"userID"`
		AgendaReferralsCount uint64 `json:"agendaReferralsCount"`
	}
)

// Private API.
type (
	repository struct {
		db                               tarantool.Connector
		mb                               messagebroker.Client
		publishUpdatedProgressTopic      string
		publishAgendaReferralsCountTopic string
	}
	// | userSource is a source processor to insert/update user's state at USER_PROGRESS space and to count total users.
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
		Value    interface{}
		Key      string
	}

	// | userProgress  is an internal type to store user current progress state in database (USER_PROGRESS space).
	userProgress struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// User's balance.
		*coin.Coin
		// Primary key.
		UserID UserID
		// AgendaPhoneNumbersHashes we store to see if users are in agenda for each other.
		AgendaPhoneNumbersHashes string
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
		// Timestamp.
		LastMiningStartedAt uint64
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxConsecutiveMiningSessionsCount uint32
		TotalUserReferalPings             uint32
	}

	// | agendaReferrals  is an internal type to store if users are in agenda for each other (agenda_referrals space).
	agendaReferrals struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack     struct{} `msgpack:",asArray"`
		UserID       UserID
		AgendaUserID UserID
	}
)

const (
	userProgressSpace    = "USER_PROGRESS"
	agendaReferralsSpace = "AGENDA_REFERRALS"
	//nolint:gomnd,nolintlint // 24 hour is session duration, and up to 10 hours between sessions
	maxTimeBetweenConsecutiveMiningSessions = (24 + 10) * time.Hour

	// Database fields for tarantool oprations, we   keep them in sync with DDL.
	fieldAgendaPhoneNumbersHashes          = 2
	fieldT1Referrals                       = 3
	fieldLastMiningStartedAt               = 4
	fieldMaxConsecutiveMiningSessionsCount = 5
	fieldGlobalValue                       = 0
)
