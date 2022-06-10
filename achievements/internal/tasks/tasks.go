// SPDX-License-Identifier: BUSL-1.1

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
)

func NewRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db: db,
		mb: mb,
	}
}

func (r *repository) CompleteTask(ctx context.Context, userID UserID, taskName TaskName) error {
	if ctx.Err() != nil {
		return errors.Wrapf(ctx.Err(), "failed to achieve a task %v because context failed (for userID:%v)", taskName, userID)
	}
	now := time.Now()

	sql := fmt.Sprintf("INSERT INTO %v (USER_ID, TASK_NAME, ACHIEVED_AT) VALUES (:userID, :taskName, :achievedAt)", achievedTasksSpace)
	params := map[string]interface{}{
		"userID":     userID,
		"taskName":   taskName,
		"achievedAt": now,
	}

	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to complete task for user %v", userID)
	}

	task := &AchievedTask{
		AchievedAt: now,
		UserID:     userID,
		TaskName:   taskName,
	}

	return errors.Wrapf(r.sendCompletedTask(ctx, &CompletedTaskMessage{AchievedTask: *task}),
		"failed to send achieved task to message broker: %v for userID:%v", taskName, userID)
}

func (r *repository) UnCompleteTask(ctx context.Context, userID UserID, taskName TaskName) error {
	if ctx.Err() != nil {
		return errors.Wrapf(ctx.Err(), "failed to achieve a task %v because context failed (for userID:%v)", taskName, userID)
	}
	now := time.Now()
	key := &completedTaskKey{
		UserID:   userID,
		TaskName: taskName,
	}
	_, err := r.getCompletedTask(ctx, userID, taskName)
	if err != nil && errors.Is(err, errNotAchieved) {
		return errors.Wrapf(err, "task %v is not completed for userID %v yet", taskName, userID)
	}
	if err := r.db.DeleteTyped(achievedTasksSpace, "pk_unnamed_ACHIEVED_USER_TASKS_1", key, &[]*AchievedTask{}); err != nil {
		return errors.Wrapf(err, "failed to delete completed user task %v user.ID:%v", taskName, userID)
	}

	return errors.Wrapf(r.sendCompletedTask(ctx, &CompletedTaskMessage{AchievedTask: AchievedTask{
		AchievedAt: now,
		UserID:     userID,
		TaskName:   taskName,
	}, Uncompleted: true}),
		"failed to send achieved task to message broker: %v for userID:%v", taskName, userID)
}

func (r *repository) getCompletedTask(ctx context.Context, userID UserID, taskName TaskName) (*AchievedTask, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrapf(ctx.Err(), "failed to achieve a task %v because context failed (for userID:%v)", taskName, userID)
	}
	key := &completedTaskKey{
		UserID:   userID,
		TaskName: taskName,
	}
	res := new(AchievedTask)
	err := r.db.GetTyped(achievedTasksSpace, "pk_unnamed_ACHIEVED_USER_TASKS_1", key, res)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read completed user's task %v for userID:%v", taskName, userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(errNotAchieved, "task %v is not completed for userID %v yet", taskName, userID)
	}

	return res, nil
}

func (r *repository) sendCompletedTask(ctx context.Context, completedTask *CompletedTaskMessage) error {
	b, err := json.Marshal(completedTask)
	if err != nil {
		return errors.Wrapf(err, "[achieve-task] failed to marshal %#v", completedTask)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     completedTask.UserID,
		Topic:   cfg.MessageBroker.Topics[0].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-task] failed to send message to broker")
}
