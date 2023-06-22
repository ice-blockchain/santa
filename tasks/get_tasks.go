// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"context"

	"github.com/pkg/errors"

	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
)

func (r *repository) GetTasks(ctx context.Context, userID string) (resp []*Task, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	userProgress, err := r.getProgress(ctx, userID, true)
	if err != nil {
		if errors.Is(err, ErrRelationNotFound) {
			return r.defaultTasks(), nil
		}

		return nil, errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}

	return userProgress.buildTasks(r), nil
}

//nolint:revive //.
func (r *repository) getProgress(ctx context.Context, userID string, tolerateOldData bool) (res *progress, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	if tolerateOldData {
		res, err = storage.Get[progress](ctx, r.db, `SELECT * FROM task_progress WHERE user_id = $1`, userID)
	} else {
		res, err = storage.ExecOne[progress](ctx, r.db, `SELECT * FROM task_progress WHERE user_id = $1`, userID)
	}

	err = errors.Wrapf(err, "failed to get TASK_PROGRESS for userID:%v", userID)
	if storage.IsErr(err, storage.ErrNotFound) {
		return nil, ErrRelationNotFound
	}

	return
}

func (p *progress) buildTasks(repo *repository) []*Task { //nolint:gocognit,funlen,revive,gocyclo,cyclop // Wrong.
	resp := repo.defaultTasks()
	for ix, task := range resp {
		switch task.Type { //nolint:exhaustive // Only those 2 have specific data persisted.
		case FollowUsOnTwitterType:
			if p.TwitterUserHandle != nil && *p.TwitterUserHandle != "" {
				task.Data = &Data{
					TwitterUserHandle: *p.TwitterUserHandle,
				}
			}
		case JoinTelegramType:
			if p.TelegramUserHandle != nil && *p.TelegramUserHandle != "" {
				task.Data = &Data{
					TelegramUserHandle: *p.TelegramUserHandle,
				}
			}
		}
		if p.CompletedTasks != nil {
			for _, completedTask := range *p.CompletedTasks {
				if task.Type == completedTask {
					task.Completed = true

					break
				}
			}
		}
		if p.PseudoCompletedTasks != nil && !task.Completed && ix != 0 && resp[ix-1].Completed {
			for _, pseudoCompletedTask := range *p.PseudoCompletedTasks {
				if task.Type == pseudoCompletedTask {
					task.Completed = true

					break
				}
			}
		}
	}

	return resp
}

func (p *progress) reallyCompleted(task *Task) bool {
	if p.CompletedTasks == nil {
		return false
	}
	reallyCompleted := false
	for _, tsk := range *p.CompletedTasks {
		if tsk == task.Type {
			reallyCompleted = true

			break
		}
	}

	return reallyCompleted
}

func (r *repository) defaultTasks() (resp []*Task) {
	resp = make([]*Task, 0, len(&AllTypes))
	for _, taskType := range &AllTypes {
		var (
			data      *Data
			completed bool
		)
		switch taskType { //nolint:exhaustive // We care only about those.
		case ClaimUsernameType:
			completed = true // To make sure network latency doesn't affect UX.
		case InviteFriendsType:
			data = &Data{RequiredQuantity: r.cfg.RequiredFriendsInvited}
		}
		resp = append(resp, &Task{Data: data, Type: taskType, Completed: completed})
	}

	return
}
