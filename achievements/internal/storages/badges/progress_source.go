package badges

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewProgressSource(db tarantool.Connector, mb messagebroker.Client) messagebroker.Processor {
	return &progressSource{r: newRepository(db, mb)}
}

func (p *progressSource) Process(ctx context.Context, message *messagebroker.Message) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	userProgress := new(progress.UserProgress)
	if err := json.Unmarshal(message.Value, userProgress); err != nil {
		return errors.Wrapf(err, "badges/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}

	return errors.Wrapf(p.r.AchieveBadgesWithCompletedRequirements(ctx, userProgress),
		"badges/progressSource: failed to achieve completed user's badges for userID:%v", userProgress.UserID)
}
