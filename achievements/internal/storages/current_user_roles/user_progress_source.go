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

	rolesCount, err := u.r.GetCurrentUserRolesCount(userProgress.UserID)
	if err != nil {
		return errors.Wrapf(err, "error get current user rules count for UserID:%v", userProgress.UserID)
	}

	return errors.Wrapf(u.processUserProgress(rolesCount, userProgress), "error processing user progress")
}

func (u *userProgressSource) processUserProgress(rolesCount uint64, userProgress *progress.UserProgress) error {
	switch {
	case rolesCount == 0:
		if err := u.r.InsertCurrentUserRole(userProgress.UserID, "PIONEER"); err != nil {
			return errors.Wrapf(err, "error insert current user role for UserID:%v", userProgress.UserID)
		}

	case rolesCount == 1 && userProgress.T1Referrals == 100:
		if err := u.r.InsertCurrentUserRole(userProgress.UserID, "AMBASSADOR"); err != nil {
			return errors.Wrapf(err, "error insert current user role for UserID:%v", userProgress.UserID)
		}
	}

	return nil
}
