// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/eskimo/users"
)

func TestReEvaluateCompletedLevels(t *testing.T) { //nolint:funlen // It's a test function
	t.Parallel()

	completedLevels := make(users.Enum[LevelType], 0, len(&AllLevelTypes))
	for _, levelType := range &AllLevelTypes {
		completedLevels = append(completedLevels, levelType)
	}
	phoneNumberHash := "bogus"
	tenContacts := users.Enum[string]([]string{"1", "2", "3", "4", "5", "7", "8", "9", "10"})
	testCases := []struct {
		p        *progress
		repo     *repository
		expected *users.Enum[LevelType]
		name     string
	}{
		{
			name: "all levels are completed",
			p: &progress{
				CompletedLevels: &completedLevels,
			},
			expected: &completedLevels,
		},
		{
			name: "skip already completed levels",
			p: &progress{
				CompletedLevels: &users.Enum[LevelType]{Level1Type, Level2Type},
			},
			repo:     &repository{cfg: &config{}},
			expected: &users.Enum[LevelType]{Level1Type, Level2Type},
		},
		{
			name: "mining streak milestone is completed, level is completed as well",
			p: &progress{
				MiningStreak: 10,
			},
			repo: &repository{cfg: &config{
				MiningStreakMilestones: map[LevelType]uint64{Level1Type: 9},
			}},
			expected: &users.Enum[LevelType]{Level1Type},
		},
		{
			name: "pings sent milestone is completed, level is completed as well",
			p: &progress{
				PingsSent: 10,
			},
			repo: &repository{cfg: &config{
				PingsSentMilestones: map[LevelType]uint64{Level1Type: 9},
			}},
			expected: &users.Enum[LevelType]{Level1Type},
		},
		{
			name: "agenda contacts joined milestone is completed, level is completed as well",
			p: &progress{
				PhoneNumberHash:      &phoneNumberHash,
				AgendaContactUserIDs: &tenContacts,
			},
			repo: &repository{cfg: &config{
				AgendaContactsJoinedMilestones: map[LevelType]uint64{Level1Type: 9},
			}},
			expected: &users.Enum[LevelType]{Level1Type},
		},
		{
			name: "completed tasks milestone is completed, level is completed as well",
			p: &progress{
				CompletedTasks: 10,
			},
			repo: &repository{cfg: &config{
				CompletedTasksMilestones: map[LevelType]uint64{Level1Type: 9},
			}},
			expected: &users.Enum[LevelType]{Level1Type},
		},
		{
			name: "several milestones are completed, level is completed as well",
			p: &progress{
				MiningStreak: 10,
				PingsSent:    10,
			},
			repo: &repository{cfg: &config{
				MiningStreakMilestones: map[LevelType]uint64{Level1Type: 9},
				PingsSentMilestones:    map[LevelType]uint64{Level1Type: 9},
			}},
			expected: &users.Enum[LevelType]{Level1Type},
		},
		{
			name: "several milestones are completed for 2 levels, 2 levels are completed as well",
			p: &progress{
				MiningStreak: 20,
				PingsSent:    20,
			},
			repo: &repository{cfg: &config{
				MiningStreakMilestones: map[LevelType]uint64{Level1Type: 9, Level2Type: 15},
				PingsSentMilestones:    map[LevelType]uint64{Level1Type: 9, Level2Type: 14},
			}},
			expected: &users.Enum[LevelType]{Level1Type, Level2Type},
		},
		{
			name:     "no levels are completed, return nil",
			p:        &progress{},
			repo:     &repository{cfg: &config{}},
			expected: nil,
		},
	}

	for _, tt := range testCases { //nolint:gocritic // it's a test, no need for optimization
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.p.reEvaluateCompletedLevels(tt.repo)
			require.EqualValues(t, tt.expected, got)
		})
	}
}

func TestReEvaluateEnabledRoles(t *testing.T) { //nolint:funlen // It's a test function
	t.Parallel()

	testCases := []struct {
		p        *progress
		repo     *repository
		expected *users.Enum[RoleType]
		name     string
	}{
		{
			name:     "all roles are returned when they are all enabled",
			p:        &progress{EnabledRoles: &users.Enum[RoleType]{AmbassadorRoleType}},
			expected: &users.Enum[RoleType]{AmbassadorRoleType},
		},
		{
			name:     "ambassador role is returned, if friends invited threshold is passed",
			p:        &progress{FriendsInvited: 5, EnabledRoles: &users.Enum[RoleType]{AmbassadorRoleType}},
			repo:     &repository{cfg: &config{RequiredInvitedFriendsToBecomeAmbassador: 4}},
			expected: &users.Enum[RoleType]{AmbassadorRoleType},
		},
		{
			name:     "nil is returned, when no roles are enabled and friends invited threshold isn't reached",
			p:        &progress{FriendsInvited: 4},
			repo:     &repository{cfg: &config{RequiredInvitedFriendsToBecomeAmbassador: 5}},
			expected: nil,
		},
	}

	for _, tt := range testCases { //nolint:gocritic // it's a test, no need for optimization
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.p.reEvaluateEnabledRoles(tt.repo)
			require.EqualValues(t, tt.expected, got)
		})
	}
}
