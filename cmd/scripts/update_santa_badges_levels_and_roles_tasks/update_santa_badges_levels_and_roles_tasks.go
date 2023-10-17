// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/santa/badges"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	appCfg "github.com/ice-blockchain/wintr/config"
	storagePG "github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/ice-blockchain/wintr/log"
)

const (
	applicationYamlUsersKey = "users"
	applicationYamlKeySanta = "santa"
	concurrencyCount        = 1000
)

var (
	//go:embed DDLSanta.sql
	ddlSanta string

	//go:embed DDLEskimo.sql
	ddlUsers string

	//nolint:gochecknoglobals // Singleton & global config mounted only during bootstrap.
	cfgSanta configSanta
)

type (
	configSanta struct {
		Milestones                               map[badges.Type]badges.AchievingRange `yaml:"milestones"`
		RequiredInvitedFriendsToBecomeAmbassador uint64                                `yaml:"requiredInvitedFriendsToBecomeAmbassador"`
		RequiredFriendsInvited                   uint64                                `yaml:"requiredFriendsInvited"`
	}
	updater struct {
		dbSanta  *storagePG.DB
		dbEskimo *storagePG.DB
	}
	eskimoUser struct {
		ID             string `json:"id" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2" db:"id"`
		FriendsInvited uint64 `json:"friendsInvited" example:"22" db:"friends_invited"`
	}

	commonUser struct {
		AchievedBadges       *users.Enum[badges.Type] `json:"achievedBadges,omitempty" example:"c1,l1,l2,c2" db:"achieved_badges"`
		CompletedTasks       *bool                    `json:"completedTasks,omitempty" example:"true"`
		PseudoCompletedTasks *bool                    `json:"pseudoCompletedTasks,omitempty" example:"true"`
		UserID               string
		CompletedLevels      int64 `json:"completedLevels,omitempty" example:"3"  db:"completed_levels"`
		Balance              int64 `json:"balance,omitempty" example:"1232323232"  db:"balance"`
	}
)

func main() {
	appCfg.MustLoadFromKey(applicationYamlKeySanta, &cfgSanta)

	dbEskimo := storagePG.MustConnect(context.Background(), ddlUsers, applicationYamlUsersKey)
	dbSanta := storagePG.MustConnect(context.Background(), ddlSanta, applicationYamlKeySanta)

	if err := dbEskimo.Ping(context.Background()); err != nil {
		log.Panic("can't ping users db", err)
	}
	if err := dbSanta.Ping(context.Background()); err != nil {
		log.Panic("can't ping santa db", err)
	}
	upd := &updater{
		dbSanta:  dbSanta,
		dbEskimo: dbEskimo,
	}
	defer upd.dbEskimo.Close()
	defer upd.dbSanta.Close()

	upd.update(context.Background())
}

