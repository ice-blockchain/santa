// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	"sync"

	"github.com/framey-io/go-tarantool"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/achievements/internal/badges"
	"github.com/ice-blockchain/santa/achievements/internal/levels"
	"github.com/ice-blockchain/santa/achievements/internal/progress"
	"github.com/ice-blockchain/santa/achievements/internal/roles"
	"github.com/ice-blockchain/santa/achievements/internal/tasks"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/log"
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
	tr := tasks.NewRepository(db, mbProducer)

	return &processor{
		close:           closeAll(mbConsumer, mbProducer, db),
		tasksRepository: tr,
	}
}

func closeAll(mbConsumer, mbProducer messagebroker.Client, db tarantool.Connector) func() error {
	return func() error {
		var result *multierror.Error
		if err := mbConsumer.Close(); err != nil {
			result = multierror.Append(result, err)
		}
		if err := mbProducer.Close(); err != nil {
			result = multierror.Append(result, err)
		}
		if err := db.Close(); err != nil {
			result = multierror.Append(result, err)
		}

		return errors.Wrapf(result.ErrorOrNil(), "failed to close processor")
	}
}

// nolint:funlen // A lot of processors here.
func processors(mb messagebroker.Client, db tarantool.Connector) map[messagebroker.Topic]messagebroker.Processor {
	return map[messagebroker.Topic]messagebroker.Processor{
		// | users-events .
		cfg.MessageBroker.ConsumingTopics[0]: newSequentialProcessorGroup( // It is crucial to insert user_progress first.
			progress.NewUsersProcessor(db, mb),
			tasks.NewUsersProcessor(db, mb),
			levels.NewUsersProcessor(db, mb),
		),
		// | economy-mining .
		cfg.MessageBroker.ConsumingTopics[1]: newParallelProcessorGroup(
			progress.NewEconomyMiningProcessor(db, mb),
			tasks.NewEconomyMiningProcessor(db, mb),
		),
		// | achievements-tasks .
		cfg.MessageBroker.ConsumingTopics[2]: newSequentialProcessorGroup( // Only one processor, so we dont need parallel.
			levels.NewCompletedTaskProcessor(db, mb),
		),
		// | achievements-badges .
		cfg.MessageBroker.ConsumingTopics[3]: newSequentialProcessorGroup(
			badges.NewAchievedBadgesProcessor(db),
		),
		// | achievements-progress .
		cfg.MessageBroker.ConsumingTopics[4]: newParallelProcessorGroup(
			tasks.NewProgressProcessor(db, mb),
			levels.NewProgressProcessor(db, mb),
			badges.NewProgressProcessor(db, mb),
			roles.NewProgressProcessor(db, mb),
			// Roles upcoming processor to be here.
		),
		// | achievements-agenda-referrals .
		cfg.MessageBroker.ConsumingTopics[5]: newSequentialProcessorGroup(
			levels.NewAgendaReferralsProcessor(db, mb),
		),
		// | achievements-levels .
		cfg.MessageBroker.ConsumingTopics[6]: newSequentialProcessorGroup(
			badges.NewLevelProcessor(db, mb),
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

func newSequentialProcessorGroup(processors ...messagebroker.Processor) messagebroker.Processor {
	return &proxyProcessor{internalProcessors: processors, parallelProcessing: false}
}

func newParallelProcessorGroup(processors ...messagebroker.Processor) messagebroker.Processor {
	return &proxyProcessor{internalProcessors: processors, parallelProcessing: true}
}

func (proxy *proxyProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	if proxy.parallelProcessing {
		return proxy.processParallel(ctx, message)
	}

	return proxy.processSequential(ctx, message)
}

func (proxy *proxyProcessor) processParallel(ctx context.Context, message *messagebroker.Message) error {
	var wg sync.WaitGroup
	errsChan := make(chan error, len(proxy.internalProcessors))
	wg.Add(len(proxy.internalProcessors))
	for _, processor := range proxy.internalProcessors {
		go func(p messagebroker.Processor) {
			defer wg.Done()
			if err := p.Process(ctx, message); err != nil {
				errsChan <- err
			}
		}(processor)
	}
	wg.Wait()
	close(errsChan)
	var errs *multierror.Error
	for err := range errsChan {
		errs = multierror.Append(errs, err)
	}

	return errors.Wrapf(errs.ErrorOrNil(), "Failed processing message %v in parallel", string(message.Value))
}

func (proxy *proxyProcessor) processSequential(ctx context.Context, message *messagebroker.Message) error {
	var errs *multierror.Error
	for _, processor := range proxy.internalProcessors {
		if err := processor.Process(ctx, message); err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "proxyProcessor: failed to processSequential %v message on %T", string(message.Value), processor))
		}
	}

	return errors.Wrapf(errs.ErrorOrNil(), "Failed processing message %v in sequential", string(message.Value))
}

func (p *processor) CompleteTask(ctx context.Context, task *Task) error {
	return errors.Wrapf(p.tasksRepository.CompleteTask(ctx, task.UserID, task.Name), "unable to complete task for userID:%v", task.UserID)
}

func (p *processor) UnCompleteTask(ctx context.Context, task *Task) error {
	return errors.Wrapf(p.tasksRepository.UnCompleteTask(ctx, task.UserID, task.Name), "unable to uncomplete task for userID:%v", task.UserID)
}
