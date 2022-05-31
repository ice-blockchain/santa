// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"context"
	"encoding/json"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/time"
	"github.com/pkg/errors"
)

func NewRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{db: db, mb: mb}
}

func (r *repository) upsertCurrentUserRole(ctx context.Context, userID users.UserID, roleName RoleName) error {
	cur := &CurrentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: time.Now(),
	}

	updateOp := []tarantool.Op{
		{Op: "=", Field: fieldUpdatedAtSeq, Arg: cur.UpdatedAt},
		{Op: "=", Field: fieldRoleNameSeq, Arg: roleName},
	}

	if err := r.db.UpsertAsync("CURRENT_USER_ROLES", cur, updateOp).GetTyped(&[]CurrentUserRole{}); err != nil {
		return errors.Wrapf(err, "error upserting current user role for userID:%v", userID)
	}

	return errors.Wrapf(r.sendCurrentUserRole(ctx, cur),
		"failed to send updated user role to message broker: %v for userID:%v", roleName, userID)
}

func (r *repository) sendCurrentUserRole(ctx context.Context, cur *CurrentUserRole) error {
	b, err := json.Marshal(cur)
	if err != nil {
		return errors.Wrapf(err, "[achieve-roles] failed to marshal %#v", cur)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     cur.UserID,
		Topic:   cfg.MessageBroker.Topics[5].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-roles] failed to send message to broker")
}
