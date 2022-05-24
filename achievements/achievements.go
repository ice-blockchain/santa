// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"

	"github.com/framey-io/go-tarantool"
	"github.com/hashicorp/go-multierror"
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
	mbConsumer := messagebroker.MustConnectAndStartConsuming(context.Background(), cancel, applicationYamlKey, processors(mbProducer, db))

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
	}
}

func processors(mb messagebroker.Client, db tarantool.Connector) map[messagebroker.Topic]messagebroker.Processor {
	return map[messagebroker.Topic]messagebroker.Processor{
		// TODO: fill with new processors
		cfg.MessageBroker.ConsumingTopics[0]: progress.New(db),
		//cfg.MessageBroker.ConsumingTopics[1]: economy_processor.NewMiningEventProcessor(db, repo),
		//cfg.MessageBroker.ConsumingTopics[2]: achievementprocessor.NewTaskProcessor(db, repo),
		//cfg.MessageBroker.ConsumingTopics[3]: achievementprocessor.NewBadgeProcessor(db),
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
