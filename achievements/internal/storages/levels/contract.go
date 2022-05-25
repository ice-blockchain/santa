package levels

import (
	"context"

	"github.com/framey-io/go-tarantool"
)

// Public API.
type (
	UserID = string

	Repository interface {
		IncrementUserLevel(ctx context.Context, userID UserID) error
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

	// | agendaReferralsSource is source processor to increment user's levels on referred users from agenda (Levels -> 9-11)
	agendaReferralsSource struct {
		r Repository
	}
)
