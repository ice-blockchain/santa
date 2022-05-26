package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db:                        db,
		mb:                        mb,
		publishAchievedTasksTopic: cfg.MessageBroker.Topics[0].Name,
	}
}

func (r *repository) AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}
	now := uint64(time.Now().UTC().UnixNano())
	sql := `INSERT INTO achieved_user_tasks(USER_ID, task_name,   ACHIEVED_AT)
                                                   VALUES(:userID,  :taskName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"taskName":   taskName,
		"achievedAt": now,
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's task %v for userID:%v", taskName, userID)
	}

	return errors.Wrapf(r.sendAchievedTask(ctx, userID, taskName, now), "failed to send achieved task to message broker: %v for userID:%v", taskName, userID)
}

func (r *repository) sendAchievedTask(ctx context.Context, userID UserID, taskName TaskName, achievedTime uint64) error {
	m := AchievedTaskMessage{
		UserID:     userID,
		TaskName:   taskName,
		AchievedAt: achievedTime,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-task] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID + taskName,
		Topic:   r.publishAchievedTasksTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-task] failed to send message to broker")
}