//nolint:revive,funlen,gocognit // .
func (u *updater) update(ctx context.Context) {
	var (
		updatedCount uint64
		maxLimit     uint64 = 10000
		offset       uint64
		recalculated []string
	)
	concurrencyGuard := make(chan struct{}, concurrencyCount)
	wg := new(sync.WaitGroup)
	for {
		/******************************************************************************************************************************************************
			1. Fetching a new batch of users from eskimo and filtering already updated users.
		******************************************************************************************************************************************************/
		sql := `SELECT 
					u.id,
					COUNT(t1.id) 		AS friends_invited
				FROM users u
				LEFT JOIN users t1
					ON t1.referred_by = u.id
					AND u.username != u.id
				LEFT JOIN updated_santa_users upd
					ON upd.user_id = u.id
				WHERE u.referred_by != u.id
					AND u.username != u.id
					AND upd.user_id IS NULL
				GROUP BY u.id
				LIMIT $1
				OFFSET $2`
		usrs, err := storagePG.Select[eskimoUser](ctx, u.dbEskimo, sql, maxLimit, offset)
		if err != nil {
			log.Panic("error on trying to get actual friends invited values crossed with already updated values", err)
		}
		if len(usrs) == 0 {
			log.Debug("no more users to handle results for")

			break
		}

		/******************************************************************************************************************************************************
			2. Fetching tasks and badges specific data.
		******************************************************************************************************************************************************/
		var userKeysProgress []string
		actualFriendsInvitedCount := make(map[string]uint64, len(usrs))
		for _, usr := range usrs {
			userKeysProgress = append(userKeysProgress, usr.ID)
			actualFriendsInvitedCount[usr.ID] = usr.FriendsInvited
		}
		sql = `SELECT 
					tp.user_id,
					'invite_friends' = ANY(tp.completed_tasks)  		AS completed_tasks,
					'invite_friends' = ANY(tp.pseudo_completed_tasks)  	AS pseudo_completed_tasks,
					bp.balance,
					bp.completed_levels 								AS completed_levels,
					bp.achieved_badges									AS achieved_badges
				FROM task_progress tp
					JOIN badge_progress bp
						ON tp.user_id = bp.user_id
					WHERE tp.user_id = ANY($1)`
		res, err := storagePG.Select[commonUser](ctx, u.dbSanta, sql, userKeysProgress)
		if err != nil {
			log.Panic("error on trying to get tasks", userKeysProgress, err)
		}
		if len(res) == 0 {
			log.Debug("no results for: ", userKeysProgress, err)
			offset += maxLimit

			continue
		}

		/******************************************************************************************************************************************************
			3. Updating santa.
		******************************************************************************************************************************************************/
		for _, r := range res {
			usr := r
			wg.Add(1)
			concurrencyGuard <- struct{}{}
			go func() {
				defer wg.Done()
				if uErr := u.updateBadgesAndStatistics(ctx, usr, actualFriendsInvitedCount[usr.UserID]); uErr != nil {
					log.Panic("can't update badges and badges statistics, userID:", usr.UserID, uErr)
				}
				if uErr := u.updateLevelsAndRoles(ctx, usr, actualFriendsInvitedCount[usr.UserID]); uErr != nil {
					log.Panic("can't update levels and roles, userID:", usr.UserID, uErr)
				}
				if uErr := u.updateTasks(ctx, usr, actualFriendsInvitedCount[usr.UserID]); uErr != nil {
					log.Panic("can't update tasks, userID:", usr.UserID, uErr)
				}
				if uErr := u.updateFriendsInvited(ctx, usr, actualFriendsInvitedCount[usr.UserID]); uErr != nil {
					log.Panic("can't update friends invited, userID:", usr.UserID, uErr)
				}
				<-concurrencyGuard
			}()
			recalculated = append(recalculated, fmt.Sprintf("('%v')", usr.UserID))
		}

		/******************************************************************************************************************************************************
			4. Persisting recalculated status.
		******************************************************************************************************************************************************/
		if len(recalculated) > 0 {
			sql = fmt.Sprintf(`INSERT INTO updated_santa_users(user_id) VALUES %v
									ON CONFLICT DO NOTHING`, strings.Join(recalculated, ","))
			if _, eErr := storagePG.Exec(ctx, u.dbEskimo, sql); eErr != nil {
				log.Panic("error on persisting recalculated values", eErr)
			}
		}
		updatedCount += uint64(len(res))
		log.Info("updated count: ", updatedCount)

		offset += maxLimit
	}
	wg.Wait()
}

