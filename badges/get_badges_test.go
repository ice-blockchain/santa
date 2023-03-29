// SPDX-License-Identifier: ice License 1.0

package badges

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/eskimo/users"
)

const (
	totalUsers = 100
)

type (
	calculateUnachievedPercentagesTestCase struct {
		expected   map[Type]float64
		name       string
		desc       string
		group      GroupType
		stats      []*statistics
		totalUsers uint64
	}
)

func (c *calculateUnachievedPercentagesTestCase) Expected(ex map[Type]float64) *calculateUnachievedPercentagesTestCase {
	c.expected = ex

	return c
}

func (c *calculateUnachievedPercentagesTestCase) WithDesc(description string) *calculateUnachievedPercentagesTestCase {
	c.desc = description

	return c
}

func (c *calculateUnachievedPercentagesTestCase) WithUsers(
	tb testing.TB,
	group GroupType,
	usersCounts ...int,
) *calculateUnachievedPercentagesTestCase {
	tb.Helper()
	stats := []*statistics{{
		Type:       Type(group),
		GroupType:  group,
		AchievedBy: totalUsers,
	}}
	require.GreaterOrEqual(tb, len(usersCounts), len(AllGroups[group]))
	achievedStr := make([]string, len(AllGroups[group])) //nolint:makezero // We're know size for sure
	for ind, badgeType := range AllGroups[group] {
		achievedByUsersCount := usersCounts[ind]
		stats = append(stats, &statistics{Type: badgeType, GroupType: group, AchievedBy: uint64(achievedByUsersCount)})
		achievedStr[ind] = strconv.FormatUint(uint64(achievedByUsersCount), 10)
	}
	c.group = group
	c.name = string(group) + "-" + strings.Join(achievedStr, ",")
	c.totalUsers = totalUsers
	c.stats = stats

	return c
}

func unachivedPercentageTestCase() *calculateUnachievedPercentagesTestCase {
	return &calculateUnachievedPercentagesTestCase{}
}

