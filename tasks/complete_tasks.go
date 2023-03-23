// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/go-tarantool-client"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/log"
)

func (r *repository) PseudoCompleteTask(ctx context.Context, task *Task) error { //nolint:funlen // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	userProgress, err := r.getProgress(ctx, task.UserID)
	if err != nil {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", task.UserID)
	}
	params, sql := userProgress.buildUpdatePseudoCompletedTasksSQL(task, r)
	if params == nil {
		return nil
	}
	if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return r.PseudoCompleteTask(ctx, task)
		}

		return errors.Wrapf(err, "failed to update task_progress.pseudo_completed_tasks for params:%#v", params)
	}
	if err = r.sendTryCompleteTasksCommandMessage(ctx, task.UserID); err != nil {
		sErr := errors.Wrapf(err, "failed to sendTryCompleteTasksCommandMessage for userID:%v", task.UserID)
		pseudoCompletedTasks := params["pseudo_completed_tasks"].(*users.Enum[Type]) //nolint:errcheck,forcetypeassert // We know for sure.
		params = make(map[string]any, 1+1+1)
		params["user_id"] = task.UserID
		params["pseudo_completed_tasks"] = userProgress.PseudoCompletedTasks
		params["old_pseudo_completed_tasks"] = pseudoCompletedTasks
		sql = `UPDATE task_progress
			   SET pseudo_completed_tasks = :pseudo_completed_tasks
			   WHERE user_id = :user_id
				 AND IFNULL(pseudo_completed_tasks,'') = IFNULL(:old_pseudo_completed_tasks,'')`
		if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Error(errors.Wrapf(sErr, "race condition while rolling back the update request"))

				return r.PseudoCompleteTask(ctx, task)
			}

			return multierror.Append( //nolint:wrapcheck // Not needed.
				sErr,
				errors.Wrapf(err, "[rollback] failed to update task_progress.pseudo_completed_tasks for params:%#v", params),
			).ErrorOrNil()
		}

		return sErr
	}

	return nil
}

func (p *progress) buildUpdatePseudoCompletedTasksSQL(task *Task, repo *repository) (params map[string]any, sql string) { //nolint:funlen // .
	for _, tsk := range p.buildTasks(repo) {
		if tsk.Type == task.Type {
			if tsk.Completed {
				return nil, ""
			}
		}
	}
	pseudoCompletedTasks := make(users.Enum[Type], 0, len(&AllTypes))
	if p.PseudoCompletedTasks != nil {
		pseudoCompletedTasks = append(pseudoCompletedTasks, *p.PseudoCompletedTasks...)
	}
	pseudoCompletedTasks = append(pseudoCompletedTasks, task.Type)
	sort.SliceStable(pseudoCompletedTasks, func(i, j int) bool {
		return TypeOrder[pseudoCompletedTasks[i]] < TypeOrder[pseudoCompletedTasks[j]]
	})
	params = make(map[string]any, 1+1+1+1)
	params["user_id"] = task.UserID
	params["pseudo_completed_tasks"] = &pseudoCompletedTasks
	params["old_pseudo_completed_tasks"] = p.PseudoCompletedTasks
	fields := append(make([]string, 0, 1+1), "pseudo_completed_tasks = :pseudo_completed_tasks ")
	switch task.Type { //nolint:exhaustive // We only care to treat those differently.
	case FollowUsOnTwitterType:
		params["twitter_user_handle"] = task.Data.TwitterUserHandle
		fields = append(fields, "twitter_user_handle = :twitter_user_handle ")
	case JoinTelegramType:
		params["telegram_user_handle"] = task.Data.TelegramUserHandle
		fields = append(fields, "telegram_user_handle = :telegram_user_handle ")
	}
	sql = fmt.Sprintf(`UPDATE task_progress
						SET %v
						WHERE user_id = :user_id
						  AND IFNULL(pseudo_completed_tasks,'') = IFNULL(:old_pseudo_completed_tasks,'')`, strings.Join(fields, ","))

	return params, sql
}

