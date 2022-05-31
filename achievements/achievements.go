// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"

	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
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

	return &processor{
		close: func() error {
			return errors.Wrap(db.Close(), "failed to close db in processor")
		},
		WriteRepository: &repository{
			db: db,
		},
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

func (r *repository) CompleteTask(ctx context.Context, task *Task) error {
	//TODO implement me
	panic("implement me")
}

func (r *repository) UnCompleteTask(ctx context.Context, task *Task) error {
	//TODO implement me
	panic("implement me")
}
