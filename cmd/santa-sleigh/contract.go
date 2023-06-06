// SPDX-License-Identifier: ice License 1.0

package main

import (
	"github.com/ice-blockchain/santa/badges"
	friendsInvited "github.com/ice-blockchain/santa/friends-invited"
	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/santa/tasks"
)

// Public API.

type (
	CompleteTaskRequestBody struct {
		Data     *tasks.Data `json:"data,omitempty"`
		UserID   string      `uri:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4" swaggerignore:"true" required:"true"`
		TaskType tasks.Type  `uri:"taskType" example:"start_mining" swaggerignore:"true" required:"true" enums:"claim_username,start_mining,upload_profile_picture,follow_us_on_twitter,join_telegram,invite_friends"` //nolint:lll // .
	}
)

// Private API.

const (
	applicationYamlKey = "cmd/santa-sleigh"
	swaggerRoot        = "/achievements/w"
)

// Values for server.ErrorResponse#Code.
const (
	userNotFoundErrorCode      = "USER_NOT_FOUND"
	invalidPropertiesErrorCode = "INVALID_PROPERTIES"
)

type (
	// | service implements server.State and is responsible for managing the state and lifecycle of the package.
	service struct {
		tasksProcessor          tasks.Processor
		levelsAndRolesProcessor levelsandroles.Processor
		badgesProcessor         badges.Processor
		friendsProcessor        friendsInvited.Processor
	}
	config struct {
		Host    string `yaml:"host"`
		Version string `yaml:"version"`
	}
)