func (r *repository) completeTasks(ctx context.Context, userID string) error { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	pr, err := r.getProgress(ctx, userID)
	if err != nil && !errors.Is(err, ErrRelationNotFound) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	if pr == nil {
		pr = new(progress)
		pr.UserID = userID
	}
	if pr.CompletedTasks != nil && len(*pr.CompletedTasks) == len(&AllTypes) {
		return nil
	}
	completedTasks := pr.reEvaluateCompletedTasks(r)
	if completedTasks != nil && pr.CompletedTasks != nil && len(*pr.CompletedTasks) == len(*completedTasks) {
		return nil
	}
	sql := `UPDATE task_progress
			SET completed_tasks = :completed_tasks
			WHERE user_id = :user_id
              AND IFNULL(completed_tasks,'') = IFNULL(:old_completed_tasks,'')`
	params := make(map[string]any, 1+1+1)
	params["user_id"] = pr.UserID
	params["completed_tasks"] = completedTasks
	params["old_completed_tasks"] = pr.CompletedTasks
	//nolint:nestif // .
	if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			tuple := &progress{CompletedTasks: completedTasks, UserID: pr.UserID}
			if err = storage.CheckNoSQLDMLErr(r.db.InsertTyped("TASK_PROGRESS", tuple, &[]*progress{})); err != nil && errors.Is(err, storage.ErrDuplicate) {
				return r.completeTasks(ctx, userID)
			}
			if err != nil {
				return errors.Wrapf(err, "failed to insert TASK_PROGRESS %#v", tuple)
			}
		}
		if err != nil {
			return errors.Wrapf(err, "failed to update task_progress.completed_tasks for params:%#v", params)
		}
	}
	//nolint:nestif // .
	if completedTasks != nil && len(*completedTasks) > 0 && (pr.CompletedTasks == nil || len(*pr.CompletedTasks) < len(*completedTasks)) {
		newlyCompletedTasks := make([]*CompletedTask, 0, len(&AllTypes))
	outer:
		for _, completedTask := range *completedTasks {
			if pr.CompletedTasks != nil {
				for _, previouslyCompletedTask := range *pr.CompletedTasks {
					if completedTask == previouslyCompletedTask {
						continue outer
					}
				}
			}
			newlyCompletedTasks = append(newlyCompletedTasks, &CompletedTask{
				UserID:         userID,
				Type:           completedTask,
				CompletedTasks: uint64(len(*completedTasks)),
			})
		}
		if err = runConcurrently(ctx, r.sendCompletedTaskMessage, newlyCompletedTasks); err != nil {
			sErr := errors.Wrapf(err, "failed to sendCompletedTaskMessages for userID:%v,completedTasks:%#v", userID, newlyCompletedTasks)
			params["completed_tasks"] = pr.CompletedTasks
			params["old_completed_tasks"] = completedTasks
			if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Error(errors.Wrapf(sErr, "[sendCompletedTaskMessages]rollback race condition"))

					return r.completeTasks(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[sendCompletedTaskMessages][rollback] failed to update task_progress.completed_tasks for params:%#v", params),
				).ErrorOrNil()
			}

			return sErr
		}
	}
	//nolint:nestif // .
	if completedTasks != nil && len(*completedTasks) == len(&AllTypes) && (pr.CompletedTasks == nil || len(*pr.CompletedTasks) < len(*completedTasks)) {
		if err = r.awardAllTasksCompletionICECoinsBonus(ctx, userID); err != nil {
			sErr := errors.Wrapf(err, "failed to awardAllTasksCompletionICECoinsBonus for userID:%v", userID)
			params["completed_tasks"] = pr.CompletedTasks
			params["old_completed_tasks"] = completedTasks
			if err = storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Error(errors.Wrapf(sErr, "[awardAllTasksCompletionICECoinsBonus]rollback race condition"))

					return r.completeTasks(ctx, userID)
				}

				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(err, "[awardAllTasksCompletionICECoinsBonus][rollback] failed to update task_progress.completed_tasks, params:%#v", params),
				).ErrorOrNil()
			}

			return sErr
		}
	}

	return nil
}

