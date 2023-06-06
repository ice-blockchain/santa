// SPDX-License-Identifier: ice License 1.0

package friendsinvited

import (
	"context"
	"math/rand"
	stdlibtime "time"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func StartProcessor(ctx context.Context, cancel context.CancelFunc) Processor {
	var cfg config
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	var mbConsumer messagebroker.Client
	prc := &processor{repository: &repository{
		cfg: &cfg,
		db:  storage.MustConnect(ctx, ddl, applicationYamlKey),
		mb:  messagebroker.MustConnect(ctx, applicationYamlKey),
	}}
	//nolint:contextcheck // It's intended. Cuz we want to close everything gracefully.
	mbConsumer = messagebroker.MustConnectAndStartConsuming(context.Background(), cancel, applicationYamlKey,
		&userTableSource{processor: prc},
	)
	prc.shutdown = closeAll(mbConsumer, prc.mb, prc.db)
	go prc.startProcessedReferralsCleaner(ctx)

	return prc
}

func (r *repository) Close() error {
	return errors.Wrap(r.shutdown(), "closing repository failed")
}

func closeAll(mbConsumer, mbProducer messagebroker.Client, db *storage.DB, otherClosers ...func() error) func() error {
	return func() error {
		err1 := errors.Wrap(mbConsumer.Close(), "closing message broker consumer connection failed")
		err2 := errors.Wrap(db.Close(), "closing db connection failed")
		err3 := errors.Wrap(mbProducer.Close(), "closing message broker producer connection failed")
		errs := make([]error, 0, 1+1+1+len(otherClosers))
		errs = append(errs, err1, err2, err3)
		for _, closeOther := range otherClosers {
			if err := closeOther(); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Wrap(multierror.Append(nil, errs...).ErrorOrNil(), "failed to close resources")
	}
}

func (p *processor) CheckHealth(ctx context.Context) error {
	if err := p.db.Ping(ctx); err != nil {
		return errors.Wrap(err, "[health-check] failed to ping DB")
	}
	type ts struct {
		TS *time.Time `json:"ts"`
	}
	now := ts{TS: time.Now()}
	bytes, err := json.MarshalContext(ctx, now)
	if err != nil {
		return errors.Wrapf(err, "[health-check] failed to marshal %#v", now)
	}
	responder := make(chan error, 1)
	p.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     p.cfg.MessageBroker.Topics[0].Name,
		Topic:   p.cfg.MessageBroker.Topics[0].Name,
		Value:   bytes,
	}, responder)

	return errors.Wrapf(<-responder, "[health-check] failed to send health check message to broker")
}

func (p *processor) startProcessedReferralsCleaner(ctx context.Context) {
	ticker := stdlibtime.NewTicker(stdlibtime.Duration(1+rand.Intn(24)) * stdlibtime.Minute) //nolint:gosec,gomnd // Not an  issue.
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			const deadline = 30 * stdlibtime.Second
			reqCtx, cancel := context.WithTimeout(ctx, deadline)
			log.Error(errors.Wrap(p.deleteProcessedReferrals(reqCtx), "failed to deleteOldReferrals"))
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (p *processor) deleteProcessedReferrals(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	sql := `DELETE FROM referrals WHERE processed_at < $1`
	if _, err := storage.Exec(ctx, p.db, sql, time.New(time.Now().Add(-24*stdlibtime.Hour)).Time); err != nil {
		return errors.Wrap(err, "failed to delete old data from referrals")
	}

	return nil
}
