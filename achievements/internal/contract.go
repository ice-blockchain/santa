package internal

import (
	"context"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/freezer/economy"
	"github.com/ice-blockchain/santa/achievements/internal/storages/badges"
	"github.com/ice-blockchain/santa/achievements/internal/storages/tasks"
)

type (
	UserID = string

	UserSource interface {
		ProcessUser(ctx context.Context, snapshot *users.UserSnapshot) error
	}
	EconomyMiningSource interface {
		ProcessMiningStart(ctx context.Context, userID UserID, miningStarted *economy.MiningStarted) error
	}
	AchievedTasksSource interface {
		ProcessAchievedTask(ctx context.Context, userID UserID, task *tasks.AchievedTaskMessage) error
	}
	AchievedBadgesSource interface {
		ProcessAchievedBadge(ctx context.Context, userID UserID, badge *badges.AchievedBadgeMessage) error
	}
)
