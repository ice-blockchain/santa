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

	a, err := r.getUserAchievement(userID)
	if err != nil {
		return nil, errors.Wrap(err, "unable get user achievement")
	}

	return a.decodeAchievement(collectibles)
}

func (r *repository) getUserAchievement(userID UserID) (*userAchievement, error) {
	sql := r.userAchievementQuery()
	params := map[string]interface{}{
		"userId": userID,
	}
	var result []*userAchievement
	if err := r.db.PrepareExecuteTyped(sql, params, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to get user achievements for user %v", userID)
	}
	if len(result) == 0 {
		return nil, errors.Wrapf(ErrRelationNotFound, "no user found with id: %v", userID)
	}

	return result[0], nil
}

//nolint:funlen // because on long inline sql
func (r *repository) userAchievementQuery() string {
	return `
SELECT
    (SELECT '[' || group_concat('{"name": "' || name ||
                               '", "type": "' || type ||
                               '", "fromInclusive": ' || cast(from_inclusive as string) ||
                               ', "toInclusive": ' || cast(to_inclusive as string) ||
                               ', "achievedAt": ' || cast(achieved_at as string) || '}') || ']'
    FROM (
        SELECT
            b.name, b.type, b.from_inclusive, b.to_inclusive, COALESCE(aub.achieved_at, 0) as achieved_at
        FROM badges AS b
            LEFT JOIN achieved_user_badges AS aub
                ON aub.badge_name = b.name
                       AND aub.user_id = :userId
        ORDER BY b.type, b.from_inclusive
    )) as user_badges,
    (SELECT '[' || group_concat('{"name": "' || name ||
                                '","taskIndex": ' || cast(task_index as string) ||
                                ',"achievedAt": ' || cast(achieved_at as string) || '}') || ']'
    FROM (SELECT t.name,
                 t.task_index,
                 coalesce(aut.achieved_at, 0) as achieved_at
          FROM tasks AS t
                   LEFT JOIN achieved_user_tasks aut
                             ON aut.task_name = t.name
                                 AND aut.user_id = :userId
          ORDER BY task_index
    )) as user_tasks,
    cur.ROLE_NAME as user_role,
    (SELECT '[' || group_concat('{"type": "' || type ||
                '","count": ' || CAST(CNT as string) || '}') || ']'
    FROM (SELECT type, COUNT(*) as cnt FROM badges GROUP BY type)) as badge_types,
    (SELECT count(1) FROM achieved_user_levels WHERE user_id = :userId) as user_level
FROM user_progress as up
    LEFT JOIN current_user_roles as cur ON cur.USER_ID = up.USER_ID
WHERE up.user_id = :userId
`
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
	var badgeTypes []*badgeType
	if err := json.NewDecoder(bytes.NewReader([]byte(a.BadgeTypes))).Decode(&badgeTypes); err != nil {
		return nil, errors.Wrapf(err, "unable to decode badgeType: %s", a.BadgeTypes)
	}
	badgeMap := make(map[string]uint64)
	for _, badge := range badgeTypes {
		badgeMap[badge.Type] = badge.Count
	}

	var userBadges []*userBadge
	if contains(collectibles, "BADGES") {
		if err := json.NewDecoder(bytes.NewReader([]byte(a.UserBadges))).Decode(&userBadges); err != nil {
			return nil, errors.Wrapf(err, "unable to decode userBadge: %s", a.UserBadges)
		}
	}

	var userTasks []*userTask
	if contains(collectibles, "TASKS") {
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
