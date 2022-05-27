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

func NewAgendaReferralsSource(db tarantool.Connector) messagebroker.Processor {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &agendaReferralsSource{
		r: newRepository(db),
	}
}

func (a *agendaReferralsSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	refFromAgendaCount := new(progress.AgendaReferralsCount)
	if err := json.Unmarshal(message.Value, refFromAgendaCount); err != nil {
		return errors.Wrapf(err, "levels/taskSource: cannot unmarshall %v into %#v", string(message.Value), refFromAgendaCount)
	}

	// Increment user's level for each task completion (Levels -> #7).
	return errors.Wrapf(a.achieveLevelsForReferralsFromAgenda(ctx, refFromAgendaCount.UserID, refFromAgendaCount.AgendaReferralsCount),
		"levels/agendaReferralsSource: failed to increment user's level for referred users from agenda:%#v", refFromAgendaCount)
}

func (a *agendaReferralsSource) achieveLevelsForReferralsFromAgenda(ctx context.Context, userID UserID, refCount uint64) error {
	achievedLevelName, isNewLevelAchieved := cfg.Levels.AgendaReferrals[refCount]
	if isNewLevelAchieved {
		if err := a.r.achieveUserLevel(ctx, userID, achievedLevelName); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to %v referrals from agenda for userID:%v", refCount, userID)
		}
	}

	return nil
}
