package achievements

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	"github.com/ice-blockchain/santa/achievements/internal"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
	"time"
)

func NewEcomonyMiningSource(db tarantool.Connector) internal.EconomyMiningSource {
	return &economyMiningSource{
		r: &repository{db: db},
	}
}

func (e *economyMiningSource) ProcessMiningStart(ctx context.Context, userID UserID, miningEvent *economy.MiningStarted) error {
	_, err := e.handleMiningSessionStart(userID, miningEvent.TS)
	if err != nil {
		return errors.Wrapf(err, "miningEventSourceProcessor: cannot handle user mining session for userID:%v", userID)
	}
	return nil
}

func (m *economyMiningSource) handleMiningSessionStart(userID UserID, lastStartedTS time.Time) (*consecutiveUserMiningSessions, error) {
	userMiningSessions, err := m.r.GetConsecutiveMiningSessions(userID)
	timeStartedNano := uint64(lastStartedTS.UTC().UnixNano())
	if errors.Is(err, storage.ErrNotFound) {
		// User's mining sessions not found - so it is his first session.
		if err = m.r.InsertConsecutiveMiningSessions(userID, timeStartedNano); err != nil {
			return nil, errors.Wrapf(err, "failed to insert user achievements record")
		}

		return &consecutiveUserMiningSessions{
			UserID:              userID,
			LastMiningStartedAt: timeStartedNano,
			MaxCount:            1,
		}, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "miningEventSourceProcessor: failed handle MiningStarted message")
	}
	// Session count found - update it.
	return m.handleConsecutiveSessionsUpdate(userMiningSessions, userID, timeStartedNano)
}

func (m *economyMiningSource) handleConsecutiveSessionsUpdate(
	userMiningSessions *ConsecutiveMiningSessions,
	userID UserID,
	timeStartedNano uint64,
) (*consecutiveUserMiningSessions, error) {
	// First we need to check if 10 hours passed from the previous session, if passed - reset counter to 0.
	if time.Since(time.Unix(0, int64(userMiningSessions.LastMiningStartedAt))) >= maxTimeBetweenConsecutiveMiningSessions {
		if err := m.r.ResetConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
			return nil, errors.Wrapf(err, "failed to reset consecutive mining sessions for userID:%v", userID)
		}

		return &consecutiveUserMiningSessions{
			UserID:              userID,
			LastMiningStartedAt: timeStartedNano,
			MaxCount:            1,
		}, nil
	}
	// And if not - increment counter of consecutive sessions.
	if err := m.r.UpdateConsecutiveMiningSessionsCount(userID, timeStartedNano); err != nil {
		return nil, errors.Wrapf(err, "failed to update consecutive mining sessions for userID:%v", userID)
	}

	return &consecutiveUserMiningSessions{
		UserID:              userID,
		LastMiningStartedAt: timeStartedNano,
		MaxCount:            userMiningSessions.MaxCount + 1,
	}, nil
}
