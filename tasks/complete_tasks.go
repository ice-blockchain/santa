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
	friendsInvited "github.com/ice-blockchain/santa/friends-invited"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/ice-blockchain/wintr/log"
)

func (r *repository) PseudoCompleteTask(ctx context.Context, task *Task) error { //nolint:funlen,revive // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	userProgress, err := r.getProgress(ctx, task.UserID, true)
	if err != nil && !errors.Is(err, ErrRelationNotFound) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", task.UserID)
	}
	if userProgress == nil {
		userProgress = new(progress)
		userProgress.UserID = task.UserID
	}
	params, sql := userProgress.buildUpdatePseudoCompletedTasksSQL(task, r)
	if params == nil {
		// FE calls endpoint when task "completed", and we have nothing to update
		// so task is completed or pseudo-completed. But FE called endpoint one more time
		// it means there is some lag between pseudo and real-completion, may be msg is messed up,
		// so try to send another one to finally complete it (cuz it is already completed from FE side anyway).
		reallyCompleted := userProgress.reallyCompleted(task)
		if !reallyCompleted {
			return errors.Wrapf(r.sendTryCompleteTasksCommandMessage(ctx, task.UserID), "failed to sendTryCompleteTasksCommandMessage for userID:%v", task.UserID)
		}

		return nil
	}
	if updatedRows, uErr := storage.Exec(ctx, r.db, sql, params...); updatedRows == 0 && uErr == nil {
		return r.PseudoCompleteTask(ctx, task)
	} else if uErr != nil {
		return errors.Wrapf(err, "failed to update task_progress.pseudo_completed_tasks for params:%#v", params...)
	}
	if err = r.sendTryCompleteTasksCommandMessage(ctx, task.UserID); err != nil {
		sErr := errors.Wrapf(err, "failed to sendTryCompleteTasksCommandMessage for userID:%v", task.UserID)
		pseudoCompletedTasks := params[2].(*users.Enum[Type]) //nolint:errcheck,forcetypeassert // We're sure.
		sql = `UPDATE task_progress
			   SET pseudo_completed_tasks = $2
			   WHERE user_id = $1
				 AND COALESCE(pseudo_completed_tasks,ARRAY[]::TEXT[]) = COALESCE($3,ARRAY[]::TEXT[])`
		if updatedRows, rErr := storage.Exec(ctx, r.db, sql, task.UserID, userProgress.PseudoCompletedTasks, pseudoCompletedTasks); rErr == nil && updatedRows == 0 {
			log.Error(errors.Wrapf(sErr, "race condition while rolling back the update request"))

			return r.PseudoCompleteTask(ctx, task)
		} else if rErr != nil {
			return multierror.Append( //nolint:wrapcheck // Not needed.
				sErr,
				errors.Wrapf(rErr, "[rollback] failed to update task_progress.pseudo_completed_tasks for params:%#v", params...),
			).ErrorOrNil()
		}

		return sErr
	}

	return nil
}

func (p *progress) buildUpdatePseudoCompletedTasksSQL(task *Task, repo *repository) (params []any, sql string) { //nolint:funlen // .
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
	params = make([]any, 0)
	params = append(params, task.UserID, p.PseudoCompletedTasks, &pseudoCompletedTasks)
	fieldIndexes := append(make([]string, 0, 1+1), "$3")
	fieldNames := append(make([]string, 0, 1+1), "pseudo_completed_tasks")
	nextIndex := 4
	switch task.Type { //nolint:exhaustive // We only care to treat those differently.
	case FollowUsOnTwitterType:
		params = append(params, task.Data.TwitterUserHandle)
		fieldNames = append(fieldNames, "twitter_user_handle")
		fieldIndexes = append(fieldIndexes, fmt.Sprintf("$%v ", nextIndex))
	case JoinTelegramType:
		params = append(params, task.Data.TelegramUserHandle)
		fieldNames = append(fieldNames, "telegram_user_handle")
		fieldIndexes = append(fieldIndexes, fmt.Sprintf("$%v ", nextIndex))
	}
	sql = fmt.Sprintf(`
			INSERT INTO task_progress(user_id, %[2]v)
			VALUES ($1, %[3]v)
			ON CONFLICT(user_id) DO UPDATE
									SET %[1]v
									WHERE COALESCE(task_progress.pseudo_completed_tasks,ARRAY[]::TEXT[]) = COALESCE($2,ARRAY[]::TEXT[])`,
		strings.Join(formatFields(fieldNames, fieldIndexes), ","),
		strings.Join(fieldNames, ","),
		strings.Join(fieldIndexes, ","),
	)

	return params, sql
}

func formatFields(names, indexes []string) []string {
	res := make([]string, len(names)) //nolint:makezero // We're know for sure.
	for ind, name := range names {
		res[ind] = fmt.Sprintf("%v = %v", name, indexes[ind])
	}

	return res
}

