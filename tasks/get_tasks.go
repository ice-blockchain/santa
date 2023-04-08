// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"context"
	storagev2 "github.com/ice-blockchain/wintr/connectors/storage/v2"

	"github.com/pkg/errors"
)

func (r *repository) GetTasks(ctx context.Context, userID string) (resp []*Task, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	userProgress, err := r.getProgress(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRelationNotFound) {
			return r.defaultTasks(), nil
		}

		return nil, errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	}

	return userProgress.buildTasks(r), nil
}

func (r *repository) getProgress(ctx context.Context, userID string) (res *progress, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	res, err = storagev2.Get[progress](ctx, r.dbV2, `SELECT 
        COALESCE(completed_tasks,'') AS completed_tasks,
		COALESCE(pseudo_completed_tasks, '') AS pseudo_completed_tasks,
		user_id,
		COALESCE(twitter_user_handle, '') AS twitter_user_handle,
		COALESCE(telegram_user_handle, '') AS telegram_user_handle,
		COALESCE(friends_invited, 0) AS friends_invited,
		username_set,
		profile_picture_set,
		COALESCE(mining_started, false) AS mining_started
    FROM task_progress where user_id = $1`, userID)
	err = errors.Wrapf(err, "failed to get TASK_PROGRESS for userID:%v", userID)
	if errors.Is(err, storagev2.ErrNotFound) {
		return nil, ErrRelationNotFound
	}

	return
}

func (p *progress) buildTasks(repo *repository) []*Task { //nolint:gocognit,funlen,revive // Wrong.
	resp := repo.defaultTasks()
	for ix, task := range resp {
		switch task.Type { //nolint:exhaustive // Only those 2 have specific data persisted.
		case FollowUsOnTwitterType:
			if p.TwitterUserHandle != "" {
				task.Data = &Data{
					TwitterUserHandle: p.TwitterUserHandle,
				}
			}
		case JoinTelegramType:
			if p.TelegramUserHandle != "" {
				task.Data = &Data{
					TelegramUserHandle: p.TelegramUserHandle,
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
