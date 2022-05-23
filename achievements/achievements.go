// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"

	"github.com/framey-io/go-tarantool"
	"github.com/hashicorp/go-multierror"
	achievementprocessor "github.com/ice-blockchain/santa/achievements/internal/achievement-processor"
	economy_processor "github.com/ice-blockchain/santa/achievements/internal/economy-processor"
	user_processor "github.com/ice-blockchain/santa/achievements/internal/user-processor"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/log"
	"github.com/pkg/errors"
)

func New(ctx context.Context, cancel context.CancelFunc) Repository {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	db := storage.MustConnect(ctx, cancel, ddl, applicationYamlKey)

	return &repository{
		db: db,
	}
}

func (r *repository) Close() error {
	log.Info("closing achievements repository...")

	return errors.Wrap(r.db.Close(), "failed to close achievements repository")
}

func StartProcessor(ctx context.Context, cancel context.CancelFunc) Processor {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	db := storage.MustConnect(ctx, cancel, ddl, applicationYamlKey)
	mbProducer := messagebroker.MustConnect(context.Background(), applicationYamlKey)
	repo := &repository{db: db, mb: mbProducer}
	mbConsumer := messagebroker.MustConnectAndStartConsuming(context.Background(), cancel, applicationYamlKey, processors(repo, db))

	return &processor{
		close: func() error {
			result := make([]error, 0, 1+1+1)
			if err := db.Close(); err != nil {
				result = append(result, err)
			}
			if err := mbConsumer.Close(); err != nil {
				result = append(result, err)
			}
			if err := mbProducer.Close(); err != nil {
				result = append(result, err)
			}
			switch len(result) {
			case 1:
				return result[0]
			case 0:
				return nil
			default:
				return multierror.Append(nil, result...)
			}
		},
		WriteRepository: repo,
	}
}

func processors(repo WriteRepository, db tarantool.Connector) map[messagebroker.Topic]messagebroker.Processor {
	return map[messagebroker.Topic]messagebroker.Processor{
		// May be it is better to iternate and look for topics name?
		// Because of current impementation requires topic to be in specific order in configuration.
		cfg.MessageBroker.ConsumingTopics[0]: user_processor.New(db, repo),
		cfg.MessageBroker.ConsumingTopics[1]: economy_processor.NewMiningEventProcessor(db, repo),
		cfg.MessageBroker.ConsumingTopics[2]: achievementprocessor.NewTaskProcessor(db, repo),
		cfg.MessageBroker.ConsumingTopics[3]: achievementprocessor.NewBadgeProcessor(db),
	}
}

func (p *processor) Close() error {
	log.Info("closing achievements processor...")

	return errors.Wrap(p.close(), "error closing achievements processor")
}

func (p *processor) CheckHealth(ctx context.Context) error {
	//nolint:nolintlint,godox // TODO implement me.
	return nil
}
