// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/go-tarantool-client"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

func (r *repository) GetSummary(ctx context.Context, userID string) (*Summary, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	if res, err := r.getProgress(ctx, userID); err != nil && !errors.Is(err, storage.ErrRelationNotFound) {
		return nil, errors.Wrapf(err, "failed to getProgress for userID:%v", userID)
	} else { //nolint:revive // .
		return newSummary(res, requestingUserID(ctx)), nil
	}
}

func (r *repository) getProgress(ctx context.Context, userID string) (res *progress, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "unexpected deadline")
	}
	res = new(progress)
	err = errors.Wrapf(r.db.GetTyped("LEVELS_AND_ROLES_PROGRESS", "pk_unnamed_LEVELS_AND_ROLES_PROGRESS_1", tarantool.StringKey{S: userID}, res),
		"failed to get LEVELS_AND_ROLES_PROGRESS for userID:%v", userID)
	if res.UserID == "" {
		return nil, storage.ErrRelationNotFound
	}

	return
}

func newSummary(pr *progress, requestingUserID string) *Summary {
	var level uint64
	if pr == nil || !pr.HideLevel || requestingUserID == pr.UserID {
		level = pr.level()
	}
	var roles []*Role
	if pr == nil || !pr.HideRole || requestingUserID == pr.UserID {
		roles = pr.roles()
	}

	return &Summary{Roles: roles, Level: level}
}

func (p *progress) level() uint64 {
	if p == nil || p.CompletedLevels == nil || len(*p.CompletedLevels) == 0 {
		return 1
	} else { //nolint:revive // .
		return uint64(len(*p.CompletedLevels))
	}
}

func (p *progress) roles() []*Role {
	if p == nil || p.EnabledRoles == nil || len(*p.EnabledRoles) == 0 {
		return []*Role{
			{
				Type:    SnowmanRoleType,
				Enabled: true,
			},
			{
				Type:    AmbassadorRoleType,
				Enabled: false,
			},
		}
	} else { //nolint:revive // .
		return []*Role{
			{
				Type:    SnowmanRoleType,
				Enabled: false,
			},
			{
				Type:    AmbassadorRoleType,
				Enabled: true,
			},
		}
	}
}
