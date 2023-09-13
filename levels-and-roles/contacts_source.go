// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
)

func (c *agendaContactsSource) Process(ctx context.Context, msg *messagebroker.Message) error { //nolint:funlen // Not worth to break.
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "unexpected deadline while processing message")
	}
	if len(msg.Value) == 0 {
		return nil
	}
	contact := new(users.Contact)
	if err := json.UnmarshalContext(ctx, msg.Value, contact); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v into %#v", string(msg.Value), contact)
	}
	before, err := c.getProgress(ctx, contact.UserID, true)
	if err != nil && !storage.IsErr(err, storage.ErrNotFound) {
		return errors.Wrapf(err, "can't get contacts for userID:%v", contact.UserID)
	}
	if before != nil && before.AgendaContactUserIDs != nil {
		for _, id := range *before.AgendaContactUserIDs {
			if id == contact.ContactUserID {
				return ErrDuplicate
			}
		}
	}
	toUpsert := make(users.Enum[users.UserID], 0)
	if before.AgendaContactUserIDs != nil {
		toUpsert = append(toUpsert, *before.AgendaContactUserIDs...)
	}
	toUpsert = append(toUpsert, contact.ContactUserID)
	if uErr := c.upsertAgendaContacts(ctx, contact.UserID, &toUpsert); uErr != nil {
		return errors.Wrapf(uErr, "can't upsert agenda contacts for userID:%v", contact.UserID)
	}
	if sErr := c.sendTryCompleteLevelsCommandMessage(ctx, contact.UserID); sErr != nil {
		//nolint:wrapcheck // Not needed.
		return multierror.Append(errors.Wrapf(sErr, "failed to sendTryCompleteLevelsCommandMessage, userID:%v", contact.UserID),
			errors.Wrapf(c.upsertAgendaContacts(ctx, contact.UserID, before.AgendaContactUserIDs),
				"can't rollback agenda contacts joined value for userID:%v", contact.UserID)).ErrorOrNil()
	}

	return nil
}

func (c *agendaContactsSource) upsertAgendaContacts(ctx context.Context, userID string, contacts *users.Enum[users.UserID]) error {
	sql := `INSERT INTO levels_and_roles_progress(user_id, agenda_contact_user_ids) VALUES ($1, $2)
				ON CONFLICT(user_id)
				DO UPDATE
					SET agenda_contact_user_ids = EXCLUDED.agenda_contact_user_ids
					WHERE COALESCE(levels_and_roles_progress.agenda_contact_user_ids, ARRAY[]::TEXT[]) != COALESCE(EXCLUDED.agenda_contact_user_ids, ARRAY[]::TEXT[])`
	_, err := storage.Exec(ctx, c.db, sql, userID, contacts)

	return errors.Wrapf(err, "can't insert/update contact user ids:%#v for userID:%v", contacts, userID)
}
