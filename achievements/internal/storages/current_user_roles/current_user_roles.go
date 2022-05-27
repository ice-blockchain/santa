// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}

func (r *repository) upsertCurrentUserRole(userID users.UserID, roleName RoleName) error {
	cur := &currentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: uint64(time.Now().UTC().UnixNano()),
	}

	return errors.Wrapf(r.db.UpsertAsync("CURRENT_USER_ROLES", cur, []tarantool.Op{}).GetTyped(&[]currentUserRole{}),
		"error upserting current user role for userID:%v", userID)
}

func (r *repository) getCurrentUserRole(userID users.UserID) (string, error) {
	var res []*userRole
	sql := `SELECT role_name FROM current_user_roles WHERE user_id = :user_id`

	params := map[string]interface{}{"user_id": userID}

	if err := r.db.PrepareExecuteTyped(sql, params, &res); err != nil {
		return "", errors.Wrapf(err, "failed to get current user role for user.ID:%v", userID)
	}

	if len(res) == 0 {
		return "", nil
	}

	return res[0].RoleName, nil
}
