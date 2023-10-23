// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/badges"
	"github.com/ice-blockchain/santa/cmd/santa-sleigh/api"
	friendsinvited "github.com/ice-blockchain/santa/friends-invited"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/santa/tasks"
	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/server"
)

// @title						Achievements API
// @version					latest
// @description				API that handles everything related to write-only operations for user's achievements and gamification progress.
// @query.collection.format	multi
// @schemes					https
// @contact.name				ice.io
// @contact.url				https://ice.io
// @BasePath					/v1w
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cfg config
	appcfg.MustLoadFromKey(applicationYamlKey, &cfg)
	api.SwaggerInfo.Host = cfg.Host
	api.SwaggerInfo.Version = cfg.Version
	server.New(new(service), applicationYamlKey, swaggerRoot).ListenAndServe(ctx, cancel)
}

func (s *service) RegisterRoutes(router *server.Router) {
	s.setupTasksRoutes(router)
}

func (s *service) Init(ctx context.Context, cancel context.CancelFunc) {
	s.tasksProcessor = tasks.StartProcessor(ctx, cancel)
	s.levelsAndRolesProcessor = levelsandroles.StartProcessor(ctx, cancel)
	s.badgesProcessor = badges.StartProcessor(ctx, cancel)
	s.friendsProcessor = friendsinvited.StartProcessor(ctx, cancel)
}

func (s *service) Close(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "could not close service because context ended")
	}

	return errors.Wrap(multierror.Append(
		errors.Wrap(s.badgesProcessor.Close(), "could not close badges processor"),
		errors.Wrap(s.levelsAndRolesProcessor.Close(), "could not close levels-and-roles processor"),
		errors.Wrap(s.tasksProcessor.Close(), "could not close tasks processor"),
		errors.Wrap(s.friendsProcessor.Close(), "could not close friends-invited processor"),
	).ErrorOrNil(), "could not close processors")
}

func (s *service) CheckHealth(ctx context.Context) error {
	log.Debug("checking health...", "package", "tasks")
	if err := s.tasksProcessor.CheckHealth(ctx); err != nil {
		return errors.Wrap(err, "tasks processor health check failed")
	}
	log.Debug("checking health...", "package", "levels-and-roles")
	if err := s.levelsAndRolesProcessor.CheckHealth(ctx); err != nil {
		return errors.Wrap(err, "levels-and-roles processor health check failed")
	}
	log.Debug("checking health...", "package", "badges")
	if err := s.badgesProcessor.CheckHealth(ctx); err != nil {
		return errors.Wrap(err, "badges processor health check failed")
	}
	log.Debug("checking health...", "package", "friends-invited")
	if err := s.friendsProcessor.CheckHealth(ctx); err != nil {
		return errors.Wrap(err, "friends-invited processor health check failed")
	}

	return nil
}
