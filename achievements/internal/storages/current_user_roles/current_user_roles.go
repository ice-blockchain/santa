package roles

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func NewRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}

func (r *repository) InsertCurrentUserRole(userID users.UserID, roleName RoleName) error {
	ua := &currentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: uint64(time.Now().UTC().UnixNano()),
	}

	return errors.Wrapf(r.db.InsertTyped(currentUserRolesSpace, ua, &[]*currentUserRole{}),
		"failed to insert current user role for user.ID:%v", userID)
}

func (r *repository) DeleteCurrentUserRole(userID users.UserID, roleName RoleName) error {
	sql := `DELETE FROM current_user_roles WHERE user_id = :userID AND roleName = :roleName`

	params := map[string]interface{}{
		"userID":   userID,
		"roleName": roleName,
	}

	return errors.Wrapf(storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)),
		"failed to delete current user role for user.ID:%v", userID)
}
