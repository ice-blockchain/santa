package tasks

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/freezer/economy"
	"github.com/ice-blockchain/santa/achievements/internal"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewEcomonyMiningSource(db tarantool.Connector, mb messagebroker.Client) internal.EconomyMiningSource {
	return &economyMiningSource{
		r: New(db, mb),
	}
}

func (m *economyMiningSource) ProcessMiningStart(ctx context.Context, userID UserID, miningEvent *economy.MiningStarted) error {
	// TODO check if task is not achieved yet
	return errors.Wrapf(m.r.AchieveTask(ctx, userID, "TASK2"), "Failed to achieve task for the first mining session for userID:%v", userID)
}
