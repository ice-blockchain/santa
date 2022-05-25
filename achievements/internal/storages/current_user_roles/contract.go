package roles

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
)

// Public API.
type (
	RoleName = string

	Repository interface {
		InsertCurrentUserRole(users.UserID, RoleName) error
		DeleteCurrentUserRole(users.UserID, RoleName) error
		GetCurrentUserRolesCount(users.UserID) (uint64, error)
	}

	CurrentUserRole struct {
		UpdatedAt time.Time    `json:"createdAt,omitempty" example:"2022-01-03T16:20:52.156534Z"`
		ID        users.UserID `json:"id,omitempty" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
		Name      RoleName
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
	}

	// | userProgressSource is a source processor to insert/delete user's roles at CURRENT_USER_ROLES space
	userProgressSource struct {
		r Repository
	}

	currentUserRole struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack  struct{}     `msgpack:",asArray"`
		UserID    users.UserID `json:"user_id"`
		RoleName  RoleName     `json:"role_name"`
		UpdatedAt uint64       `json:"updated_at"`
	}
)

const (
	currentUserRolesSpace = "current_user_roles"
)