//nolint:funlen // .
func (u *updater) updateBadgesAndStatistics(ctx context.Context, usr *commonUser, actualFriendsInvited uint64) error {
	achievedBadges, newBadgesTypeCount := reEvaluateEnabledBadges(usr.AchievedBadges, actualFriendsInvited, usr.Balance)
	var completedLevelsSQL string
	if ((usr.CompletedTasks != nil && *usr.CompletedTasks) || (usr.PseudoCompletedTasks != nil && *usr.PseudoCompletedTasks)) &&
		actualFriendsInvited < cfgSanta.RequiredFriendsInvited {
		completedLevelsSQL = ",completed_levels = GREATEST(completed_levels - 1, 0)"
	}
	newBadgesTypeCount = diffBadgeStatistics(usr, newBadgesTypeCount)
	sql := fmt.Sprintf(`UPDATE badge_progress
								SET friends_invited = $2,
									achieved_badges = $3
									%v
							WHERE user_id = $1
								  AND (friends_invited != $2
								  	   OR COALESCE(badge_progress.achieved_badges, ARRAY[]::TEXT[]) != COALESCE($3, ARRAY[]::TEXT[]))
									   OR $4 = TRUE`, completedLevelsSQL)
	if _, err := storagePG.Exec(ctx, u.dbSanta, sql, usr.UserID, actualFriendsInvited, achievedBadges, completedLevelsSQL != ""); err != nil {
		return errors.Wrapf(err, "failed to update badge_progress, userID:%v, friendsInvited:%v", usr.UserID, actualFriendsInvited)
	}
	var mErr *multierror.Error
	for badgeType, val := range newBadgesTypeCount {
		if val == 0 {
			continue
		}
		sign := "+"
		if val < 0 {
			sign = "-"
			val *= -1
		}
		sql = fmt.Sprintf(`UPDATE badge_statistics
										SET achieved_by = GREATEST(achieved_by %v $1, 0)
									WHERE badge_type = $2`, sign)
		_, err := storagePG.Exec(ctx, u.dbSanta, sql, val, badgeType)
		mErr = multierror.Append(errors.Wrapf(err, "failed to update badge_statistics, userID:%v, badgeType:%v, val:%v", usr.UserID, badgeType, val))
	}

	return errors.Wrapf(multierror.Append(mErr, nil).ErrorOrNil(), "can't update badge statistics")
}

func diffBadgeStatistics(usr *commonUser, newBadgesTypeCount map[badges.Type]int64) map[badges.Type]int64 {
	oldBadgesTypeCounts := make(map[badges.Type]int64, len(badges.AllTypes))
	oldGroupCounts := make(map[badges.GroupType]int64, len(badges.AllGroups))
	for _, badge := range *usr.AchievedBadges {
		switch badges.GroupTypeForEachType[badge] { //nolint:exhaustive // We need to handle only 2 groups.
		case badges.CoinGroupType:
			oldBadgesTypeCounts[badge]++
			oldGroupCounts[badges.CoinGroupType]++
		case badges.SocialGroupType:
			oldBadgesTypeCounts[badge]++
			oldGroupCounts[badges.SocialGroupType]++
		default:
			continue
		}
	}
	if newBadgesTypeCount != nil {
		for _, key := range &badges.AllTypes {
			if _, ok1 := oldBadgesTypeCounts[key]; ok1 {
				if _, ok2 := newBadgesTypeCount[key]; ok2 {
					newBadgesTypeCount[key] -= oldBadgesTypeCounts[key]
				}
			}
		}
	}

	return newBadgesTypeCount
}

func (u *updater) updateLevelsAndRoles(ctx context.Context, usr *commonUser, actualFriendsInvited uint64) error {
	enabledRoles := reEvaluateEnabledRole(actualFriendsInvited)
	var completedTasksSQL, completedLevelsSQL string
	if ((usr.CompletedTasks != nil && *usr.CompletedTasks) || (usr.PseudoCompletedTasks != nil && *usr.PseudoCompletedTasks)) &&
		actualFriendsInvited < cfgSanta.RequiredFriendsInvited {
		completedTasksSQL = ", completed_tasks = GREATEST(completed_tasks - 1, 0)"
		completedLevelsSQL = ",completed_levels = array_remove(completed_levels, '11')" // We know for sure from config file this level id that need to be removed.
	}
	sql := fmt.Sprintf(`UPDATE 	levels_and_roles_progress
								SET friends_invited = $2,
									enabled_roles = $3
									%v
									%v
							WHERE user_id = $1 
								  AND (friends_invited != $2 
								  OR COALESCE(levels_and_roles_progress.enabled_roles, ARRAY[]::TEXT[]) != COALESCE($3, ARRAY[]::TEXT[])
								  OR $4 = TRUE)`, completedTasksSQL, completedLevelsSQL)
	_, err := storagePG.Exec(ctx, u.dbSanta, sql, usr.UserID, actualFriendsInvited, enabledRoles, (completedTasksSQL != "" || completedLevelsSQL != ""))

	return errors.Wrapf(err, "failed to update levels_and_roles_progress, userID:%v, friendsInvited:%v", usr.UserID, actualFriendsInvited)
}

