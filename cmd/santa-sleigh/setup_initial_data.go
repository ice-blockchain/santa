package main

import (
	"context"
	"github.com/ice-blockchain/santa/achievements"
	"github.com/pkg/errors"
)

func (s *service) setupInitialBadgesData(ctx context.Context) error {
	for _, badge := range cfg.InitialBadges {
		err := s.achievementsProcessor.AddBadge(ctx, &achievements.Badge{
			Name: badge.Name,
			Type: badge.Type,
			ProgressInterval: struct {
				Left  uint64 `json:"left" example:"11"`
				Right uint64 `json:"right" example:"22"`
			}{badge.LeftProgressInterval, badge.RightProgressInterval},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to add badge %#v", badge)
		}
	}
	return nil
}