func (p *progress) reEvaluateCompletedTasks(repo *repository) *users.Enum[Type] { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if p.CompletedTasks != nil && len(*p.CompletedTasks) == len(&AllTypes) {
		return p.CompletedTasks
	}
	alreadyCompletedTasks := make(map[Type]any, len(&AllTypes))
	if p.CompletedTasks != nil {
		for _, task := range *p.CompletedTasks {
			alreadyCompletedTasks[task] = struct{}{}
		}
	}
	completedTasks := make(users.Enum[Type], 0, len(&AllTypes))
	for ix, taskType := range &AllTypes {
		if _, alreadyCompleted := alreadyCompletedTasks[taskType]; alreadyCompleted {
			completedTasks = append(completedTasks, taskType)

			continue
		}
		if len(completedTasks) != ix {
			break
		}
		var completed bool
		switch taskType {
		case ClaimUsernameType:
			completed = p.UsernameSet
		case StartMiningType:
			completed = p.MiningStarted
		case UploadProfilePictureType:
			completed = p.ProfilePictureSet
		case FollowUsOnTwitterType:
			if p.TwitterUserHandle != "" {
				completed = true
			}
		case JoinTelegramType:
			if p.TelegramUserHandle != "" {
				completed = true
			}
		case InviteFriendsType:
			if p.FriendsInvited >= repo.cfg.RequiredFriendsInvited {
				completed = true
			}
		}
		if completed {
			completedTasks = append(completedTasks, taskType)
		}
	}
	if len(completedTasks) == 0 {
		return nil
	}

	return &completedTasks
}

func (r *repository) awardAllTasksCompletionICECoinsBonus(ctx context.Context, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	type (
		AddBalanceCommand struct {
			BaseFactor string `json:"baseFactor,omitempty" example:"1243"`
			UserID     string `json:"userId,omitempty" example:"some unique id"`
			EventID    string `json:"eventId,omitempty" example:"some unique id"`
		}
	)
	cmd := &AddBalanceCommand{
		BaseFactor: fmt.Sprint(allTasksCompletionBaseMiningRatePrizeFactor),
		UserID:     userID,
		EventID:    "all_tasks_completion_ice_bonus",
	}
	valueBytes, err := json.MarshalContext(ctx, cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", cmd)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     cmd.UserID,
		Topic:   r.cfg.MessageBroker.ProducingTopics[0].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (r *repository) sendCompletedTaskMessage(ctx context.Context, completedTask *CompletedTask) error {
	valueBytes, err := json.MarshalContext(ctx, completedTask)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %#v", completedTask)
	}
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     completedTask.UserID,
		Topic:   r.cfg.MessageBroker.Topics[2].Name,
		Value:   valueBytes,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (r *repository) sendTryCompleteTasksCommandMessage(ctx context.Context, userID string) error {
	msg := &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   r.cfg.MessageBroker.Topics[1].Name,
	}
	responder := make(chan error, 1)
	defer close(responder)
	r.mb.SendMessage(ctx, msg, responder)

	return errors.Wrapf(<-responder, "failed to send `%v` message to broker", msg.Topic)
}

func (s *tryCompleteTasksCommandSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}

	return errors.Wrapf(s.completeTasks(ctx, msg.Key), "failed to completeTasks for userID:%v", msg.Key)
}

