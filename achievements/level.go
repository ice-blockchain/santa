package achievements

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func (r *repository) IncrementUserLevel(ctx context.Context, userID UserID) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}
	key := tarantool.StringKey{S: userID}
	incrementOps := []tarantool.Op{
		{Op: "+", Field: 3 /* TODO: Const? Field could move if DDL'll be changed, we need to sync it with DDL */, Arg: 1},
	}

	return errors.Wrapf(r.db.UpdateTyped(userAchievementsSpace, "pk_unnamed_USER_ACHIEVEMENTS_1", key, incrementOps, &[]*userAchievements{}),
		"failed to update %v record with the new level value userID:%v", userAchievementsSpace, userID)
}
