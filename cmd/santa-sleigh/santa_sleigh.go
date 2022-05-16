// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ice-blockchain/santa/achievements"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
	"github.com/pkg/errors"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	log.Info(fmt.Sprintf("service version %v is starting...", cfg.Version))
	srv := server.New(new(service), applicationYamlKey, "")
	srv.ListenAndServe(ctx, cancel)
}

func (s *service) RegisterRoutes(_ *gin.Engine) {
}

func (s *service) Init(ctx context.Context, cancel context.CancelFunc) {
	s.achievementsProcessor = achievements.StartProcessor(ctx, cancel)
	err := s.setupInitialBadgesData(ctx)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to setup initial state of badges"))
	}
}

func (s *service) Close(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "could not close achievementsProcessor because context ended")
	}

	return errors.Wrap(s.achievementsProcessor.Close(), "could not close achievementsProcessor")
}

func (s *service) CheckHealth(ctx context.Context, r *server.RequestCheckHealth) server.Response {
	log.Debug("checking health...", "package", "achievements")

	if err := s.achievementsProcessor.CheckHealth(ctx); err != nil {
		return server.Unexpected(err)
	}

	return server.OK(r)
}
