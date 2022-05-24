package levels

import (
	"context"
	"encoding/json"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressSource(db tarantool.Connector) messagebroker.Processor {
	return &progressSource{
		r: &repository{db: db},
	}
}

func (m *progressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "levels/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}

	if err := m.achieveLevelsForConsecutiveMiningSessions(ctx, userProgress); err != nil {
		return errors.Wrapf(err, "levels/progressSource: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

func (m *progressSource) achieveLevelsForConsecutiveMiningSessions(ctx context.Context, progress *progress.UserProgress) error {
	switch progress.MaxConsecutiveMiningSessionsCount {
	case 90, 60, 30, 10, 5: //nolint:gomnd,nolintlint // Consecutive mining sessions increments level (Levels -> #2-6).
		if err := m.r.IncrementUserLevel(ctx, progress.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to %v consecutive mining sessions for userID:%v", progress.MaxConsecutiveMiningSessionsCount, progress.UserID)
		}
	case 1: //nolint:gomnd,nolintlint // First mining - increment user's level (Levels -> #1) and achieve task (Tasks #2).
		if err := m.r.IncrementUserLevel(ctx, progress.UserID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to first mining session for userID:%v", progress.UserID)
		}
	}

	return nil
}
