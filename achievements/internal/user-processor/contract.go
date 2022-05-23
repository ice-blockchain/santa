package userprocessor

import (
	"context"
	"strings"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
)

type (
	UserID          = string
	BadgeName       = string
	TaskName        = string
	WriteRepository interface {
		AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
		AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error
		IncrementUserLevel(ctx context.Context, userID UserID) error
	}

	userSourceProcessor struct {
		r  WriteRepository
		db tarantool.Connector
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		// For now we're saving only integer, but scalar may be one of
		// boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary,
		// but I cant find golang mapping in docs (interface{}?).
		Value uint64
	}

	// `userAchievements` is an internal type to store user achievements badges in database (USER_ACHIEVEMENTS space).
	userAchievements struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// User's role, one of PIONEER, ABMASSADOR, ...3rd...
		Role string
		// User's balance (ICE).
		Balance uint64
		Level   uint32
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
	}
)

// nolint:gocognit // We're using conditions here to check if user achieved a new task.
func (u *userSourceProcessor) getCompletedTask(user *users.UserSnapshot, userAchievementState *userAchievements) string {
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

const (
	userAchievementsSpace     = "USER_ACHIEVEMENTS"
	t1ReferralsToAchieveTask6 = 5
	defaultUserPictureName    = "default-user-image.jpg"
	fieldT1Referrals          = 4
)
