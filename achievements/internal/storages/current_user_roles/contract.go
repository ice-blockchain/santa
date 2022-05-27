// SPDX-License-Identifier: BUSL-1.1

package currentuserroles

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
)

// Public API.

type (
	RoleName = string

	Repository interface {
		upsertCurrentUserRole(users.UserID, RoleName) error
		getCurrentUserRole(users.UserID) (string, error)
	}
)

// Private API.

const (
	ambassadorLimit = 100
)

type (
	repository struct {
		db tarantool.Connector
	}

	// | userProgressSource is a source processor to insert/delete user's roles at CURRENT_USER_ROLES space.
	userProgressSource struct {
		r Repository
	}

	currentUserRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{} `msgpack:",asArray"`
		UserID    users.UserID
		RoleName  RoleName
		UpdatedAt uint64
	}

	userRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		RoleName RoleName
	}
)