//nolint:funlen // A lot of testcases here
func Test_Repository_calculateUnachievedPercentages(t *testing.T) {
	t.Parallel()
	tests := []*calculateUnachievedPercentagesTestCase{
		unachivedPercentageTestCase().
			WithDesc("All users have first 2 badges, and no one have last").
			WithUsers(t, SocialGroupType, 100, 100, 50, 25, 10, 10, 5, 0, 0, 0).
			Expected(map[Type]float64{
				Social1Type:  0,
				Social2Type:  0,
				Social3Type:  50,
				Social4Type:  25,
				Social5Type:  15,
				Social6Type:  0,
				Social7Type:  5,
				Social8Type:  5,
				Social9Type:  0,
				Social10Type: 0,
			}),
		unachivedPercentageTestCase().
			WithDesc("50% of users did not get even 1st badge and 10% get all").
			WithUsers(t, LevelGroupType, 50, 10, 10, 10, 10, 10).
			Expected(map[Type]float64{
				Level1Type: 50,
				Level2Type: 40,
				Level3Type: 0,
				Level4Type: 0,
				Level5Type: 0,
				Level6Type: 0,
			}),
		unachivedPercentageTestCase().
			WithDesc("All users achieved all badges").
			WithUsers(t, CoinGroupType, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100).
			Expected(map[Type]float64{
				Coin1Type:  0,
				Coin2Type:  0,
				Coin3Type:  0,
				Coin4Type:  0,
				Coin5Type:  0,
				Coin6Type:  0,
				Coin7Type:  0,
				Coin8Type:  0,
				Coin9Type:  0,
				Coin10Type: 0,
			}),
		unachivedPercentageTestCase().
			WithDesc("All users achieved first 9 badges").
			WithUsers(t, CoinGroupType, 100, 100, 100, 100, 100, 100, 100, 100, 100, 0).
			Expected(map[Type]float64{
				Coin1Type:  0,
				Coin2Type:  0,
				Coin3Type:  0,
				Coin4Type:  0,
				Coin5Type:  0,
				Coin6Type:  0,
				Coin7Type:  0,
				Coin8Type:  0,
				Coin9Type:  0,
				Coin10Type: 100,
			}),
		unachivedPercentageTestCase().
			WithDesc("No one does not have anything").
			WithUsers(t, SocialGroupType, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).
			Expected(map[Type]float64{
				Social1Type:  100,
				Social2Type:  0,
				Social3Type:  0,
				Social4Type:  0,
				Social5Type:  0,
				Social6Type:  0,
				Social7Type:  0,
				Social8Type:  0,
				Social9Type:  0,
				Social10Type: 0,
			}),
		unachivedPercentageTestCase().
			WithDesc("Achieved users greater than total due to any reason").
			WithUsers(t, SocialGroupType, math.MaxInt, 110, 101, 90, 0, 0, 0, 0, 0, 0).
			Expected(map[Type]float64{
				Social1Type:  0,
				Social2Type:  0,
				Social3Type:  0,
				Social4Type:  10,
				Social5Type:  90,
				Social6Type:  0,
				Social7Type:  0,
				Social8Type:  0,
				Social9Type:  0,
				Social10Type: 0,
			}),
		unachivedPercentageTestCase().
			WithDesc("0.2+0.1 != 0.3000000000004").
			WithUsers(t, SocialGroupType, 100, 20, 10, 0, 0, 0, 0, 0, 0, 0).
			Expected(map[Type]float64{
				Social1Type:  0,
				Social2Type:  80,
				Social3Type:  10,
				Social4Type:  10,
				Social5Type:  0,
				Social6Type:  0,
				Social7Type:  0,
				Social8Type:  0,
				Social9Type:  0,
				Social10Type: 0,
			}),
	}

	random := randomAchievedUsers(len(AllGroups[SocialGroupType]))
	tests = append(tests, unachivedPercentageTestCase().WithUsers(t, SocialGroupType, random...).
		Expected(expectedForRandomAchievedUsers(SocialGroupType, random...)))
	random = randomAchievedUsers(len(AllGroups[CoinGroupType]))
	tests = append(tests, unachivedPercentageTestCase().WithUsers(t, CoinGroupType, random...).
		Expected(expectedForRandomAchievedUsers(CoinGroupType, random...)))
	random = randomAchievedUsers(len(AllGroups[LevelGroupType]))
	tests = append(tests, unachivedPercentageTestCase().WithUsers(t, LevelGroupType, random...).
		Expected(expectedForRandomAchievedUsers(LevelGroupType, random...)))
	r := &repository{}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%v(%v)", tt.desc, tt.name), func(t *testing.T) {
			t.Parallel()
			actual := r.calculateUnachievedPercentages(tt.group, tt.stats)
			// Avoid values like 28.999999999999996.
			sum := float64(0.0)
			for k, v := range actual {
				actual[k] = math.Round(v)
				sum += v
			}
			require.NotEmpty(t, actual)
			require.Len(t, actual, len(AllGroups[tt.group]))
			require.Equal(t, tt.expected, actual)
			// User can be in progress with only 1 badge (per groupType ofc) at any given time
			// so sum of percentages is 100-usersWhoOwnsLastBadge.
			lastBadgeAchievedBy := tt.stats[len(AllGroups[tt.group])].AchievedBy
			lastBadgeAchievedByPercentage := float64(lastBadgeAchievedBy*percent100) / float64(totalUsers)
			assert.Equal(t, percent100-lastBadgeAchievedByPercentage, sum)
		})
	}
}

