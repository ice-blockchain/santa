// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/hashicorp/go-multierror"
	economy_processor "github.com/ice-blockchain/santa/achievements/internal/economy-processor"
	user_processor "github.com/ice-blockchain/santa/achievements/internal/user-processor"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"

	appCfg "github.com/ice-blockchain/wintr/config"
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
	mbConsumer := messagebroker.MustConnectAndStartConsuming(context.Background(), cancel, applicationYamlKey, processors(db))
	return &processor{
		close: func() error {
			errDB := db.Close()
			errMB := mbConsumer.Close()
			if errDB != nil && errMB == nil {
				return errors.Wrap(errDB, "failed to close processor")
			} else if errDB == nil && errMB != nil {
				return errors.Wrap(errMB, "failed to close processor")
			} else if errDB != nil && errMB != nil {
				return errors.Wrap(multierror.Append(nil, errMB, errDB), "failed to close processor")
			} else {
				return nil
			}
		},
		WriteRepository: &repository{
			db: db,
		},
	}
}

func processors(db tarantool.Connector) map[messagebroker.Topic]messagebroker.Processor {
	return map[messagebroker.Topic]messagebroker.Processor{
		// may be it is better to iternate and look for topics name?
		// because of current impementation requires topic to be in specific order in configuration
		cfg.MessageBroker.ConsumingTopics[0]: user_processor.New(db),
		cfg.MessageBroker.ConsumingTopics[1]: economy_processor.New(db),
	}
}

func (p *processor) Close() error {
	log.Info("closing achievements processor...")

	return errors.Wrap(p.close(), "error closing achievemnts processor")
}

func (p *processor) CheckHealth(ctx context.Context) error {
	//nolint:nolintlint,godox // TODO implement me.
	return nil
}
