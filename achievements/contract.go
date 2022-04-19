// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"io"
)

// Public API.

type (
	UserAchievements struct {
		Level  uint64           `json:"level" example:"11"`
		Role   string           `json:"role" example:"AMBASSADOR"`
		Badges []*BadgeOverview `json:"badges,omitempty"`
		Tasks  []*Task          `json:"tasks,omitempty"`
	}
	BadgeInventory struct {
		Badge
		Achieved bool `json:"achieved" example:"false"`
		// The percentage of all the users that have this badge.
		GlobalAchievementPercentage float64 `json:"globalAchievementPercentage" example:"25.5"`
	}
	BadgeOverview struct {
		Badge
		Position struct {
			X     uint64 `json:"x" example:"3"`
			OutOf uint64 `json:"outOf" example:"10"`
		} `json:"position"`
	}
	Task struct {
		Achieved bool   `json:"achieved" example:"false"`
		Name     string `json:"name" example:"CLAIM_USERNAME"`
		Index    uint64 `json:"index" example:"0"`
	}
	Badge struct {
		Name             string `json:"name" example:"ICE Breaker"`
		Type             string `json:"type" example:"SOCIAL"`
		ProgressInterval struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		} `json:"interval"`
	}
	Repository interface {
		io.Closer
	}
	Processor interface {
		Repository
		CheckHealth(context.Context) error
	}
)

// Private API.

type (
	repository struct {
	}
	processor struct {
	}
)
