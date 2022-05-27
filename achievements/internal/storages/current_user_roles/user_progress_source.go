// SPDX-License-Identifier: BUSL-1.1

package currentuserroles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressSource(db tarantool.Connector) messagebroker.Processor {
	return &userProgressSource{
		r: newRepository(db),
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
		return errors.Wrapf(err, "error get current user role for UserID:%v", userProgress.UserID)
	}

	return errors.Wrapf(u.processUserProgress(currentRole, userProgress), "error processing user progress")
}

func (u *userProgressSource) processUserProgress(role string, userProgress *progress.UserProgress) error {
	switch {
	case role == "":
		if err := u.r.upsertCurrentUserRole(userProgress.UserID, "PIONEER"); err != nil {
			return errors.Wrapf(err, "error setting PIONEER role for UserID:%v", userProgress.UserID)
		}

	case role == "PIONEER" && userProgress.T1Referrals == ambassadorLimit:
		if err := u.r.upsertCurrentUserRole(userProgress.UserID, "AMBASSADOR"); err != nil {
			return errors.Wrapf(err, "error setting AMBASSADOR role for UserID:%v", userProgress.UserID)
		}
	}

	return nil
}
