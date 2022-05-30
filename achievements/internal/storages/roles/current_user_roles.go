// SPDX-License-Identifier: BUSL-1.1

package roles

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/pkg/errors"
)

func NewRepository(db tarantool.Connector, mb messagebroker.Client) repository {
	return repository{db: db, mb: mb}
}

func (r *repository) upsertCurrentUserRole(ctx context.Context, userID users.UserID, roleName RoleName) error {
	updatedAt := time.Now().UTC()
	cur := &currentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: uint64(updatedAt.UnixNano()),
	}

	updateOp := []tarantool.Op{
		{Op: "=", Field: 1, Arg: roleName},
		{Op: "=", Field: 1 + 1, Arg: updatedAt},
	}

	if err := r.db.UpsertAsync("CURRENT_USER_ROLES", cur, updateOp).GetTyped(&[]currentUserRole{}); err != nil {
		return errors.Wrapf(err, "error upserting current user role for userID:%v", userID)
	}

	return errors.Wrapf(r.sendCurrentUserRole(ctx, userID, roleName, updatedAt),
		"failed to send updated user role to message broker: %v for userID:%v", roleName, userID)
}

func (r *repository) sendCurrentUserRole(ctx context.Context, userID users.UserID, roleName RoleName, updatedAt time.Time) error {
	m := CurrentUserRole{
		UserID:    userID,
		RoleName:  roleName,
		UpdatedAt: updatedAt,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-roles] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID,
		Topic:   cfg.MessageBroker.Topics[5].Name,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-roles] failed to send message to broker")
}
