package roles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func New(db tarantool.Connector) messagebroker.Processor {
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

	return nil
}
