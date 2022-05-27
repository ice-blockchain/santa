// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}

func (r *repository) achieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve level failed because context failed")
	}
	now := uint64(time.Now().UTC().UnixNano())
	sql := `INSERT INTO achieved_user_levels(USER_ID, LEVEL_NAME,   ACHIEVED_AT)
                                                   VALUES(:userID,  :levelName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"levelName":  levelName,
		"achievedAt": now,
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's level for userID:%v", userID)
	}

	return nil
}
