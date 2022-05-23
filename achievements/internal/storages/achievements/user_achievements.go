package achievements

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"math"
)

func (r *repository) DeleteUserAchievements(userID users.UserID) error {
	sql := `DELETE FROM user_achievements WHERE user_id = :userID`
	params := map[string]interface{}{
		"userID": userID,
	}

	return errors.Wrapf(storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)),
		"failed to delete user achievements record for user.ID:%v", userID)
}

func (r *repository) GetUserAchievements(userID users.UserID) (*UserAchievements, error) {
	res := new(userAchievements)
	if err := r.db.GetTyped(userAchievementsSpace, "pk_unnamed_USER_ACHIEVEMENTS_1", tarantool.StringKey{S: userID}, res); err != nil {
		return nil, errors.Wrapf(err, "unable to get user_achievements record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no user achievements record for userID:%v", userID)
	}

	return res.UserAchievements(), nil
}

func (r *repository) InsertUserAchievements(userID users.UserID) error {
	ua := &userAchievements{
		UserID: userID,
		// Other fields have DEFAULT values on DDL.
	}

	return errors.Wrapf(r.db.InsertTyped(userAchievementsSpace, ua, &[]*userAchievements{}),
		"failed to insert user achievements record for user.ID:%v", userID)
}

func (r *repository) UpdateT1ReferralsCount(userID users.UserID, diff int64) error {
	key := tarantool.StringKey{S: userID}
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldT1Referrals, Arg: diff},
	}

	return errors.Wrapf(r.db.UpdateTyped(userAchievementsSpace, "pk_unnamed_USER_ACHIEVEMENTS_1", key, incrementOps, &[]*userAchievements{}),
		"failed to update %v record with the new count of T1 referals for userID:%v", userAchievementsSpace, userID)
}

func (r *repository) UpdateTotalUsersCount(diff int64) error {
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: 1, Arg: diff},
	}

	return errors.Wrap(r.db.UpsertAsync("GLOBAL", &global{Key: "TOTAL_USERS", Value: 1}, incrementOps).GetTyped(&[]*global{}),
		"failed to update global record the KEY = 'TOTAL_USERS'")
}

func (u *userAchievements) UserAchievements() *UserAchievements {
	return &UserAchievements{
		UserID:      u.UserID,
		Role:        u.Role,
		Balance:     u.Balance,
		Level:       u.Level,
		T1Referrals: u.T1Referrals,
	}
}
