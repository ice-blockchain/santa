// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/eskimo/users"
)

func TestReEvaluateCompletedTasks(t *testing.T) { //nolint:funlen // It's a test function
	t.Parallel()

	completedTasks := make(users.Enum[Type], 0, len(&AllTypes))
	for _, taskType := range &AllTypes {
		completedTasks = append(completedTasks, taskType)
	}
	bogusUsername := "bogus" //nolint:goconst // .
	testCases := []struct {
		p        *progress
		repo     *repository
		expected *users.Enum[Type]
		name     string
	}{
		{
			name: "all tasks are completed",
			p: &progress{
				CompletedTasks: &completedTasks,
			},
			expected: &completedTasks,
		},
		{
			name: "skip already completed tasks",
			p: &progress{
				CompletedTasks: &users.Enum[Type]{ClaimUsernameType, StartMiningType},
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType},
		},
		{
			name: "claim username task is completed",
			p: &progress{
				UsernameSet: true,
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType},
		},
		{
			name: "start mining task is completed",
			p: &progress{
				UsernameSet:   true,
				MiningStarted: true,
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType},
		},
		{
			name: "upload profile picture task is completed",
			p: &progress{
				UsernameSet:       true,
				MiningStarted:     true,
				ProfilePictureSet: true,
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType, UploadProfilePictureType},
		},
		{
			name: "follow us on twitter task is completed",
			p: &progress{
				UsernameSet:       true,
				MiningStarted:     true,
				ProfilePictureSet: true,
				TwitterUserHandle: &bogusUsername,
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType, UploadProfilePictureType, FollowUsOnTwitterType},
		},
		{
			name: "join telegram task is completed",
			p: &progress{
				UsernameSet:        true,
				MiningStarted:      true,
				ProfilePictureSet:  true,
				TwitterUserHandle:  &bogusUsername,
				TelegramUserHandle: &bogusUsername,
			},
			repo:     &repository{cfg: &config{RequiredFriendsInvited: 1}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType, UploadProfilePictureType, FollowUsOnTwitterType, JoinTelegramType},
		},
		{
			name: "invite friends task is completed",
			p: &progress{
				UsernameSet:        true,
				MiningStarted:      true,
				ProfilePictureSet:  true,
				TwitterUserHandle:  &bogusUsername,
				TelegramUserHandle: &bogusUsername,
				FriendsInvited:     2,
			},
			repo:     &repository{cfg: &config{RequiredFriendsInvited: 1}},
			expected: &users.Enum[Type]{ClaimUsernameType, StartMiningType, UploadProfilePictureType, FollowUsOnTwitterType, JoinTelegramType, InviteFriendsType},
		},
		{
			name: "upload profile picture is completed, but start mining isn't",
			p: &progress{
				UsernameSet:       true,
				ProfilePictureSet: true,
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[Type]{ClaimUsernameType},
		},
		{
			name:     "no tasks are completed, return nil",
			p:        &progress{},
			repo:     &repository{cfg: &config{}},
			expected: nil,
		},
	}

	for _, tt := range testCases { //nolint:gocritic // it's a test, no need for optimization
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.p.reEvaluateCompletedTasks(tt.repo)
			require.EqualValues(t, tt.expected, got)
		})
	}
}
