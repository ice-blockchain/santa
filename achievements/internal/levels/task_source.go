// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/tasks"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewCompletedTaskProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &taskSource{
		r: newRepository(db, mb).(*repository),
	}
}

func (t *taskSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	completedTask := new(tasks.CompletedTaskMessage)
	if err := json.Unmarshal(message.Value, completedTask); err != nil {
		return errors.Wrapf(err, "levels/taskSource: cannot unmarshall %v into %#v", string(message.Value), completedTask)
	}

	// Increment user's level for each task completion (Levels -> #7).
	return errors.Wrapf(t.achieveLevelsForTaskCompletion(ctx, completedTask),
		"levels/taskSource: failed to increment user's level for task completion:%#v", completedTask)
}

func (t *taskSource) achieveLevelsForTaskCompletion(ctx context.Context, completedTask *tasks.CompletedTaskMessage) error {
	levelName := cfg.Levels.TaskCompletion[completedTask.TaskName]
	if completedTask.Uncompleted {
		if err := t.r.unachieveUserLevel(ctx, completedTask.UserID, levelName); err != nil && !errors.Is(err, errNotAchieved) {
			return errors.Wrapf(err,
				"levels/taskSource: failed to unachieve user's level for task completion:%#v", completedTask)
		}
	} else {
		if err := t.r.achieveUserLevel(ctx, completedTask.UserID, levelName); err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err,
				"levels/taskSource: failed to achieve user's level for task completion:%#v", completedTask)
		}
	}

	return nil
}
