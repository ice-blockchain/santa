// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/ice-blockchain/santa/achievements"
	"github.com/ice-blockchain/wintr/server"
)

// Public API.

const (
	badgeTypeSocial = "SOCIAL"
	badgeTypeIce    = "ICE"
	badgeTypeLevel  = "LEVEL"
)

type (
	RequestGetUserAchievements struct {
		AuthenticatedUser server.AuthenticatedUser `json:"authenticatedUser" swaggerignore:"true"`
		UserID            string                   `uri:"userId" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
		// Collectibles are a subset of achievements, you can include any of [`TASKS`,`BADGES`].
		IncludeCollectibles []string `form:"includeCollectibles"`
	}
	RequestGetUserBadges struct {
		AuthenticatedUser server.AuthenticatedUser `json:"authenticatedUser" swaggerignore:"true"`
		UserID            string                   `uri:"userId" example:"did:ethr:0x4B73C58370AEfcEf86A6021afCDe5673511376B2"`
		BadgeType         string                   `form:"badgeType" example:"SOCIAL"`
	}
)

// Private API.

const applicationYamlKey = "cmd/santa"

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	// | service implements server.State and is responsible for managing the state and lifecycle of the package.
	service struct {
		achievementsRepository achievements.Repository
	}
	config struct {
		Host    string `yaml:"host"`
		Version string `yaml:"version"`
	}
)
