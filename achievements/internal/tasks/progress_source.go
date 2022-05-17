// SPDX-License-Identifier: BUSL-1.1

package tasks

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressProcessor(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &progressSource{
		r: NewRepository(db, mb),
	}
}

func (p *progressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "tasks/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}
	// Tasks -> #6 (Invite 5 friends).
	if userProgress.T1Referrals >= cfg.Tasks.T1Referrals {
		if err := p.r.CompleteTask(ctx, userProgress.UserID, taskGetFiveReferrals); err != nil && !errors.Is(err, errAlreadyAchieved) {
			return errors.Wrapf(err, "tasks/progressSource: failed to achieve %v task for user %v", taskGetFiveReferrals, userProgress.UserID)
		}
	} else {
		if err := p.r.UnCompleteTask(ctx, userProgress.UserID, taskGetFiveReferrals); err != nil && !errors.Is(err, errNotAchieved) {
			return errors.Wrapf(err, "tasks/progressSource: failed to achieve %v task for user %v", taskGetFiveReferrals, userProgress.UserID)
		}
	}

	return nil
}