func (r *repository) completeTasks(ctx context.Context, userID string) error { //nolint:revive,funlen,gocognit,gocyclo,cyclop // .
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	pr, err := r.getProgress(ctx, userID, false)
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
	if completedTasks != nil && pr.CompletedTasks != nil {
		log.Debug(fmt.Sprintf("[completeTasks] for userID:%v progress:%#v(%v) completedTasks:%v", userID, *pr, *pr.CompletedTasks, *completedTasks))
	}
	if completedTasks != nil && pr.CompletedTasks != nil && len(*pr.CompletedTasks) == len(*completedTasks) {
		return nil
	}
	sql := `
		INSERT INTO task_progress(user_id, completed_tasks) VALUES ($1, $2)
			ON CONFLICT(user_id) DO UPDATE
				SET completed_tasks = EXCLUDED.completed_tasks
			WHERE COALESCE(task_progress.completed_tasks,ARRAY[]::TEXT[]) = COALESCE($3,ARRAY[]::TEXT[])`
	params := []any{
		pr.UserID,
		completedTasks,
		pr.CompletedTasks,
	}
	if updatedRows, uErr := storage.Exec(ctx, r.db, sql, params...); uErr == nil && updatedRows == 0 {
		return r.completeTasks(ctx, userID)
	} else if uErr != nil {
		return errors.Wrapf(uErr, "failed to insert/update task_progress.completed_tasks for params:%#v", params...)
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
			params[1] = pr.CompletedTasks
			params[2] = completedTasks
			if updatedRows, rErr := storage.Exec(ctx, r.db, sql, params...); rErr == nil && updatedRows == 0 {
				log.Error(errors.Wrapf(sErr, "[sendCompletedTaskMessages]rollback race condition"))

				return r.completeTasks(ctx, userID)
			} else if rErr != nil {
				return multierror.Append( //nolint:wrapcheck // Not needed.
					sErr,
					errors.Wrapf(rErr, "[sendCompletedTaskMessages][rollback] failed to update task_progress.completed_tasks for params:%#v", params...),
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
			if p.TwitterUserHandle != nil && *p.TwitterUserHandle != "" {
				completed = true
			}
		case JoinTelegramType:
			if p.TelegramUserHandle != nil && *p.TelegramUserHandle != "" {
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
	if pr, err := s.getProgress(ctx, userID, true); err != nil && !errors.Is(err, ErrRelationNotFound) ||
		(pr != nil && pr.CompletedTasks != nil && len(*pr.CompletedTasks) == len(&AllTypes)) {
		return errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}
	sql := `INSERT INTO task_progress(user_id, mining_started) VALUES ($1, $2)
			ON CONFLICT(user_id) DO UPDATE SET mining_started = EXCLUDED.mining_started
			WHERE task_progress.mining_started != EXCLUDED.mining_started;`
	_, err := storage.Exec(ctx, s.db, sql, userID, true)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %v %v", userID, true),
		errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, userID),
			"failed to sendTryCompleteTasksCommandMessage for userID:%v", userID),
	).ErrorOrNil()
}

func (s *userTableSource) Process(ctx context.Context, msg *messagebroker.Message) error {
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

	return errors.Wrapf(s.upsertProgress(ctx, snapshot), "failed to upsert progress for:%#v", snapshot)
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
	sql := `INSERT INTO task_progress(user_id, username_set, profile_picture_set) VALUES ($1, $2, $3)
			ON CONFLICT(user_id) DO UPDATE SET 
			        username_set = EXCLUDED.username_set,
			    	profile_picture_set = EXCLUDED.profile_picture_set
			WHERE task_progress.username_set != EXCLUDED.username_set
			   OR task_progress.profile_picture_set != EXCLUDED.profile_picture_set;`
	_, err := storage.Exec(ctx, s.db, sql, insertTuple.UserID, insertTuple.UsernameSet, insertTuple.ProfilePictureSet)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", us),
		errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, us.ID), "failed to sendTryCompleteTasksCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

func (f *friendsInvitedSource) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	friends := new(friendsInvited.Count)
	if err := json.UnmarshalContext(ctx, msg.Value, friends); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), friends)
	}

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(f.updateFriendsInvited(ctx, friends), "failed to update tasks friends invited for %#v", friends),
		errors.Wrapf(f.sendTryCompleteTasksCommandMessage(ctx, friends.UserID), "failed to sendTryCompleteTasksCommandMessage for userID:%v", friends.UserID),
	).ErrorOrNil()
}

func (f *friendsInvitedSource) updateFriendsInvited(ctx context.Context, friends *friendsInvited.Count) error {
	sql := `INSERT INTO task_progress(user_id, friends_invited) VALUES ($1, $2)
		   ON CONFLICT(user_id) DO UPDATE  
		   		SET friends_invited = EXCLUDED.friends_invited
		   	WHERE task_progress.friends_invited != EXCLUDED.friends_invited`
	_, err := storage.Exec(ctx, f.db, sql, friends.UserID, friends.FriendsInvited)

	return errors.Wrapf(err, "failed to set task_progress.friends_invited, params:%#v", friends)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := []any{us.Before.ID}
	_, errDelUser := storage.Exec(ctx, s.db, `DELETE FROM task_progress WHERE user_id = $1`, params...)

	return errors.Wrapf(errDelUser, "failed to delete task_progress for:%#v", us)
}
