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
		return errors.Wrapf(err, "tasks/progressSource: cannot unmarshall %v into %#v", string(message.Value), userProgress)
	}
	badgesToAchieve, err := p.r.GetBadgesWithCompletedRequirements(userProgress)
	if err != nil {
		return errors.Wrapf(err, "badges/progressSource: failed to check if user fulfill badge requirments, userID:%v", userProgress.UserID)
	}
	for _, badgeName := range badgesToAchieve {
		err = p.r.AchieveBadge(ctx, userProgress.UserID, badgeName)
		if err != nil && !errors.Is(err, ErrAlreadyAchieved) {
			return errors.Wrapf(err, "badges/progressSource: failed to achieve a badge:%v, userID:%v", badgeName, userProgress.UserID)
		}
	}
	return nil
}
