package levels

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewEconomyMiningSource(db tarantool.Connector) messagebroker.Processor {
	return &economyMiningSource{
		r: &repository{db: db},
		p: progress.NewRepository(db),
	}
}

func (m *economyMiningSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	miningEvent := new(economy.MiningStarted)
	if err := json.Unmarshal(message.Value, miningEvent); err != nil {
		return errors.Wrapf(err, "levels/economyMiningSource: cannot unmarshall %v into %#v", string(message.Value), miningEvent)
	}

	userProgress, err := m.p.GetUserProgress(userID)
	if err != nil {
		return errors.Wrapf(err, "levels/miningEventSourceProcessor: cannot get user progress for userID:%v", userID)
	}

	if err := m.achieveLevelsForConsecutiveMiningSessions(ctx, userProgress); err != nil {
		return errors.Wrapf(err, "levels/miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

// TODO think about parameter (pass the inserted one from achievements package?)
func (m *economyMiningSource) achieveLevelsForConsecutiveMiningSessions(ctx context.Context, sessions *progress.UserProgress) error {
	//nolint:godot,nolintlint // FIXME: handle decrement in case when user spends >10h and continie mining,
	// counter resets to 0, and we'll be here again with the same value.
	switch sessions.MaxConsecutiveMiningSessionsCount {
	case 90, 60, 30, 10, 5: //nolint:gomnd,nolintlint // Consecutive mining sessions increments level (Levels -> #2-6).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to %v consecutive mining sessions for userID:%v", sessions.MaxConsecutiveMiningSessionsCount, sessions.UserID)
		}
	case 1: //nolint:gomnd,nolintlint // First mining - increment user's level (Levels -> #1) and achieve task (Tasks #2).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to first mining session for userID:%v", sessions.UserID)
		}
	}

	return nil
}
