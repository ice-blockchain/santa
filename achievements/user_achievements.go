// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"
)

func contains(data []string, find string) bool {
	for _, item := range data {
		if item == find {
			return true
		}
	}

	return false
}

func (r *repository) GetUserAchievements(ctx context.Context, userID UserID, collectibles []string) (*UserAchievements, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "failed to get user achievements because of context failed")
	}
	achievement, err := r.getUserAchievement(userID)
	if err != nil {
		return nil, errors.Wrap(err, "unable get user achievement")
	}
	userAchievement := &UserAchievements{
		Role:   achievement.Role,
		Badges: nil,
		Tasks:  nil,
		Level:  achievement.Level,
	}
	if contains(collectibles, "BADGES") {
		userAchievement.Badges, err = r.getBadgeOverview(userID)
		if err != nil {
			return nil, errors.Wrap(err, "unable get badges")
		}
	}
	if contains(collectibles, "TASKS") {
		userAchievement.Tasks, err = r.getTasks(userID)
		if err != nil {
			return nil, errors.Wrap(err, "unable get tasks")
		}
	}

	return userAchievement, nil
}

//nolint:funlen // because on long inline sql
func (r *repository) getBadgeOverview(userID UserID) ([]*BadgeOverview, error) {
	sql := "SELECT type, count(*) as size FROM badges GROUP BY type"
	var res []*badgeTypes
	if err := r.db.PrepareExecuteTyped(sql, nil, &res); err != nil {
		return nil, errors.Wrapf(err, "failed to get badge types list")
	}

	types := make(map[string]uint64)
	for _, item := range res {
		types[item.Type] = item.Size
	}

	sql = `
	SELECT
		name, type, from_inclusive, to_inclusive,
		(select true
		 from achieved_user_badges
		 where BADGE_NAME = badges.name
		   and user_id = :userId
		 ) as achieved
	FROM badges
	ORDER BY type, from_inclusive`

	var badges []*badgeOverview
	params := map[string]interface{}{"userId": userID}
	if err := r.db.PrepareExecuteTyped(sql, params, &badges); err != nil {
		return nil, errors.Wrapf(err, "failed to get badge list with achived for user %v", userID)
	}

	overview := make([]*BadgeOverview, 0, len(badges))
	var idx uint64
	var lastType string
	for _, b := range badges {
		if b.BadgeType != lastType {
			idx = 1
			lastType = b.BadgeType
		}
		overview = append(overview, &BadgeOverview{
			Badge: Badge{
				Name: b.Name,
				Type: b.BadgeType,
				Interval: ProgressInterval{
					Left:  b.FromInclusive,
					Right: b.ToInclusive,
				},
			},
			Position: Position{
				X:     idx,
				OutOf: types[b.BadgeType],
			},
		})
		idx++
	}

	return overview, nil
}

func (r *repository) getTasks(userID UserID) ([]*Task, error) {
	params := map[string]interface{}{
		"userId": userID,
	}
	sql := `
	SELECT 
		name, task_index, 
		(
		SELECT 
			true 
		FROM 
			achieved_user_tasks 
		WHERE 
			TASK_NAME = tasks.name 
			AND user_id = :userId
		) as achieved
	FROM 
		tasks
	ORDER BY 
		task_index`
	var result []*Task
	if err := r.db.PrepareExecuteTyped(sql, params, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to get task list with achived for user %v", userID)
	}
	if len(result) == 0 {
		return nil, errors.Wrapf(ErrRelationNotFound, "no user found with id: %v", userID)
	}

	return result, nil
}

func (r *repository) getUserAchievement(userID UserID) (*userAchievement, error) {
	space := "USER_ACHIEVEMENTS"
	index := "pk_unnamed_USER_ACHIEVEMENTS_1"
	key := tarantool.StringKey{S: userID}

	var res *userAchievement
	if err := r.db.GetTyped(space, index, key, &res); err != nil {
		return nil, errors.Wrapf(err, "unable to get %q record for userID:%v", space, userID)
	}

	return res, nil
}
