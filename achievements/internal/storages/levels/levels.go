package levels

import (
	"context"
	"fmt"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}

func (r *repository) IncrementUserLevel(ctx context.Context, userID UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}
	nextLevel := `'L'||CAST((SELECT COUNT(1)+1 as next_level FROM achieved_user_levels WHERE user_id = :userID) AS STRING)`
	sql := fmt.Sprintf(`REPLACE INTO achieved_user_levels(USER_ID, LEVEL_NAME,ACHIEVED_AT)
                                                     VALUES( :userID,  %v,       :achievedAt);`, nextLevel)
	params := map[string]interface{}{
		"userID":     userID,
		"achievedAt": uint64(time.Now().UTC().UnixNano()),
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's level for userID:%v", userID)
	}

	return nil
}
