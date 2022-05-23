// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"github.com/ice-blockchain/santa/achievements/internal"
	"github.com/ice-blockchain/santa/achievements/internal/storages/tasks"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func NewTaskSource(db tarantool.Connector) internal.AchievedTasksSource {
	return &taskSource{r: &repository{db: db}}
}

func (t *taskSource) ProcessAchievedTask(ctx context.Context, userID UserID, task *tasks.AchievedTaskMessage) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	// Increment user's level for each task completion (Levels -> #7).
	return errors.Wrapf(t.r.IncrementUserLevel(ctx, task.UserID), "taskSource: failed to increment user's level for task completion:%#v", task)
}
