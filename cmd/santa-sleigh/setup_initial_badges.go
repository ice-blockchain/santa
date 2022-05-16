package main

import (
	"context"
	"github.com/ice-blockchain/santa/achievements"
	"github.com/pkg/errors"
)

func (s *service) setupInitialBadgesData(ctx context.Context) error {
	// TODO may be move initial setup to DDL or to config?
	badges := []*achievements.Badge{
		// region social badges
		&achievements.Badge{
			Name: "Ice breaker",
			Type: "SOCIAL",
			ProgressInterval: struct {
				Left  uint64 `json:"left" example:"11"`
				Right uint64 `json:"right" example:"22"`
			}{0, 3},
		},
		&achievements.Badge{
			Name: "Trouble maker",
			Type: "SOCIAL",
			ProgressInterval: struct {
				Left  uint64 `json:"left" example:"11"`
				Right uint64 `json:"right" example:"22"`
			}{4, 9},
		},
		// endregion social badges

	}
	for _, badge := range badges {
		err := s.achievementsProcessor.AddBadge(ctx, badge)
		if err != nil {
			return errors.Wrapf(err, "failed to add badge %#v", badge)
		}
	}
	return nil
}
