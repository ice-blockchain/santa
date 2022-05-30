// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"sync"

	"github.com/framey-io/go-tarantool"
	"github.com/hashicorp/go-multierror"
	"github.com/ice-blockchain/santa/achievements/internal/storages/badges"
	"github.com/ice-blockchain/santa/achievements/internal/storages/levels"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	"github.com/ice-blockchain/santa/achievements/internal/storages/tasks"
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

// nolint:funlen // A lot of processors here.
func processors(mb messagebroker.Client, db tarantool.Connector) map[messagebroker.Topic]messagebroker.Processor {
	return map[messagebroker.Topic]messagebroker.Processor{
		// | users-events .
		cfg.MessageBroker.ConsumingTopics[0]: newProxyProcessorWithAsync(false, // It is crucial to insert user_progress first.
			progress.NewUsersProcessor(db, mb),
			tasks.NewUsersProcessor(db, mb),
			levels.NewUserSource(db, mb),
		),
		// | economy-mining .
		cfg.MessageBroker.ConsumingTopics[1]: newProxyProcessorWithAsync(true,
			progress.NewEconomyMiningProcessor(db, mb),
			tasks.NewEconomyMiningProcessor(db, mb),
		),
		// | achievements-tasks .
		cfg.MessageBroker.ConsumingTopics[2]: newProxyProcessorWithAsync(false, // Only one processor, so we dont need async.
			levels.NewTaskSource(db, mb),
		),
		// | achievements-badges .
		cfg.MessageBroker.ConsumingTopics[3]: newProxyProcessorWithAsync(false,
			badges.NewAchievedBadgesProcessor(db),
		),
		// | achievements-progress .
		cfg.MessageBroker.ConsumingTopics[4]: newProxyProcessorWithAsync(true,
			tasks.NewProgressProcessor(db, mb),
			levels.NewProgressSource(db, mb),
			badges.NewProgressSource(db, mb),
			// Roles upcoming processor to be here.
		),
		// | achievements-agenda-referrals .
		cfg.MessageBroker.ConsumingTopics[5]: newProxyProcessorWithAsync(false,
			levels.NewAgendaReferralsSource(db, mb),
		),
		// | achievements-levels .
		cfg.MessageBroker.ConsumingTopics[6]: newProxyProcessorWithAsync(false,
			badges.NewLevelSource(db, mb),
		),
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

func newProxyProcessorWithAsync(async bool, processors ...messagebroker.Processor) messagebroker.Processor {
	return &proxyProcessor{internalProcessors: processors, asyncProcessing: async}
}

func (proxy *proxyProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if proxy.asyncProcessing {
		return proxy.processAsync(ctx, message)
	}

	return proxy.process(ctx, message)
}

func (proxy *proxyProcessor) processAsync(ctx context.Context, message *messagebroker.Message) error {
	var wg sync.WaitGroup
	errs := make([]error, 0, len(proxy.internalProcessors))
	errsChan := make(chan error, len(proxy.internalProcessors))
	wg.Add(len(proxy.internalProcessors))
	for _, processor := range proxy.internalProcessors {
		go func(p messagebroker.Processor) {
			defer wg.Done()
			err := p.Process(ctx, message)
			if err != nil {
				errsChan <- err
			}
		}(processor)
	}
	wg.Wait()
	close(errsChan)
	for err := range errsChan {
		errs = append(errs, err)
	}

	return proxy.handleError(errs)
}

func (proxy *proxyProcessor) process(ctx context.Context, message *messagebroker.Message) error {
	errs := make([]error, 0, len(proxy.internalProcessors))
	for _, processor := range proxy.internalProcessors {
		if err := processor.Process(ctx, message); err != nil {
			errs = append(errs, errors.Wrapf(err, "proxyProcessor: failed to process %v message on %T", string(message.Value), processor))
		}
	}

	return proxy.handleError(errs)
}

func (proxy *proxyProcessor) handleError(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return multierror.Append(nil, errs...)
	}
}
