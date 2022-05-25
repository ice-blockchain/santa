package levels

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewAgendaReferralsSource(db tarantool.Connector) messagebroker.Processor {
	return &agendaReferralsSource{r: &repository{db: db}}
}

func (a *agendaReferralsSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	refFromAgendaCount := new(progress.AgendaReferralsCount)
	if err := json.Unmarshal(message.Value, refFromAgendaCount); err != nil {
		return errors.Wrapf(err, "levels/taskSource: cannot unmarshall %v into %#v", string(message.Value), refFromAgendaCount)
	}

	// Increment user's level for each task completion (Levels -> #7).
	return errors.Wrapf(a.achieveLevelsForReferralsFromAgenda(ctx, refFromAgendaCount.UserID, refFromAgendaCount.AgendaReferralsCount),
		"levels/agendaReferralsSource: failed to increment user's level for referred users from agenda:%#v", refFromAgendaCount)
}

func (a *agendaReferralsSource) achieveLevelsForReferralsFromAgenda(ctx context.Context, userID UserID, refCount uint64) error {
	switch refCount {
	case 10, 5, 1: //nolint:gomnd,nolintlint //  (Levels -> #9-11).
		if err := a.r.IncrementUserLevel(ctx, userID); err != nil {
			return errors.Wrapf(err, "failed to increment user's level due to %v referrals from agenda for userID:%v", refCount, userID)
		}
	}

	return nil
}
