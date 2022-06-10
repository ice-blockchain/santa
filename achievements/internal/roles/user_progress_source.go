// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/achievements/internal/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

func NewProgressProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &userProgressSource{
		r: NewRepository(db, mb).(*repository),
	}
}

func (u *userProgressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}

	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "achievements/current_user_roles: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}

	currentRole, err := u.r.getCurrentUserRole(userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "error getting current user role for userID:%v", userProgress.UserID)
	}

	var newRole string

	switch {
	case currentRole == "":
		newRole = roleNamePioneer
	case userProgress.T1Referrals >= requiredReferralsForAmbassadorRole && currentRole == roleNamePioneer:
		newRole = roleNameAmbassador
	case userProgress.T1Referrals < requiredReferralsForAmbassadorRole && currentRole == roleNameAmbassador:
		newRole = roleNamePioneer
	default:
		return nil
	}

	return errors.Wrapf(u.r.upsertCurrentUserRole(ctx, userProgress.UserID, newRole),
		"error setting %v role for UserID:%v", newRole, userProgress.UserID)
}
