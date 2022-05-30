// SPDX-License-Identifier: BUSL-1.1

package tasks

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewUsersProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &usersSource{
		r: NewRepository(db, mb),
	}
}

func (u *usersSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "tasks/userSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}
	if user.User != nil {
		completedTasks := u.detectCompletedTasks(user)
		for _, task := range completedTasks {
			if err := u.r.CompleteTask(ctx, user.ID, task); err != nil && !errors.Is(err, errAlreadyAchieved) {
				return errors.Wrapf(err, "failed to complete task %#v for userID:%v", task, user.ID)
			}
		}
	}

	return nil
}

func (u *usersSource) detectCompletedTasks(user *users.UserSnapshot) []string {
	completedTasks := []string{}
	// 1. Claim your nickname.
	if u.isNicknameClaimed(user) {
		completedTasks = append(completedTasks, taskClaimUsername)
	}
	// 3. Upload profile picture.
	if u.isProfilePictureUploaded(user) {
		completedTasks = append(completedTasks, taskUploadProfilePicture)
	}

	return completedTasks
}

func (u *usersSource) isNicknameClaimed(user *users.UserSnapshot) bool {
	return user.Username != "" && (user.Before == nil || user.Before.Username == "")
}

func (u *usersSource) isProfilePictureUploaded(user *users.UserSnapshot) bool {
	defaultUserPictureName := cfg.Tasks.DefaultUserPictureName
	hadDefaultPictureBefore := user.Before != nil && strings.HasSuffix(user.Before.ProfilePictureURL, defaultUserPictureName)

	return (!strings.HasSuffix(user.ProfilePictureURL, defaultUserPictureName)) && (user.Before == nil || hadDefaultPictureBefore)
}
