package tasks

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/santa/achievements/internal/storages/achievements"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
	"strings"
)

func NewUserSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &usersSource{
		r: New(db, mb),
	}
}

func (u *usersSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user := new(users.UserSnapshot)
	if err := json.Unmarshal(message.Value, user); err != nil {
		return errors.Wrapf(err, "levels/userSource: cannot unmarshall %v into %#v", string(message.Value), user)
	}
	var userAchievementState *achievements.UserAchievements
	userAchievementState = nil // TODO pass value
	achievedTask := u.getCompletedTask(user, userAchievementState)
	//nolint:godox,nolintlint // TODO: think about how to achieve social sharing (endpoint call after sharing?), join twitter, etc.
	if achievedTask != "" {
		if err := u.r.AchieveTask(ctx, user.ID, achievedTask); err != nil {
			return errors.Wrapf(err, "failed to achieve task %#v for userID:%v", achievedTask, user.ID)
		}
	}
	return nil
}

func (u *usersSource) getCompletedTask(user *users.UserSnapshot, userAchievementState *achievements.UserAchievements) string {
	achievedTask := ""
	// 1. Claim your nickname.
	if user.Username != "" && (user.Before == nil || user.Before.Username == "") {
		achievedTask = "TASK1"
	}
	// 3. Upload profile picture.
	hadDefaultPictureBefore := strings.HasSuffix(user.ProfilePictureURL, defaultUserPictureName)
	if !strings.HasSuffix(user.ProfilePictureURL, defaultUserPictureName) && (user.Before == nil || hadDefaultPictureBefore) {
		achievedTask = "TASK3"
	}
	// 6. Invite 5 friends.
	//nolint:godot,nolintlint // FIXME: handle referral deletion, it can downgrade and become 5 again but the task is already achieved
	// Or is the max count of referrals stored in the table, not the current one?
	if userAchievementState.T1Referrals == t1ReferralsToAchieveTask6 {
		achievedTask = "TASK6"
	}

	return achievedTask
}
