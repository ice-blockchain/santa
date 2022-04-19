// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"github.com/ICE-Blockchain/santa/achievements"
	appCfg "github.com/ICE-Blockchain/wintr/config"
	"github.com/ICE-Blockchain/wintr/log"
	"github.com/ICE-Blockchain/wintr/server"
	"github.com/gin-gonic/gin"
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
