package achievementprocessor

import (
	"context"
	"encoding/json"
	"github.com/ice-blockchain/santa/achievements"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewTaskProcessor(db tarantool.Connector, repository achievements.WriteRepository) messagebroker.Processor {
	return &taskSourceProcessor{db: db, r: repository}
}

func (t *taskSourceProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	task := new(achievements.AchievedTaskMessage)
	if err := json.Unmarshal(message.Value, task); err != nil {
		return errors.Wrapf(err, "taskSourceProcessor: cannot unmarshal %v into %#v", string(message.Value), task)
	}
	// increment user's level for each task completion (Levels -> #7)
	return errors.Wrapf(t.r.IncrementUserLevel(ctx, task.UserID), "taskSourceProcessor: failed to increment user's level for task completion:%#v", task)
}
