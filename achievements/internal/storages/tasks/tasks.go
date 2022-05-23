package tasks

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"time"
)

func New(db tarantool.Connector, mb messagebroker.Client) Repository {
	var config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
	appCfg.MustLoadFromKey("achievements", &config)
	return &repository{
		db:                        db,
		mb:                        mb,
		publishAchievedTasksTopic: config.MessageBroker.Topics[0].Name,
	}
}
func (r *repository) AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}

	// Check if such task exists before achieve it.
	taskObject, err := r.GetTask(ctx, taskName)
	if err != nil {
		return errors.Wrapf(err, "failed to read task for taskName:%v", taskName)
	}

	now := uint64(time.Now().UTC().UnixNano())
	achievedTaskByUser := &achievedTask{
		UserID:     userID,
		TaskName:   taskObject.Name,
		AchievedAt: now,
	}

	if err := r.db.InsertTyped("ACHIEVED_USER_TASKS", achievedTaskByUser, &[]*achievedTask{}); err != nil {
		return errors.Wrapf(err,
			"failed to achieve task %v for user %v", taskName, userID)
	}

	return errors.Wrapf(r.sendAchievedTask(ctx, userID, taskObject, now), "failed to send achieved task to message broker: %#v", taskObject)
}

func (r *repository) sendAchievedTask(ctx context.Context, userID UserID, task *task, achievedTime uint64) error {
	m := AchievedTaskMessage{
		UserID:     userID,
		TaskName:   task.Name,
		TaskIndex:  task.Index,
		AchievedAt: achievedTime,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-task] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   r.publishAchievedTasksTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-task] failed to send message to broker")
}

func (r *repository) GetTask(ctx context.Context, taskName TaskName) (*task, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "get task failed because context failed")
	}
	var res task
	if err := r.db.GetTyped(tasksSpace, "pk_unnamed_TASKS_1", tarantool.StringKey{S: taskName}, &res); err != nil {
		return nil, errors.Wrapf(err, "unable to get tasks record for taskName:%v", taskName)
	}
	if res.Name == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no task record for name:%v", taskName)
	}

	return &res, nil
}

func (t *task) Task() *Task {
	return &Task{
		Name:  t.Name,
		Index: t.Index,
	}
}
