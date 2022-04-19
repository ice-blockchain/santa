// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"github.com/ICE-Blockchain/santa/achievements"
	"github.com/ICE-Blockchain/santa/cmd/santa/api"
	appCfg "github.com/ICE-Blockchain/wintr/config"
	"github.com/ICE-Blockchain/wintr/log"
	"github.com/ICE-Blockchain/wintr/server"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

//nolint:godot // Because those are comments parsed by swagger
// @title                    User Achievements API
// @version                  latest
// @description              API that handles everything related to read only operations for user's achievements(badges, levels, roles, task completions, etc).
// @query.collection.format  multi
// @schemes                  https
// @contact.name             ICE
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

func (s *service) RegisterRoutes(engine *gin.Engine) {
	s.setupAchievementRoutes(engine)
}

func (s *service) Init(ctx context.Context, cancel context.CancelFunc) {
	s.achievementsRepository = achievements.New(ctx, cancel)
}

func (s *service) Close(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "could not close repository because context ended")
	}

	return errors.Wrap(s.achievementsRepository.Close(), "could not close repository")
}

func (s *service) CheckHealth(ctx context.Context, r *server.RequestCheckHealth) server.Response {
	log.Debug("checking health...", "package", "achievements")

	// TODO implement me

	return server.OK(r)
}
