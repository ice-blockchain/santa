// SPDX-License-Identifier: ice License 1.0

package badges

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ice-blockchain/eskimo/users"
	wintrconfig "github.com/ice-blockchain/wintr/config"
)

//nolint:funlen // A lot of testcases in test
func Test_Progress_ReevaluateAchievedBadges(t *testing.T) {
	t.Parallel()
	defCfg := defaultCfg()
	testCases := []*struct {
		*progress
		cfg                    *config
		expectedNewBadgesState *users.Enum[Type]
		name                   string
	}{
		{
			name:                   "1st badges requires zero so they are achieved automatically",
			progress:               badgeProgress(nil, 0, 0, 0),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type},
		},
		{
			name:                   "No badges with non-zero balance",
			progress:               badgeProgress(nil, 1, 0, 0),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type, Coin1Type},
		},
		{
			name:                   "Nothing to achieve cuz we already have social1 and level1",
			progress:               badgeProgress(&users.Enum[Type]{Social1Type, Level1Type}, 0, defCfg.Milestones[Social1Type].FromInclusive, 0),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type},
		},
		{
			name:                   "Achieve next one for the socials",
			progress:               badgeProgress(&users.Enum[Type]{Social1Type, Level1Type}, 0, defCfg.Milestones[Social2Type].FromInclusive, 0),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type, Social2Type},
		},
		{
			name:     "Achieve a lot of new badges at once",
			progress: badgeProgress(&users.Enum[Type]{Social1Type, Level1Type}, 0, math.MaxUint64, 0),
			cfg:      defCfg,
			expectedNewBadgesState: &users.Enum[Type]{
				Social1Type,
				Level1Type,
				Social2Type,
				Social3Type,
				Social4Type,
				Social5Type,
				Social6Type,
				Social7Type,
				Social8Type,
				Social9Type,
				Social10Type,
			},
		},
		{
			name: "Downgrade value for already achieved badge does not change badge state",
			progress: badgeProgress(&users.Enum[Type]{
				Level1Type,
				Social1Type,
				Social2Type,
				Social3Type,
				Social4Type,
				Social5Type,
				Social6Type,
				Social7Type,
				Social8Type,
				Social9Type,
				Social10Type,
			}, 0, 1, 0),
			cfg: defCfg,
			expectedNewBadgesState: &users.Enum[Type]{
				Level1Type,
				Social1Type,
				Social2Type,
				Social3Type,
				Social4Type,
				Social5Type,
				Social6Type,
				Social7Type,
				Social8Type,
				Social9Type,
				Social10Type,
			},
		},
		{
			name:                   "Achieve next one for the balances",
			progress:               badgeProgress(&users.Enum[Type]{Social1Type, Level1Type}, defCfg.Milestones[Coin1Type].ToInclusive, 0, 0),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type, Coin1Type},
		},
		{
			name:                   "Test inclusive verification for to value",
			progress:               badgeProgress(&users.Enum[Type]{Social1Type, Level1Type}, 0, 0, 2),
			cfg:                    defCfg,
			expectedNewBadgesState: &users.Enum[Type]{Social1Type, Level1Type, Level2Type},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actualAchievedBadges := tt.progress.reEvaluateAchievedBadges(&repository{cfg: tt.cfg})
			actualBadges := []Type{}
			if actualAchievedBadges != nil {
				actualBadges = []Type(*actualAchievedBadges)
			}
			expected := []Type{}
			if tt.expectedNewBadgesState != nil {
				expected = []Type(*tt.expectedNewBadgesState)
			}
			assert.ElementsMatch(t, expected, actualBadges)
		})
	}
}

//nolint:funlen // A lot of testcases
func Test_IsBadgeGroupAchieved(t *testing.T) {
	t.Parallel()
	testCases := []*struct {
		name                  string
		alreadyAchievedBadges *users.Enum[Type]
		group                 GroupType
		expected              bool
	}{
		{
			"no badges achieved, group is no achieved as well",
			nil,
			CoinGroupType,
			false,
		},
		{
			"no badges achieved in certain group, but in another one",
			&users.Enum[Type]{Level1Type, Level2Type, Level3Type, Level4Type, Level5Type, Level6Type},
			CoinGroupType,
			false,
		},
		{
			"Badges are, partially achieved, but group itself is not",
			&users.Enum[Type]{Coin1Type, Coin2Type, Coin3Type},
			CoinGroupType,
			false,
		},
		{
			"Last badge is required for the group to be achieved",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type,
			},
			CoinGroupType,
			false,
		},
		{
			"All badges in the group are achieved",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
			},
			CoinGroupType,
			true,
		},
		{
			"All badges in the group are achieved, and partially achieved in another group(Coins)",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
				Level1Type, Level2Type, Level3Type,
			},
			CoinGroupType,
			true,
		},
		{
			"All badges in the group are achieved, and partially achieved in another group(Levels)",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
				Level1Type, Level2Type, Level3Type,
			},
			LevelGroupType,
			false,
		},
		{
			"Multiple groups achieved (Levels)",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
				Level1Type, Level2Type, Level3Type, Level4Type, Level5Type, Level6Type,
			},
			LevelGroupType,
			true,
		},
		{
			"Multiple groups achieved (Coins)",
			&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin4Type,
				Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
				Level1Type, Level2Type, Level3Type, Level4Type, Level5Type, Level6Type,
			},
			CoinGroupType,
			true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actualIsGroupAchieved := IsBadgeGroupAchieved(tt.alreadyAchievedBadges, tt.group)
			assert.Equal(t, tt.expected, actualIsGroupAchieved)
		})
	}
}

func defaultCfg() *config {
	var cfg config
	const applicationYamlTestKey = applicationYamlKey + "_test"
	wintrconfig.MustLoadFromKey(applicationYamlTestKey, &cfg)

	return &cfg
}

func badgeProgress(alreadyAchieved *users.Enum[Type], balance, friends, levels uint64) *progress {
	return &progress{
		AchievedBadges:  alreadyAchieved,
		Balance:         balance,
		FriendsInvited:  friends,
		CompletedLevels: levels,
	}
}
