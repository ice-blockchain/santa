package roles

import (
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/pkg/errors"
)

func NewRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}

func (r *repository) UpsertCurrentUserRole(userID users.UserID, roleName RoleName) error {
	ua := &currentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: uint64(time.Now().UTC().UnixNano()),
	}

	return errors.Wrapf(r.db.UpsertAsync("current_user_roles", ua, nil).GetTyped(&[]*currentUserRole{}),
		"error upserting current user role for userID:%v", userID)
}

func (r *repository) GetCurrentUserRole(userID users.UserID) (string, error) {
	var roleName string

	sql := `SELECT role_name FROM current_user_roles WHERE user_id = :user_id`

	params := map[string]interface{}{"userID": userID}

	if err := r.db.PrepareExecuteTyped(sql, params, &roleName); err != nil {
		return "", errors.Wrapf(err, "failed to get current user role for user.ID:%v", userID)
	}

	return roleName, nil
}
