// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &progressSource{
		r: newRepository(db, mb),
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

	if err := p.achieveLevelsForConsecutiveMiningSessions(ctx, userProgress); err != nil {
		return errors.Wrapf(err, "levels/progressSource: cannot handle user mining session for userID:%v", userID)
	}

	return nil
}

func (p *progressSource) achieveLevelsForConsecutiveMiningSessions(ctx context.Context, userProgress *progress.UserProgress) error {
	achievedLevelName, isNewLevelAchieved := cfg.Levels.ConsecutiveMiningSessions[userProgress.MaxConsecutiveMiningSessionsCount]
	if isNewLevelAchieved {
		if err := p.r.achieveUserLevel(ctx, userProgress.UserID, achievedLevelName); err != nil {
			return errors.Wrapf(err,
				"failed to increment user's level due to %v consecutive mining sessions for userID:%v",
				userProgress.MaxConsecutiveMiningSessionsCount, userProgress.UserID)
		}
	}

	return nil
}
