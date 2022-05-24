package progress

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"math"
)

func NewRepository(db tarantool.Connector) Repository {
	return &repository{db: db}
}
func (r *repository) DeleteUserProgress(userID users.UserID) error {
	sql := `DELETE FROM user_achievements WHERE user_id = :userID`
	params := map[string]interface{}{
		"userID": userID,
	}

	return errors.Wrapf(storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)),
		"failed to delete user achievements record for user.ID:%v", userID)
}

func (r *repository) GetUserProgress(userID users.UserID) (*UserProgress, error) {
	res := new(userProgress)
	if err := r.db.GetTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", tarantool.StringKey{S: userID}, res); err != nil {
		return nil, errors.Wrapf(err, "unable to get user_achievements record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no user achievements record for userID:%v", userID)
	}

	return res.UserProgress(), nil
}

func (r *repository) InsertUserProgress(userID users.UserID) error {
	ua := &userProgress{
		UserID:                            userID,
		Balance:                           0,
		T1Referrals:                       0,
		AgendaReferrals:                   0,
		LastMiningStartedAt:               0,
		MaxConsecutiveMiningSessionsCount: 0,
		TotalUserReferalPings:             0,
	}

	return errors.Wrapf(r.db.InsertTyped(userProgressSpace, ua, &[]*userProgress{}),
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

	return errors.Wrapf(r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, incrementOps, &[]*userProgress{}),
		"failed to update %v record with the new count of T1 referals for userID:%v", userProgressSpace, userID)
}

func (r *repository) UpdateConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS},   // | last_mining_started_at = lastStartedTS.
		{Op: "+", Field: maxConsecutiveMiningSessionsCountField, Arg: 1}, // | max_count +=1.
	}

	return errors.Wrapf(r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1",
		key, ops, &[]*userProgress{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", userProgressSpace, userID)
}

func (r *repository) ResetConsecutiveMiningSessionsCount(userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: lastMinintStartedAtField, Arg: lastStartedTS},   // | last_mining_started_at = lastStartedTS.
		{Op: "=", Field: maxConsecutiveMiningSessionsCountField, Arg: 1}, // | max_count = 1 (lastest session just started).
	}

	return errors.Wrapf(r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1",
		key, ops, &[]*userProgress{}),
		"failed to update %v record with the new count consecutive mining sessions for userID:%v", userProgressSpace, userID)
}

func (u *userProgress) UserProgress() *UserProgress {
	return &UserProgress{
		UserID:                            u.UserID,
		Balance:                           u.Balance,
		T1Referrals:                       u.T1Referrals,
		AgendaReferrals:                   u.AgendaReferrals,
		LastMiningStartedAt:               u.LastMiningStartedAt,
		MaxConsecutiveMiningSessionsCount: u.MaxConsecutiveMiningSessionsCount,
		TotalUserReferalPings:             u.TotalUserReferalPings,
	}
}
