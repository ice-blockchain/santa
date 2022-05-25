package roles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewCurrentUserRolesProcessor(db tarantool.Connector) messagebroker.Processor {
	return &userProgressSource{
		r: &repository{db: db},
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

	currentRole, err := u.r.GetCurrentUserRole(userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "error get current user rules count for UserID:%v", userProgress.UserID)
	}

	return errors.Wrapf(u.processUserProgress(currentRole, userProgress), "error processing user progress")
}

func (u *userProgressSource) processUserProgress(role string, userProgress *progress.UserProgress) error {
	switch {
	case role == "":
		if err := u.r.UpsertCurrentUserRole(userProgress.UserID, "PIONEER"); err != nil {
			return errors.Wrapf(err, "error upsert current user role for UserID:%v", userProgress.UserID)
		}

	case role == "PIONEER" && userProgress.T1Referrals == 100:
		if err := u.r.UpsertCurrentUserRole(userProgress.UserID, "AMBASSADOR"); err != nil {
			return errors.Wrapf(err, "error insert current user role for UserID:%v", userProgress.UserID)
		}
	}

	return nil
}
