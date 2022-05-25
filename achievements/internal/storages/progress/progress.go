package progress

import (
	"context"
	"encoding/json"
	"math"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) ReadRepository {
	// Maybe it is better to pass global cfg from achievements here?
	var config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`
	}
	appCfg.MustLoadFromKey("achievements", &config)

	return &repository{
		db:                               db,
		mb:                               mb,
		publishUpdatedProgressTopic:      config.MessageBroker.Topics[2].Name,
		publishAgendaReferralsCountTopic: config.MessageBroker.Topics[3].Name,
	}
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

func (r *repository) InsertUserProgress(ctx context.Context, user *users.User) error {
	ua := &userProgress{
		UserID:                            user.ID,
		Balance:                           0,
		T1Referrals:                       0,
		AgendaPhoneNumbersHashes:          user.AgendaPhoneNumberHashes,
		LastMiningStartedAt:               0,
		MaxConsecutiveMiningSessionsCount: 0,
		TotalUserReferalPings:             0,
	}
	res := []*userProgress{}
	if err := r.db.InsertTyped(userProgressSpace, ua, &res); err != nil {
		return errors.Wrapf(err,
			"failed to insert user achievements record for user.ID:%v", user.ID)
	}

	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0].UserProgress()), "progress: failed to send updated progress message for UserID:%v", user.ID)
}

func (r *repository) UpdateT1ReferralsCount(ctx context.Context, userID users.UserID, diff int64) error {
	key := tarantool.StringKey{S: userID}
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldT1Referrals, Arg: diff},
	}
	res := []*userProgress{}
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, incrementOps, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the new count of T1 referals for userID:%v", userProgressSpace, userID)
	}

	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0].UserProgress()), "progress: failed to send updated progress message for UserID:%v", userID)
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

func (r *repository) UpdateConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: fieldLastMiningStartedAt, Arg: lastStartedTS},   // | last_mining_started_at = lastStartedTS.
		{Op: "+", Field: fieldMaxConsecutiveMiningSessionsCount, Arg: 1}, // | max_count +=1.
	}
	res := []*userProgress{}
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, ops, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the new count consecutive mining sessions for userID:%v", userProgressSpace, userID)
	}

	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0].UserProgress()),
		"progress/mining sessions: failed to send updated progress message for UserID:%v", userID)
}

func (r *repository) ResetConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS uint64) error {
	key := tarantool.StringKey{S: userID}
	ops := []tarantool.Op{
		{Op: "=", Field: fieldLastMiningStartedAt, Arg: lastStartedTS},   // | last_mining_started_at = lastStartedTS.
		{Op: "=", Field: fieldMaxConsecutiveMiningSessionsCount, Arg: 1}, // | max_count = 1 (lastest session just started).
	}
	res := []*userProgress{}
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, ops, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the new count consecutive mining sessions for userID:%v", userProgressSpace, userID)
	}

	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0].UserProgress()), "progress: failed to send updated progress message for UserID:%v", userID)
}

func (r *repository) sendUpdatedUserProgress(ctx context.Context, p *UserProgress) error {
	b, err := json.Marshal(p)
	if err != nil {
		return errors.Wrapf(err, "[user-progress] failed to marshal %#v", p)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     p.UserID,
		Topic:   r.publishUpdatedProgressTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[user-progress] failed to send message to broker")
}

func (u *userProgress) UserProgress() *UserProgress {
	return &UserProgress{
		UserID:                            u.UserID,
		Balance:                           u.Balance,
		T1Referrals:                       u.T1Referrals,
		AgendaPhoneNumbersHashes:          u.AgendaPhoneNumbersHashes,
		LastMiningStartedAt:               u.LastMiningStartedAt,
		MaxConsecutiveMiningSessionsCount: u.MaxConsecutiveMiningSessionsCount,
		TotalUserReferralPings:            u.TotalUserReferalPings,
	}
}
