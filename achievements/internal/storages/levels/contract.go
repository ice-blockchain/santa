// SPDX-License-Identifier: BUSL-1.1

package levels

import (
	"context"

	"github.com/framey-io/go-tarantool"
)

// Public API.
type (
	UserID     = string
	LevelName  = string
	Repository interface {
		achieveUserLevel(ctx context.Context, userID UserID, levelName LevelName) error
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
	}

	// | taskSource is source processor to increment user's levels on task completion ( each task 1 level, Levels -> #7 ).
	taskSource struct {
		r Repository
	}
	// | userSource is source processor to increment user's levels on phone number confirmation (Levels -> #8).
	userSource struct {
		r Repository
	}
	// | progressSource is source processor to increment user's levels on consecutive mining sessions.
	progressSource struct {
		r Repository
	}

	// | agendaReferralsSource is source processor to increment user's levels on referred users from agenda (Levels -> 9-11).
	agendaReferralsSource struct {
		r Repository
	}

	config struct {
		Levels struct {
			AgendaReferrals           map[uint64]string `yaml:"agendaReferrals"`
			ConsecutiveMiningSessions map[uint32]string `yaml:"consecutiveMiningSessions"`
			TaskCompletion            map[string]string `yaml:"taskCompletion"`
		} `yaml:"levels"`
	}
)

const (
	levelForPhoneNumberConfirmation = "L13"
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config
