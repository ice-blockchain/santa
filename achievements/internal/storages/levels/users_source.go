package levels

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/santa/achievements/internal"
	"github.com/pkg/errors"
)

func NewUserSource(db tarantool.Connector) internal.UserSource {
	return &userSource{
		r: &repository{db: db},
	}
}

func (u *userSource) ProcessUser(ctx context.Context, user *users.UserSnapshot) error {
	return errors.Wrapf(u.achieveLevels(ctx, user), "Failed to increment user's level for the phone number confirmation")
}

func (u *userSource) achieveLevels(ctx context.Context, user *users.UserSnapshot) error {
	// New level for user (Levels -> 8 Confirm phone number)
	// it seems eskimo can send unconfirmed number at initial user creation for now
	// but in case of user modification (before != nil) it sends confirmed number, catch it here.
	if user.PhoneNumber != "" && user.Before != nil && user.Before.PhoneNumber == "" {
		err := u.r.IncrementUserLevel(ctx, user.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to increment user's level for the phone number confirmation userID:%v", user.ID)
		}
	}

	return nil
}
