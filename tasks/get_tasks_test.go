// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/eskimo/users"
)

func TestBuildTasks(t *testing.T) { //nolint:funlen // It's a test function
	t.Parallel()
	bogusUsername := "bogus"
	testCases := []struct {
		name     string
		p        *progress
		repo     *repository
		expected []*Task
	}{
		{
			name: "twitter task is completed",
			p: &progress{
				TwitterUserHandle: &bogusUsername,
				CompletedTasks:    &users.Enum[Type]{FollowUsOnTwitterType},
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: false,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: &Data{TwitterUserHandle: bogusUsername}, Type: FollowUsOnTwitterType, Completed: true,
				},
				{
					Data: nil, Type: JoinTelegramType, Completed: false,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
		{
			name: "twitter task is in progress, but not marked as completed",
			p: &progress{
				TwitterUserHandle: &bogusUsername,
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: false,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: &Data{TwitterUserHandle: bogusUsername}, Type: FollowUsOnTwitterType, Completed: false,
				},
				{
					Data: nil, Type: JoinTelegramType, Completed: false,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
		{
			name: "telegram task is completed",
			p: &progress{
				TelegramUserHandle: &bogusUsername,
				CompletedTasks:     &users.Enum[Type]{JoinTelegramType},
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: false,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: nil, Type: FollowUsOnTwitterType, Completed: false,
				},
				{
					Data: &Data{TelegramUserHandle: bogusUsername}, Type: JoinTelegramType, Completed: true,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
		{
			name: "telegram task is in progress, but not marked as completed",
			p: &progress{
				TelegramUserHandle: &bogusUsername,
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: false,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: nil, Type: FollowUsOnTwitterType, Completed: false,
				},
				{
					Data: &Data{TelegramUserHandle: bogusUsername}, Type: JoinTelegramType, Completed: false,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
		{
			name: "start mining task is pseudocompleted",
			p: &progress{
				CompletedTasks:       &users.Enum[Type]{ClaimUsernameType},
				PseudoCompletedTasks: &users.Enum[Type]{StartMiningType},
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: true,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: nil, Type: FollowUsOnTwitterType, Completed: false,
				},
				{
					Data: nil, Type: JoinTelegramType, Completed: false,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
		{
			name: "upload profile picture task is pseudocompleted, but previous one isn't",
			p: &progress{
				CompletedTasks:       &users.Enum[Type]{ClaimUsernameType},
				PseudoCompletedTasks: &users.Enum[Type]{UploadProfilePictureType},
			},
			repo: &repository{cfg: &config{RequiredFriendsInvited: 2}},
			expected: []*Task{
				{
					Data: nil, Type: ClaimUsernameType, Completed: true,
				},
				{
					Data: nil, Type: StartMiningType, Completed: false,
				},
				{
					Data: nil, Type: UploadProfilePictureType, Completed: false,
				},
				{
					Data: nil, Type: FollowUsOnTwitterType, Completed: false,
				},
				{
					Data: nil, Type: JoinTelegramType, Completed: false,
				},
				{
					Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
				},
			},
		},
	}

	for _, tt := range testCases { //nolint:gocritic // it's a test, no need for optimization
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.p.buildTasks(tt.repo)
			require.EqualValues(t, tt.expected, got)
		})
	}
}

func TestDefaultTasks(t *testing.T) {
	t.Parallel()

	t.Run("default, success", func(t *testing.T) {
		t.Parallel()

		r := &repository{cfg: &config{RequiredFriendsInvited: 2}}
		resp := r.defaultTasks()
		require.EqualValues(t, []*Task{
			{
				Data: nil, Type: ClaimUsernameType, Completed: true,
			},
			{
				Data: nil, Type: StartMiningType, Completed: false,
			},
			{
				Data: nil, Type: UploadProfilePictureType, Completed: false,
			},
			{
				Data: nil, Type: FollowUsOnTwitterType, Completed: false,
			},
			{
				Data: nil, Type: JoinTelegramType, Completed: false,
			},
			{
				Data: &Data{RequiredQuantity: 2}, Type: InviteFriendsType, Completed: false,
			},
		}, resp)
	})
}
