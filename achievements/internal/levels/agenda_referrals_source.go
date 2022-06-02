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

func NewAgendaReferralsProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &agendaReferralsSource{
		r: newRepository(db, mb).(*repository),
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
	levels, err := a.r.getUserLevels(ctx, refFromAgendaCount.UserID)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch achieved user levels")
	}
	for _, level := range levels {
		if err := a.achieveLevelsForReferralsFromAgenda(ctx, refFromAgendaCount.UserID,
			refFromAgendaCount.AgendaReferralsCount, level); err != nil {
			return errors.Wrapf(err,
				"levels/agendaReferralsSource: failed to increment user's level for referred users from agenda:%#v", refFromAgendaCount)
		}
	}

	return nil
}

func (a *agendaReferralsSource) achieveLevelsForReferralsFromAgenda(
	ctx context.Context,
	userID UserID,
	referralsCount uint64,
	level *levelWithAchieved,
) error {
	targetReferralsCount, isReferralLevel := cfg.Levels.AgendaReferrals[level.Name]
	if isReferralLevel {
		if (!level.Achieved) && referralsCount >= targetReferralsCount {
			return errors.Wrapf(achieveLevelAndHandleError(ctx, a.r, userID, level.Name),
				"failed to increment user's level due to %v referrals from agenda for userID:%v", referralsCount, userID)
		} else if level.Achieved && referralsCount < targetReferralsCount {
			return errors.Wrapf(unachieveLevelAndHandleError(ctx, a.r, userID, level.Name),
				"failed to decrement user's level due to %v referrals from agenda for userID:%v", referralsCount, userID)
		}
	}

	return nil
}
