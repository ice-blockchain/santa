// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

func (r *repository) GetUserAchievements(ctx context.Context, userID UserID, collectibles []string) (*UserAchievements, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "failed to get user achievements because of context failed")
	}

	params := map[string]interface{}{
		"userId": userID,
	}
	sql := r.userAchievementQuery(collectibles)
	var result []*userAchievement
	if err := r.db.PrepareExecuteTyped(sql, params, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to query user achievements for user %v", userID)
	}
	if len(result) == 0 {
		return nil, errors.Wrapf(ErrRelationNotFound, "no user found with id: %v", userID)
	}

	return result[0].decodeAchievement(collectibles)
}

func (r *repository) userAchievementQuery(collectibles []string) string {
	var sql string
	switch {
	case contains(collectibles, Tasks) && !contains(collectibles, Badges):
		sql = `SELECT '' as achieved_badges, achieved_tasks, user_role, '' as badge_types, levels FROM user_achievements WHERE user_id = :userId`
	case !contains(collectibles, Tasks) && contains(collectibles, Badges):
		sql = `SELECT achieved_badges, '' as achieved_tasks, user_role, badge_types, levels FROM user_achievements WHERE user_id = :userId`
	case contains(collectibles, Tasks) && contains(collectibles, Badges):
		sql = `SELECT achieved_badges, achieved_tasks, user_role, badge_types, levels FROM user_achievements WHERE user_id = :userId`
	}

	return sql
}

func contains(data []string, match string) bool {
	for _, item := range data {
		if item == match {
			return true
		}
	}

	return false
}

func (a *userAchievement) decodeAchievement(collectibles []string) (*UserAchievements, error) {
	badgeMap := make(map[string]uint64)
	var userBadges []*userBadge
	if contains(collectibles, Badges) {
		if err := json.NewDecoder(bytes.NewReader([]byte(a.UserBadges))).Decode(&userBadges); err != nil {
			return nil, errors.Wrapf(err, "unable to decode userBadge: %s", a.UserBadges)
		}

		var badgeTypes []*badgeType
		if err := json.NewDecoder(bytes.NewReader([]byte(a.BadgeTypes))).Decode(&badgeTypes); err != nil {
			return nil, errors.Wrapf(err, "unable to decode badgeType: %s", a.BadgeTypes)
		}
		for _, badge := range badgeTypes {
			badgeMap[badge.Type] = badge.Count
		}
	}

	var userTasks []*userTask
	if contains(collectibles, Tasks) {
		if err := json.NewDecoder(bytes.NewReader([]byte(a.UserTasks))).Decode(&userTasks); err != nil {
			return nil, errors.Wrapf(err, "unable to decode userTasks: %s", a.UserTasks)
		}
	}

	return a.buildAchievement(badgeMap, userBadges, userTasks), nil
}

func (a *userAchievement) buildAchievement(badgeMap map[string]uint64, userBadges []*userBadge, userTasks []*userTask) *UserAchievements {
	var idx uint64
	var lastType string
	overviews := make([]*BadgeOverview, 0, len(userBadges))
	for _, badge := range userBadges {
		if lastType != badge.Type {
			idx = 1
			lastType = badge.Type
		}
		overviews = append(overviews, badge.toOverview(idx, badgeMap[badge.Type]))
		idx++
	}

	tasks := make([]*Task, 0, len(userTasks))
	for _, task := range userTasks {
		tasks = append(tasks, task.toTask())
	}

	return &UserAchievements{
		Role:   a.UserRole,
		Badges: overviews,
		Tasks:  tasks,
		Level:  a.UserLevel,
	}
}

func (t *userTask) toTask() *Task {
	return &Task{
		Name:     t.Name,
		Index:    t.TaskIndex,
		Achieved: t.AchievedAt > 0,
	}
}

func (b *userBadge) toOverview(x, outOf uint64) *BadgeOverview {
	return &BadgeOverview{
		Badge: Badge{
			Name: b.Name,
			Type: b.Type,
			Interval: ProgressInterval{
				Left:  b.FromInclusive,
				Right: b.ToInclusive,
			},
		},
		Position: Position{
			X:     x,
			OutOf: outOf,
		},
	}
}
