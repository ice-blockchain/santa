// SPDX-License-Identifier: ice License 1.0

package main

import (
	"github.com/ice-blockchain/santa/badges"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/santa/tasks"
)

// Public API.

type (
	GetTasksArg struct {
		UserID string `uri:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4" swaggerignore:"true" required:"true"`
	}
	GetLevelsAndRolesSummaryArg struct {
		UserID string `uri:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4" allowForbiddenGet:"true" swaggerignore:"true" required:"true"`
	}
	GetBadgeSummaryArg struct {
		UserID string `uri:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4" allowForbiddenGet:"true" swaggerignore:"true" required:"true"`
	}
	GetBadgesArg struct {
		UserID    string           `uri:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4" allowForbiddenGet:"true" swaggerignore:"true" required:"true"`
		GroupType badges.GroupType `uri:"badgeType" example:"social" swaggerignore:"true" required:"true" enums:"level,coin,social"`
	}
)

// Private API.

const (
	applicationYamlKey = "cmd/santa"
	swaggerRoot        = "/achievements/r"
)

// Values for server.ErrorResponse#Code.
const (
	badgesHiddenErrorCode = "BADGES_HIDDEN"
)

type (
	// | service implements server.State and is responsible for managing the state and lifecycle of the package.
	service struct {
		tasksRepository          tasks.Repository
		levelsAndRolesRepository levelsandroles.Repository
		badgesRepository         badges.Repository
	}
	config struct {
		Host    string `yaml:"host"`
		Version string `yaml:"version"`
	}
)
