// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
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

	var role string
	switch {
	case userProgress.T1Referrals == requiredReferralsForPioneerRole:
		role = "PIONEER"
	case userProgress.T1Referrals == requiredReferralsForAmbassadorRole:
		role = "AMBASSADOR"
	default:
		return nil
	}

	return errors.Wrapf(u.r.upsertCurrentUserRole(ctx, userProgress.UserID, role),
		"error setting %v role for UserID:%v", role, userProgress.UserID)
}
