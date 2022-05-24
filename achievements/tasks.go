package achievements

import (
	"context"
	"encoding/json"
	achievementprocessor "github.com/ice-blockchain/santa/achievements/internal/achievement-processor"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
	"time"
)

func (r *repository) AchieveTask(ctx context.Context, userID UserID, task *Task) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}
	now := uint64(time.Now().UTC().UnixNano())
	achievedTaskByUser := &achievedTask{
		UserID:     userID,
		TaskName:   task.Name,
		AchievedAt: now,
	}

	if err := r.db.InsertTyped("achieved_user_tasks", achievedTaskByUser, &[]*achievedTask{}); err != nil {
		return errors.Wrapf(err,
			"failed to achieve task %#v for user %v", task, userID)
	}
	return errors.Wrap(r.sendAchievedTask(ctx, userID, task, now), "failed to send achieved task to message broker")
}

func (r *repository) sendAchievedTask(ctx context.Context, userID UserID, task *Task, achievedTime uint64) error {
	m := achievementprocessor.AchievedTaskMessage{
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
		Topic:   cfg.MessageBroker.Topics[0].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-task] failed to send message to broker")
}
