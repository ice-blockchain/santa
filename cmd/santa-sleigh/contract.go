// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"github.com/ice-blockchain/santa/achievements"
)

// Private API.

const applicationYamlKey = "cmd/santa-sleigh"

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	// | service implements server.State and is responsible for managing the state and lifecycle of the package.
	service struct {
		achievementsProcessor achievements.Processor
	}
	config struct {
		Host          string `yaml:"host"`
		Version       string `yaml:"version"`
		InitialBadges []struct {
			Name                  string `yaml:"name"`
			Type                  string `yaml:"type"`
			LeftProgressInterval  uint64 `yaml:"leftProgressInterval"`
			RightProgressInterval uint64 `yaml:"rightProgressInterval"`
		} `yaml:"initialBadges"`
	}
)
