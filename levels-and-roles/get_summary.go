// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"
	storagev2 "github.com/ice-blockchain/wintr/connectors/storage/v2"

	"github.com/pkg/errors"

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
	res, err = storagev2.Get[progress](ctx, r.dbV2, `SELECT 
        COALESCE(enabled_roles,'') AS enabled_roles,
		COALESCE(completed_levels,'') AS completed_levels,
		user_id,
		COALESCE(phone_number_hash,'') AS phone_number_hash,
		COALESCE(mining_streak,0) AS mining_streak,
		COALESCE(pings_sent,0) AS pings_sent,
		COALESCE(agenda_contacts_joined,0) AS agenda_contacts_joined,
		COALESCE(friends_invited,0) AS friends_invited,
		COALESCE(completed_tasks,0) completed_tasks,
		COALESCE(hide_level,false) AS hide_level,
		COALESCE(hide_role,false) AS hide_role
    FROM levels_and_roles_progress WHERE user_id = $1`, userID)
	if errors.Is(err, storagev2.ErrNotFound) {
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
