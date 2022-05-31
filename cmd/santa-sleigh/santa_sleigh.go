// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/achievements"
	"github.com/ice-blockchain/santa/cmd/santa-sleigh/api"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

//nolint:godot // Because those are comments parsed by swagger
// @title                    User Achievements API
// @version                  latest
// @description              API that handles everything related to write operations for user's achievements(badges, levels, roles, task completions, etc).
// @query.collection.format  multi
// @schemes                  https
// @contact.name             ice.io
// @contact.url              https://ice.io
// @BasePath                 /v1
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	api.SwaggerInfo.Host = cfg.Host
	api.SwaggerInfo.Version = cfg.Version
	srv := server.New(new(service), applicationYamlKey, "/achievements")
	srv.ListenAndServe(ctx, cancel)
}

func (s *service) RegisterRoutes(router *gin.Engine) {
	s.setupAchievementRoutes(router)
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