func (u *updater) updateTasks(ctx context.Context, usr *commonUser, actualFriendsInvited uint64) error {
	var completedTasksSQL, whereSQL string
	if ((usr.CompletedTasks != nil && *usr.CompletedTasks) || (usr.PseudoCompletedTasks != nil && *usr.PseudoCompletedTasks)) &&
		actualFriendsInvited < cfgSanta.RequiredFriendsInvited {
		completedTasksSQL = `, completed_tasks = array_remove(completed_tasks, 'invite_friends')
							 , pseudo_completed_tasks = array_remove(pseudo_completed_tasks, 'invite_friends')`
	}
	sql := fmt.Sprintf(`UPDATE task_progress
								SET friends_invited = $2
								%v
							WHERE user_id = $1
								  AND (friends_invited != $2 %v OR $3 = TRUE)`, completedTasksSQL, whereSQL)
	_, err := storagePG.Exec(ctx, u.dbSanta, sql, usr.UserID, actualFriendsInvited, completedTasksSQL != "")

	return errors.Wrapf(err, "failed to update task_progress, userID:%v, friendsInvited:%v", usr.UserID, actualFriendsInvited)
}

func (u *updater) updateFriendsInvited(ctx context.Context, usr *commonUser, actualFriendsInvited uint64) error {
	sql := `UPDATE friends_invited
				   SET invited_count = $2
				   WHERE user_id = $1
				   		 AND friends_invited.invited_count != $2`
	_, err := storagePG.Exec(ctx, u.dbSanta, sql, usr.UserID, actualFriendsInvited)

	return errors.Wrapf(err, "failed to update friends invited, userID:%v, friendsInvited:%v", usr.UserID, actualFriendsInvited)
}

func reEvaluateEnabledRole(friendsInvited uint64) *users.Enum[levelsandroles.RoleType] {
	if friendsInvited >= cfgSanta.RequiredInvitedFriendsToBecomeAmbassador {
		completedLevels := append(make(users.Enum[levelsandroles.RoleType], 0, len(&levelsandroles.AllRoleTypesThatCanBeEnabled)), levelsandroles.AmbassadorRoleType)

		return &completedLevels
	}

	return nil
}

//nolint:funlen // .
func reEvaluateEnabledBadges(
	alreadyAchievedBadges *users.Enum[badges.Type], friendsInvited uint64, balance int64,
) (achievedBadges users.Enum[badges.Type], badgesTypeCounts map[badges.Type]int64) {
	badgesTypeCounts = make(map[badges.Type]int64)
	achievedBadges = make(users.Enum[badges.Type], 0, len(&badges.AllTypes))
	if alreadyAchievedBadges != nil {
		for _, badge := range *alreadyAchievedBadges {
			if strings.HasPrefix(string(badge), "l") {
				achievedBadges = append(achievedBadges, badge)
			}
		}
	}
	for _, badgeType := range &badges.AllTypes {
		var achieved bool
		switch badges.GroupTypeForEachType[badgeType] { //nolint:exhaustive // We need to handle only 2 cases.
		case badges.CoinGroupType:
			if balance > 0 {
				achieved = uint64(balance) >= cfgSanta.Milestones[badgeType].FromInclusive
			}
		case badges.SocialGroupType:
			achieved = friendsInvited >= cfgSanta.Milestones[badgeType].FromInclusive
		default:
			continue
		}
		if achieved {
			achievedBadges = append(achievedBadges, badgeType)
			badgesTypeCounts[badgeType]++
		}
	}
	if len(achievedBadges) == 0 {
		return nil, nil
	}
	sort.SliceStable(achievedBadges, func(i, j int) bool {
		return badges.AllTypeOrder[achievedBadges[i]] < badges.AllTypeOrder[achievedBadges[j]]
	})

	return achievedBadges, badgesTypeCounts
}
