// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

type (
	RoleName   = string
	Repository interface{}

	CurrentUserRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{}     `msgpack:",asArray"`
		UserID    users.UserID `json:"userId"`
		UpdatedAt *time.Time   `json:"updatedAt"`
		RoleName  RoleName     `json:"roleName"`
	}
)

// Private API.

const (
	requiredReferralsForAmbassadorRole = 100
	fieldUpdatedAtSeq                  = 1
	fieldRoleNameSeq                   = 2
)

type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}

	// | userProgressSource is a source processor to insert/delete user's roles at CURRENT_USER_ROLES space.
	userProgressSource struct {
		r *repository
	}

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}

	userRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		RoleName RoleName
	}
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config
