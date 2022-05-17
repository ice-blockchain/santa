// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"

	"github.com/ice-blockchain/santa/achievements"
	"github.com/pkg/errors"
)

var initialBadges = []*achievements.Badge{ //nolint:gochecknoglobals // Temporary variable to store initial data.
	{
		Name: "Ice breaker",
		Type: "SOCIAL",
		ProgressInterval: struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		}{0, 3},
	},
	{
		Name: "Trouble maker",
		Type: "SOCIAL",
		ProgressInterval: struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		}{4, 9},
	},
	{
		Name: "Snowy plower",
		Type: "SOCIAL",
		ProgressInterval: struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		}{10, 24},
	},
}

func (s *service) setupInitialBadgesData(ctx context.Context) error {
	//nolint:nolintlint,godox // TODO may be move initial setup to DDL or to config, it seems to be a temporary method.
	for _, badge := range initialBadges {
		err := s.achievementsProcessor.AddBadge(ctx, badge)
		if err != nil {
			return errors.Wrapf(err, "failed to add badge %#v", badge)
		}
	}

	return nil
}
