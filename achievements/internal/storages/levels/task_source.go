// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"
	"github.com/ice-blockchain/santa/achievements/internal/storages/tasks"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func NewTaskSource(db tarantool.Connector) messagebroker.Processor {
	return &taskSource{r: &repository{db: db}}
}

func (t *taskSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	achievedTask := new(tasks.AchievedTaskMessage)
	if err := json.Unmarshal(message.Value, achievedTask); err != nil {
		return errors.Wrapf(err, "levels/taskSource: cannot unmarshall %v into %#v", string(message.Value), achievedTask)
	}

	// Increment user's level for each task completion (Levels -> #7).
	return errors.Wrapf(t.r.IncrementUserLevel(ctx, achievedTask.UserID), "taskSource: failed to increment user's level for task completion:%#v", achievedTask)
}