func (s *miningSessionSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	var ses struct {
		UserID string `json:"userId,omitempty" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
	}
	if err := json.UnmarshalContext(ctx, msg.Value, &ses); err != nil {
		return errors.Wrapf(err, "process: cannot unmarshall %v into %#v", string(msg.Value), &ses)
	}
	if ses.UserID == "" {
		return nil
	}

	return errors.Wrapf(s.upsertProgress(ctx, ses.UserID), "failed to upsertProgress for userID:%v", ses.UserID)
}

func (s *miningSessionSource) upsertProgress(ctx context.Context, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if pr, err := s.getProgress(ctx, userID); err != nil && !errors.Is(err, ErrRelationNotFound) ||
		(pr != nil && pr.CompletedTasks != nil && len(*pr.CompletedTasks) == len(&AllTypes)) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	const miningStartedColumnIndex = 8
	insertTuple := &progress{UserID: userID, MiningStarted: true}
	ops := append(make([]tarantool.Op, 0, 1), tarantool.Op{Op: "=", Field: miningStartedColumnIndex, Arg: insertTuple.MiningStarted})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("TASK_PROGRESS", insertTuple, ops, &[]*progress{})),
			"failed to upsert progress for %#v", insertTuple),
		errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, userID),
			"failed to sendTryCompleteTasksCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *userTableSource) Process(ctx context.Context, msg *messagebroker.Message) error { //nolint:gocognit // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	snapshot := new(users.UserSnapshot)
	if err := json.UnmarshalContext(ctx, msg.Value, snapshot); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), snapshot)
	}
	if (snapshot.Before == nil || snapshot.Before.ID == "") && (snapshot.User == nil || snapshot.User.ID == "") {
		return nil
	}
	if snapshot.Before != nil && snapshot.Before.ID != "" && (snapshot.User == nil || snapshot.User.ID == "") {
		return errors.Wrapf(s.deleteProgress(ctx, snapshot), "failed to delete progress for:%#v", snapshot)
	}
	if err := s.upsertProgress(ctx, snapshot); err != nil {
		return errors.Wrapf(err, "failed to upsert progress for:%#v", snapshot)
	}

	return nil
}

func (s *userTableSource) upsertProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	insertTuple := &progress{
		UserID:            us.ID,
		UsernameSet:       us.Username != "" && us.Username != us.ID,
		ProfilePictureSet: us.ProfilePictureURL != "" && !strings.Contains(us.ProfilePictureURL, "default-profile-picture"),
	}
	const usernameSetColumnIndex, profilePictureSetColumnIndex = 6, 7
	ops := append(make([]tarantool.Op, 0, 1+1),
		tarantool.Op{Op: "=", Field: usernameSetColumnIndex, Arg: insertTuple.UsernameSet},
		tarantool.Op{Op: "=", Field: profilePictureSetColumnIndex, Arg: insertTuple.ProfilePictureSet})

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.UpsertTyped("TASK_PROGRESS", insertTuple, ops, &[]*progress{})), "failed to upsert progress for %#v", us),
		errors.Wrapf(s.updateFriendsInvited(ctx, us), "failed to updateFriendsInvited for user:%#v", us),
		errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, us.ID), "failed to sendTryCompleteTasksCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

//nolint:gocognit // .
func (s *userTableSource) updateFriendsInvited(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil || us.User == nil || us.User.ReferredBy == "" || us.User.ReferredBy == us.User.ID || (us.Before != nil && us.Before.ID != "" && us.User.ReferredBy == us.Before.ReferredBy) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `REPLACE INTO referrals(user_id,referred_by) VALUES (:user_id,:referred_by)`
	params := make(map[string]any, 1+1)
	params["user_id"] = us.User.ID
	params["referred_by"] = us.User.ReferredBy
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		return errors.Wrapf(err, "failed to REPLACE INTO referrals, params:%#v", params)
	}
	delete(params, "user_id")
	sql = `UPDATE task_progress 
		   SET friends_invited = (SELECT COUNT(*) FROM referrals WHERE referred_by = :referred_by)
		   WHERE user_id = :referred_by`
	if err := storage.CheckSQLDMLErr(s.db.PrepareExecute(sql, params)); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = errors.Wrapf(storage.CheckNoSQLDMLErr(s.db.InsertTyped("TASK_PROGRESS", &progress{UserID: us.User.ReferredBy, FriendsInvited: 1}, &[]*progress{})), "failed to insert progress for referredBy:%v, from :%#v", us.User.ReferredBy, us) //nolint:lll // .
		}
		if err != nil && !errors.Is(err, storage.ErrDuplicate) {
			return errors.Wrapf(err, "failed to set task_progress.friends_invited, params:%#v", params)
		}
	}

	return errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, us.User.ReferredBy),
		"failed to sendTryCompleteTasksCommandMessage, userID:%v,referredBy:%v", us.User.ID, us.User.ReferredBy)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := map[string]any{"user_id": us.Before.ID}

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM task_progress WHERE user_id = :user_id`, params)),
			"failed to delete task_progress for:%#v", us),
		errors.Wrapf(storage.CheckSQLDMLErr(s.db.PrepareExecute(`DELETE FROM referrals WHERE user_id = :user_id`, params)),
			"failed to delete referrals for:%#v", us),
	).ErrorOrNil()
}
