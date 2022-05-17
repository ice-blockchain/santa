// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	wt "github.com/ice-blockchain/wintr/time"
	"github.com/pkg/errors"
)

// Public API.
type (
	UserID       = string
	Repository   interface{}
	UserProgress struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// User's balance.
		Balance *coin.ICEFlake
		// Timestamp.
		LastMiningStartedAt *wt.Time `json:"lastMiningStartedAt"`
		// Primary key.
		UserID UserID `json:"userId"`
		// Agenda phone numbers hashes we store to see if users are in agenda for each other.
		AgendaPhoneNumbersHashes string `json:"agendaPhoneNumbersHashes"`
		// Count of user's referrals on Tier 1.
		T1Referrals uint64 `json:"t1Referrals"`
		// Consecutive count (no more than 10 hours pause between the mining sessions).
		MaxConsecutiveMiningSessionsCount uint32 `json:"maxConsecutiveMiningSessionsCount"`
		TotalUserReferralPings            uint32 `json:"totalUserReferralPings"`
	}
	AgendaReferralsCount struct {
		UserID               UserID `json:"userId"`
		AgendaReferralsCount uint64 `json:"agendaReferralsCount"`
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}
	// | userSource is a source processor to insert/update user's state at USER_PROGRESS space and to count total users.
	userSource struct {
		r *repository
	}
	// | economyMiningSource is a source processor to count user's consecutive mining sessions.
	economyMiningSource struct {
		r *repository
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Value    interface{}
		Key      string
	}

	// | agendaReferrals  is an internal type to store if users are in agenda for each other (agenda_referrals space).
	agendaReferrals struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack     struct{} `msgpack:",asArray"`
		UserID       UserID
		AgendaUserID UserID
	}

	withCount struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Count    uint64
	}

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var (
	cfg       config
	errNoData = errors.New("empty data")
)

const (
	userProgressSpace    = "USER_PROGRESS"
	agendaReferralsSpace = "AGENDA_REFERRALS"
	// nolint:gomnd // 24 hour is session duration, and up to 10 hours between sessions
	maxTimeBetweenConsecutiveMiningSessions = (24 + 10) * time.Hour

	// Database fields for tarantool oprations, we   keep them in sync with DDL.
	fieldAgendaPhoneNumbersHashes          = 3
	fieldT1Referrals                       = 4
	fieldLastMiningStartedAt               = 1
	fieldMaxConsecutiveMiningSessionsCount = 5
	fieldGlobalValue                       = 0
)
