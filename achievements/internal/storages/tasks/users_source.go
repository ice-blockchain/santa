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

func NewUserSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &usersSource{
		r: newRepository(db, mb),
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
		achievedTask := u.getCompletedTask(user)
		if achievedTask != "" {
			err := u.r.AchieveTask(ctx, user.ID, achievedTask)
			if err != nil && !errors.Is(err, ErrAlreadyAchieved) {
				return errors.Wrapf(err, "failed to achieve task %#v for userID:%v", achievedTask, user.ID)
			}
		}
	}

	return nil
}

func (u *usersSource) getCompletedTask(user *users.UserSnapshot) string {
	achievedTask := ""
	// 1. Claim your nickname.
	if user.Username != "" && (user.Before == nil || user.Before.Username == "") {
		achievedTask = taskClaimUsername
	}
	// 3. Upload profile picture.
	hadDefaultPictureBefore := strings.HasSuffix(user.ProfilePictureURL, defaultUserPictureName)
	if !strings.HasSuffix(user.ProfilePictureURL, defaultUserPictureName) && (user.Before == nil || hadDefaultPictureBefore) {
		achievedTask = taskUploadProfilePicture
	}

	return achievedTask
}
