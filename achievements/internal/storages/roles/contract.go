// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

// Public API.

type (
	RoleName = string

	Repository interface {
	}

	CurrentUserRole struct {
		UpdatedAt time.Time    `json:"updated_at"`
		UserID    users.UserID `json:"user_id"`
		RoleName  `json:"role_name"`
	}
)

// Private API.

const (
	requiredReferralsForPioneerRole    = 1
	requiredReferralsForAmbassadorRole = 100
)

type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}

	// | userProgressSource is a source processor to insert/delete user's roles at CURRENT_USER_ROLES space.
	userProgressSource struct {
		r repository
	}

	currentUserRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{} `msgpack:",asArray"`
		UserID    users.UserID
		RoleName  RoleName
		UpdatedAt uint64
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
var cfg config
