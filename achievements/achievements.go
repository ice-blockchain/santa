// SPDX-License-Identifier: BUSL-1.1

package achievements

import (
	"context"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/connectors/storage"
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
	return errors.Wrap(r.db.Close(), "closing achievements repository failed")
}

func StartProcessor(ctx context.Context, cancel context.CancelFunc) Processor {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	db := storage.MustConnect(ctx, cancel, ddl, applicationYamlKey)

	return &processor{
		close: func() error {
			return errors.Wrap(db.Close(), "failed to close db in processor")
		},
		WriteBadgesRepository: &repository{
			db: db,
		},
	}
}

func (p *processor) Close() error {
	return errors.Wrap(p.close(), "error closing achievemn")
}

func (p *processor) CheckHealth(ctx context.Context) error {
	// TODO implement me
	return nil
}
