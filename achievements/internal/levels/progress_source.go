// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/achievements/internal/progress"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

func NewProgressProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &progressSource{
		r: newRepository(db, mb).(*repository),
	}
}

func (p *progressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userID := message.Key
	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "levels/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}
	levels, err := p.r.getUserLevels(ctx, userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch achieved user levels")
	}
	for _, level := range levels {
		if err := p.achieveLevelsForConsecutiveMiningSessions(ctx, userProgress, level); err != nil {
			return errors.Wrapf(err, "levels/progressSource: cannot handle user mining session for userID:%v", userID)
		}
	}

	return nil
}

func (p *progressSource) achieveLevelsForConsecutiveMiningSessions(
	ctx context.Context,
	userProgress *progress.UserProgress,
	level *levelWithAchieved,
) error {
	targetSessions, isMiningSessionLevel := cfg.Levels.ConsecutiveMiningSessions[level.Name]
	if isMiningSessionLevel {
		if (!level.Achieved) && userProgress.MaxConsecutiveMiningSessionsCount >= targetSessions {
			return errors.Wrapf(achieveLevelAndHandleError(ctx, p.r, userProgress.UserID, level.Name),
				"failed to increment user's level due to %v consecutive mining sessions for userID:%v",
				userProgress.MaxConsecutiveMiningSessionsCount, userProgress.UserID)
		} else if level.Achieved && userProgress.MaxConsecutiveMiningSessionsCount < targetSessions {
			return errors.Wrapf(unachieveLevelAndHandleError(ctx, p.r, userProgress.UserID, level.Name),
				"failed to decrement user's level due to %v consecutive mining sessions for userID:%v",
				userProgress.MaxConsecutiveMiningSessionsCount, userProgress.UserID)
		}
	}

	return nil
}