//nolint:funlen // A lot of testcases
func Test_Progress_BuildBadges(t *testing.T) {
	t.Parallel()
	repo := &repository{cfg: defaultCfg()}
	tests := []*struct {
		progress *progress
		stats    map[Type]float64
		group    GroupType
		name     string
		expected []*Badge
	}{
		{
			name:     "Nothing achieved",
			progress: badgeProgress(nil, 0, 0, 0),
			group:    LevelGroupType,
			stats:    map[Type]float64{},
			expected: []*Badge{
				expectedBadge(repo, Level1Type, false, 0),
				expectedBadge(repo, Level2Type, false, 0),
				expectedBadge(repo, Level3Type, false, 0),
				expectedBadge(repo, Level4Type, false, 0),
				expectedBadge(repo, Level5Type, false, 0),
				expectedBadge(repo, Level6Type, false, 0),
			},
		},
		{
			name: "partially achieved",
			progress: badgeProgress(&users.Enum[Type]{
				Level1Type, Level2Type, Level3Type,
			}, 0, 0, 0),
			group: LevelGroupType,
			stats: map[Type]float64{
				Level1Type: 50.0,
				Level2Type: 30.0,
				Level3Type: 20.0,
			},
			expected: []*Badge{
				expectedBadge(repo, Level1Type, true, 50.0),
				expectedBadge(repo, Level2Type, true, 30.0),
				expectedBadge(repo, Level3Type, true, 20.0),
				expectedBadge(repo, Level4Type, false, 0),
				expectedBadge(repo, Level5Type, false, 0),
				expectedBadge(repo, Level6Type, false, 0),
			},
		},
		{
			name: "all achieved, preserve order",
			progress: badgeProgress(&users.Enum[Type]{
				Coin1Type, Coin2Type, Coin3Type, Coin5Type, Coin4Type,
				Coin10Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type,
			}, 0, 0, 0),
			group: CoinGroupType,
			stats: map[Type]float64{
				Coin1Type:  10,
				Coin2Type:  10,
				Coin3Type:  10,
				Coin4Type:  10,
				Coin5Type:  10,
				Coin6Type:  10,
				Coin7Type:  10,
				Coin8Type:  10,
				Coin9Type:  10,
				Coin10Type: 10,
			},
			expected: []*Badge{
				expectedBadge(repo, Coin1Type, true, 10),
				expectedBadge(repo, Coin2Type, true, 10),
				expectedBadge(repo, Coin3Type, true, 10),
				expectedBadge(repo, Coin4Type, true, 10),
				expectedBadge(repo, Coin5Type, true, 10),
				expectedBadge(repo, Coin6Type, true, 10),
				expectedBadge(repo, Coin7Type, true, 10),
				expectedBadge(repo, Coin8Type, true, 10),
				expectedBadge(repo, Coin9Type, true, 10),
				expectedBadge(repo, Coin10Type, true, 10),
			},
		},
		{
			name:     "empty statistics",
			progress: badgeProgress(&users.Enum[Type]{Coin1Type}, 0, 0, 0),
			group:    CoinGroupType,
			stats:    map[Type]float64{},
			expected: []*Badge{
				expectedBadge(repo, Coin1Type, true, 0),
				expectedBadge(repo, Coin2Type, false, 0),
				expectedBadge(repo, Coin3Type, false, 0),
				expectedBadge(repo, Coin4Type, false, 0),
				expectedBadge(repo, Coin5Type, false, 0),
				expectedBadge(repo, Coin6Type, false, 0),
				expectedBadge(repo, Coin7Type, false, 0),
				expectedBadge(repo, Coin8Type, false, 0),
				expectedBadge(repo, Coin9Type, false, 0),
				expectedBadge(repo, Coin10Type, false, 0),
			},
		},
		{
			name:     "some are achieved but in another group",
			progress: badgeProgress(&users.Enum[Type]{Coin1Type, Level1Type}, 0, 0, 0),
			group:    SocialGroupType,
			stats: map[Type]float64{
				Social1Type: 50,
				Social2Type: 50,
			},
			expected: []*Badge{
				expectedBadge(repo, Social1Type, false, 50),
				expectedBadge(repo, Social2Type, false, 50),
				expectedBadge(repo, Social3Type, false, 0),
				expectedBadge(repo, Social4Type, false, 0),
				expectedBadge(repo, Social5Type, false, 0),
				expectedBadge(repo, Social6Type, false, 0),
				expectedBadge(repo, Social7Type, false, 0),
				expectedBadge(repo, Social8Type, false, 0),
				expectedBadge(repo, Social9Type, false, 0),
				expectedBadge(repo, Social10Type, false, 0),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.progress.buildBadges(repo, tt.group, tt.stats)
			assert.Equal(t, tt.expected, actual, tt.name)
		})
	}
}

