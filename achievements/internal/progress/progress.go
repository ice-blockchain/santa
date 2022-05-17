// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	stdlibtime "time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/coin"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/time"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db: db,
		mb: mb,
	}
}

func (r *repository) deleteUserProgress(userID users.UserID) error {
	sql := fmt.Sprintf(`DELETE FROM %v WHERE user_id = :userID`, userProgressSpace)
	params := map[string]interface{}{
		"userID": userID,
	}

	return errors.Wrapf(storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params)),
		"failed to delete user progress record for user.ID:%v", userID)
}

func (r *repository) getUserProgress(userID users.UserID) (*UserProgress, error) {
	res := new(UserProgress)
	if err := r.db.GetTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", tarantool.StringKey{S: userID}, res); err != nil {
		return nil, errors.Wrapf(err, "unable to get user_achievements record for userID:%v", userID)
	}
	if res.UserID == "" {
		return nil, errors.Wrapf(storage.ErrNotFound, "no user achievements record for userID:%v", userID)
	}

	return res, nil
}

func (r *repository) insertUserProgress(ctx context.Context, user *users.User) error {
	ua := &UserProgress{
		UserID:                   user.ID,
		Balance:                  coin.NewAmountUint64(0),
		AgendaPhoneNumbersHashes: user.AgendaPhoneNumberHashes,
	}
	res := []*UserProgress{}
	if err := r.db.InsertTyped(userProgressSpace, ua, &res); err != nil {
		return errors.Wrapf(err,
			"failed to insert user progress record for user.ID:%v", user.ID)
	}
	if len(res) == 0 {
		return nil
	}
	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0]), "progress: failed to send updated progress message for UserID:%v", user.ID)
}

func (r *repository) updateT1ReferralsCount(ctx context.Context, userID users.UserID, diff int64) error {
	key := tarantool.StringKey{S: userID}
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldT1Referrals, Arg: diff},
	}
	res := []*UserProgress{}
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, incrementOps, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the new count of T1 referals for userID:%v", userProgressSpace, userID)
	}
	if len(res) == 0 {
		return nil
	}
	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0]), "progress: failed to send updated progress message for UserID:%v", userID)
}

func (r *repository) incrementOrDecrementTotalUsersCount(diff int64) error {
	op := "+"
	if math.Signbit(float64(diff)) {
		op = "-"
	}
	incrementOps := []tarantool.Op{
		{Op: op, Field: fieldGlobalValue, Arg: diff},
	}

	return errors.Wrap(r.db.UpsertAsync("GLOBAL", &global{Key: "TOTAL_USERS", Value: 1}, incrementOps).GetTyped(&[]*global{}),
		"failed to update global record the KEY = 'TOTAL_USERS'")
}

func (r *repository) updateConsecutiveMiningSessionsCount(ctx context.Context, userID UserID, lastStartedTS stdlibtime.Time, reset bool) error {
	key := tarantool.StringKey{S: userID}
	var op string
	if reset {
		op = "="
	} else {
		op = "+"
	}
	ops := []tarantool.Op{
		{Op: "=", Field: fieldLastMiningStartedAt, Arg: time.New(lastStartedTS.UTC())}, // | last_mining_started_at = lastStartedTS.
		{Op: op, Field: fieldMaxConsecutiveMiningSessionsCount, Arg: 1},                // | max_count +=1 or =1 (in case of reset).
	}
	res := []*UserProgress{}
	if err := r.db.UpdateTyped(userProgressSpace, "pk_unnamed_USER_PROGRESS_1", key, ops, &res); err != nil {
		return errors.Wrapf(err, "failed to update %v record with the new count consecutive mining sessions for userID:%v", userProgressSpace, userID)
	}
	if len(res) == 0 {
		return nil
	}
	return errors.Wrapf(r.sendUpdatedUserProgress(ctx, res[0]),
		"progress/mining sessions: failed to send updated progress message for UserID:%v", userID)
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
		Topic:   cfg.MessageBroker.Topics[2].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[user-progress] failed to send message to broker")
}
