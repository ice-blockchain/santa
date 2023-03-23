// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/badges"
	"github.com/ice-blockchain/santa/cmd/santa/api"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/santa/tasks"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

// @title						Achievements API
// @version					latest
// @description				API that handles everything related to read-only operations for user's achievements and gamification progress.
// @query.collection.format	multi
// @schemes					https
// @contact.name				ice.io
// @contact.url				https://ice.io
// @BasePath					/v1r
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cfg config
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	api.SwaggerInfo.Host = cfg.Host
	api.SwaggerInfo.Version = cfg.Version
	server.New(new(service), applicationYamlKey, swaggerRoot).ListenAndServe(ctx, cancel)
}

func (s *service) RegisterRoutes(router *server.Router) {
	s.setupTasksRoutes(router)
	s.setupLevelsAndRolesRoutes(router)
	s.setupBadgesRoutes(router)
}

func (s *service) Init(ctx context.Context, cancel context.CancelFunc) {
	s.tasksRepository = tasks.New(ctx, cancel)
	s.levelsAndRolesRepository = levelsandroles.New(ctx, cancel)
	s.badgesRepository = badges.New(ctx, cancel)
}

func (s *service) Close(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "could not close service because context ended")
	}

	return errors.Wrap(multierror.Append(
		errors.Wrap(s.badgesRepository.Close(), "could not close badges repository"),
		errors.Wrap(s.levelsAndRolesRepository.Close(), "could not close levels-and-roles repository"),
		errors.Wrap(s.tasksRepository.Close(), "could not close tasks repository"),
	).ErrorOrNil(), "could not close repositories")
}

func (s *service) CheckHealth(ctx context.Context) error {
	log.Debug("checking health...", "package", "badges")
	if _, err := s.badgesRepository.GetBadges(ctx, badges.CoinGroupType, "bogus"); err != nil && !errors.Is(err, tasks.ErrRelationNotFound) {
		return errors.Wrap(err, "get badges failed")
	}
	log.Debug("checking health...", "package", "tasks")
	if _, err := s.tasksRepository.GetTasks(ctx, "bogus"); err != nil && !errors.Is(err, tasks.ErrRelationNotFound) {
		return errors.Wrap(err, "get tasks failed")
	}
	log.Debug("checking health...", "package", "levels-and-roles")
	if _, err := s.levelsAndRolesRepository.GetSummary(ctx, "bogus"); err != nil {
		return errors.Wrap(err, "levels-and-roles.GetSummary failed")
	}

	return nil
}
