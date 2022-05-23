package levels

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	"github.com/ice-blockchain/santa/achievements/internal"
	"github.com/ice-blockchain/santa/achievements/internal/storages/achievements"
	"github.com/pkg/errors"
)

func NewEcomonyMiningSource(db tarantool.Connector) internal.EconomyMiningSource {
	return &economyMiningSource{
		r: &repository{db: db},
	}
}

func (m *economyMiningSource) ProcessMiningStart(ctx context.Context, userID UserID, miningEvent *economy.MiningStarted) error {
	var updatedSessions *achievements.ConsecutiveMiningSessions
	updatedSessions = nil // TODO pass value
	if err := m.achieveLevelsForConsecutiveMiningSessions(ctx, updatedSessions); err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

// TODO think about parameter (pass the inserted one from achievements package?)
func (m *economyMiningSource) achieveLevelsForConsecutiveMiningSessions(ctx context.Context, sessions *achievements.ConsecutiveMiningSessions) error {
	//nolint:godot,nolintlint // FIXME: handle decrement in case when user spends >10h and continie mining,
	// counter resets to 0, and we'll be here again with the same value.
	switch sessions.MaxCount {
	case 90, 60, 30, 10, 5: //nolint:gomnd,nolintlint // Consecutive mining sessions increments level (Levels -> #2-6).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to %v consecutive mining sessions", sessions.MaxCount)
		}
	case 1: //nolint:gomnd,nolintlint // First mining - increment user's level (Levels -> #1) and achieve task (Tasks #2).
		if err := m.r.IncrementUserLevel(ctx, sessions.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to first mining session for userID:%v", sessions.UserID)
		}
	}

	return nil
}