//nolint:funlen // A lot of testcases.
func Test_Progress_BuildSummary(t *testing.T) {
	t.Parallel()
	tests := []*struct {
		name              string
		progress          *progress
		expectedSummaries []*BadgeSummary
	}{
		{
			"Nothning achieved",
			badgeProgress(nil, 0, 0, 0),
			[]*BadgeSummary{
				expectedBadgeSummaryFromBadgeType(Level1Type),
				expectedBadgeSummaryFromBadgeType(Coin1Type),
				expectedBadgeSummaryFromBadgeType(Social1Type),
			},
		},
		{
			"Automatically achieved",
			badgeProgress(&users.Enum[Type]{
				Level1Type,
				Coin1Type,
				Social1Type,
			}, 0, 0, 0),
			[]*BadgeSummary{
				expectedBadgeSummaryFromBadgeType(Level1Type),
				expectedBadgeSummaryFromBadgeType(Coin1Type),
				expectedBadgeSummaryFromBadgeType(Social1Type),
			},
		},
		{
			"In some groups achieved more than one",
			badgeProgress(&users.Enum[Type]{
				Level1Type,
				Level2Type,
				Level3Type,
				Coin1Type,
				Coin2Type,
				Coin3Type,
				Coin4Type,
				Coin5Type,
				Social1Type,
			}, 0, 0, 0),
			[]*BadgeSummary{
				expectedBadgeSummaryFromBadgeType(Level3Type),
				expectedBadgeSummaryFromBadgeType(Coin5Type),
				expectedBadgeSummaryFromBadgeType(Social1Type),
			},
		},
		{
			"One group is complete",
			badgeProgress(&users.Enum[Type]{
				Level1Type,
				Level2Type,
				Level3Type,
				Level4Type,
				Level5Type,
				Level6Type,
				Coin1Type,
				Social1Type,
			}, 0, 0, 0),
			[]*BadgeSummary{
				expectedBadgeSummaryFromBadgeType(Level6Type),
				expectedBadgeSummaryFromBadgeType(Coin1Type),
				expectedBadgeSummaryFromBadgeType(Social1Type),
			},
		},
		{
			"All groups are completed",
			badgeProgress(&users.Enum[Type]{
				Level1Type, Level2Type, Level3Type, Level4Type, Level5Type, Level6Type,
				Coin1Type, Coin2Type, Coin3Type, Coin4Type, Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type,
				Social1Type, Social2Type, Social3Type, Social4Type, Social5Type,
				Social6Type, Social7Type, Social8Type, Social9Type, Social10Type,
			}, 0, 0, 0),
			[]*BadgeSummary{
				expectedBadgeSummaryFromBadgeType(Level6Type),
				expectedBadgeSummaryFromBadgeType(Coin10Type),
				expectedBadgeSummaryFromBadgeType(Social10Type),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actualSummaries := tt.progress.buildBadgeSummaries()
			assert.ElementsMatch(t, tt.expectedSummaries, actualSummaries, tt.name)
		})
	}
}

func expectedForRandomAchievedUsers(group GroupType, userCounts ...int) map[Type]float64 {
	allTypes := AllGroups[group]
	result := make(map[Type]float64)
	for ind := 1; ind < len(allTypes); ind++ {
		badgeType := allTypes[ind]
		result[badgeType] = math.Round(float64(userCounts[ind-1]-userCounts[ind]) * percent100 / float64(totalUsers))
	}
	result[allTypes[0]] = float64(totalUsers-userCounts[0]) * percent100 / totalUsers

	return result
}

func randomAchievedUsers(count int) []int {
	achievedUserCounts := make([]int, count) //nolint:makezero // We're know size for sure.
	for i := 0; i < count; i++ {
		achievedUserCounts[i] = rand.Intn(totalUsers) //nolint:gosec // We dont need strong random for tests.
		if i > 0 && achievedUserCounts[i] > achievedUserCounts[i-1] {
			achievedUserCounts[i] = randomAchievedUsers(1)[0]
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(achievedUserCounts)))

	return achievedUserCounts
}

func expectedBadge(repo *repository, badgeType Type, achieved bool, percent float64) *Badge {
	return &Badge{
		Name:                        AllNames[GroupTypeForEachType[badgeType]][badgeType],
		Type:                        badgeType,
		GroupType:                   GroupTypeForEachType[badgeType],
		PercentageOfUsersInProgress: percent,
		Achieved:                    achieved,
		AchievingRange: AchievingRange{
			repo.cfg.Milestones[badgeType].FromInclusive,
			repo.cfg.Milestones[badgeType].ToInclusive,
		},
	}
}

func expectedBadgeSummary(name string, group GroupType, index, lastIndex uint64) *BadgeSummary {
	return &BadgeSummary{
		Name:      name,
		GroupType: group,
		Index:     index,
		LastIndex: lastIndex,
	}
}

func expectedBadgeSummaryFromBadgeType(lastAchievedType Type) *BadgeSummary {
	group := GroupTypeForEachType[lastAchievedType]
	var order int
	switch group {
	case SocialGroupType:
		order = SocialTypeOrder[lastAchievedType]
	case LevelGroupType:
		order = LevelTypeOrder[lastAchievedType]
	case CoinGroupType:
		order = CoinTypeOrder[lastAchievedType]
	}

	return expectedBadgeSummary(AllNames[group][lastAchievedType], group, uint64(order), uint64(len(AllGroups[group])-1))
}
