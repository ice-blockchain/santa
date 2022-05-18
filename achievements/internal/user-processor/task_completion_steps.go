package userprocessor

import (
	"github.com/ice-blockchain/eskimo/users"
)

var userRelatedTasks = []TaskCompletionStep{
	{
		Condition: func(input interface{}) bool { // 1. Claim your nickname
			return input.(users.User).Username != ""
		},
		AchievedTaskName: "TASK1",
	},
	{
		Condition: func(input interface{}) bool { // 3. Upload profile picture
			return input.(users.User).ProfilePictureURL != ""
		},
		AchievedTaskName: "TASK3",
	},
	// 6. Invite 5 friends seems to be here too (we'll need to read usersAchievements and t1 count from there)
}
