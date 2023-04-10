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
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
)

func (r *repository) PseudoCompleteTask(ctx context.Context, task *Task) error { //nolint:funlen,gocognit // .
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
	err = storage.DoInTransaction(ctx, r.db, func(conn storage.QueryExecer) error {
		var updatedRows uint64
		if updatedRows, err = storage.Exec(ctx, r.db, sql, params...); err != nil || updatedRows == 0 {
			if updatedRows == 0 || errors.Is(err, storage.ErrNotFound) {
				return ErrRaceCondition
			}

			return errors.Wrapf(err, "failed to update task_progress.pseudo_completed_tasks for params:%#v", params...)
		}
		if err = r.sendTryCompleteTasksCommandMessage(ctx, task.UserID); err != nil {
			return errors.Wrapf(err, "failed to sendTryCompleteTasksCommandMessage for userID:%v", task.UserID)
		}

		return nil
	})
	if err != nil && errors.Is(err, ErrRaceCondition) {
		return r.PseudoCompleteTask(ctx, task)
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
	fields := append(make([]string, 0, 1+1), "pseudo_completed_tasks = $3 ")
	nextIndex := 4
	switch task.Type { //nolint:exhaustive // We only care to treat those differently.
	case FollowUsOnTwitterType:
		params = append(params, task.Data.TwitterUserHandle)
		fields = append(fields, fmt.Sprintf("twitter_user_handle = $%v ", nextIndex))
	case JoinTelegramType:
		params = append(params, task.Data.TelegramUserHandle)
		fields = append(fields, fmt.Sprintf("telegram_user_handle = $%v ", nextIndex))
	}
	sql = fmt.Sprintf(`UPDATE task_progress
						SET %v
						WHERE user_id = $1
						  AND COALESCE(pseudo_completed_tasks,'') = COALESCE($2,'')`, strings.Join(fields, ","))

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
	sql := `
		INSERT INTO task_progress(user_id, completed_tasks) VALUES ($1, $2)
			ON CONFLICT(user_id) DO UPDATE
				SET completed_tasks = EXCLUDED.completed_tasks
			WHERE COALESCE(task_progress.completed_tasks,'') = COALESCE($3,'')`
	params := []any{
		pr.UserID,
		completedTasks,
		pr.CompletedTasks,
	}
	err = storage.DoInTransaction(ctx, r.db, func(conn storage.QueryExecer) error {
		if updatedRows, uErr := storage.Exec(ctx, conn, sql, params...); uErr != nil || updatedRows == 0 {
			if updatedRows == 0 || errors.Is(uErr, storage.ErrNotFound) {
				return ErrRaceCondition
			}

			return errors.Wrapf(uErr, "failed to insert/update task_progress.completed_tasks for params:%#v", params...)
		}
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
				return errors.Wrapf(err, "failed to sendCompletedTaskMessages for userID:%v,completedTasks:%#v", userID, newlyCompletedTasks)
			}
		}
		if completedTasks != nil && len(*completedTasks) == len(&AllTypes) && (pr.CompletedTasks == nil || len(*pr.CompletedTasks) < len(*completedTasks)) {
			if err = r.awardAllTasksCompletionICECoinsBonus(ctx, userID); err != nil {
				return errors.Wrapf(err, "failed to awardAllTasksCompletionICECoinsBonus for userID:%v", userID)
			}
		}

		return nil
	})
	if err != nil && errors.Is(err, ErrRaceCondition) {
		return r.completeTasks(ctx, userID)
	}

	return errors.Wrapf(err, "failed to execute transaction to complete new tasks for userID:%v", userID)
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
	sql := `INSERT INTO task_progress(user_id, mining_started) VALUES ($1, $2)
			ON CONFLICT(user_id) DO UPDATE SET mining_started = true;`
	_, err := storage.Exec(ctx, s.db, sql, userID, true)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %v %v", userID, true),
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
	sql := `INSERT INTO task_progress(user_id, username_set, profile_picture_set) VALUES ($1, $2, $3)
			ON CONFLICT(user_id) DO UPDATE SET username_set = $2, profile_picture_set = $3;`
	_, err := storage.Exec(ctx, s.db, sql, insertTuple.UserID, insertTuple.UsernameSet, insertTuple.ProfilePictureSet)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(err, "failed to upsert progress for %#v", us),
		errors.Wrapf(s.updateFriendsInvited(ctx, us), "failed to updateFriendsInvited for user:%#v", us),
		errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, us.ID), "failed to sendTryCompleteTasksCommandMessage for userID:%v", us.ID),
	).ErrorOrNil()
}

//nolint:gocognit // .
func (s *userTableSource) updateFriendsInvited(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil || us.User == nil || us.User.ReferredBy == "" || us.User.ReferredBy == us.User.ID || (us.Before != nil && us.Before.ID != "" && us.User.ReferredBy == us.Before.ReferredBy) { //nolint:lll,revive // .
		return errors.Wrap(ctx.Err(), "context failed")
	}
	sql := `INSERT INTO referrals(user_id,referred_by) VALUES ($1,$2)
				ON CONFLICT(user_id) DO UPDATE
				SET referred_by = $2`
	params := []any{
		us.User.ID,
		us.User.ReferredBy,
	}
	if _, err := storage.Exec(ctx, s.db, sql, params...); err != nil {
		return errors.Wrapf(err, "failed to REPLACE INTO referrals, params:%#v", params...)
	}
	sql = `INSERT INTO task_progress(user_id, friends_invited) VALUES ($1, (SELECT COUNT(*) FROM referrals WHERE referred_by = $1))
		   ON CONFLICT(user_id) DO UPDATE  
		   		SET friends_invited = EXCLUDED.friends_invited`
	if _, err := storage.Exec(ctx, s.db, sql, us.User.ReferredBy); err != nil {
		return errors.Wrapf(err, "failed to set task_progress.friends_invited, params:%#v", params...)
	}

	return errors.Wrapf(s.sendTryCompleteTasksCommandMessage(ctx, us.User.ReferredBy),
		"failed to sendTryCompleteTasksCommandMessage, userID:%v,referredBy:%v", us.User.ID, us.User.ReferredBy)
}

func (s *userTableSource) deleteProgress(ctx context.Context, us *users.UserSnapshot) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	params := []any{us.Before.ID}
	_, errDelUser := storage.Exec(ctx, s.db, `DELETE FROM task_progress WHERE user_id = $1`, params...)
	_, errDelRefs := storage.Exec(ctx, s.db, `DELETE FROM referrals WHERE user_id = $1 OR referred_by = $1`, params...)

	return multierror.Append( //nolint:wrapcheck // Not needed.
		errors.Wrapf(errDelUser, "failed to delete task_progress for:%#v", us),
		errors.Wrapf(errDelRefs, "failed to delete referrals for:%#v", us),
	).ErrorOrNil()
}
