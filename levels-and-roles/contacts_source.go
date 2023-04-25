// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"
	"strings"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
)

func (c *contactsTableSource) Process(ctx context.Context, msg *messagebroker.Message) error {
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

	return errors.Wrapf(storage.DoInTransaction(ctx, c.db, func(conn storage.QueryExecer) error {
		contactsCount, err := c.upsertContacts(ctx, contact)
		if err != nil {
			return errors.Wrapf(err, "can't upsert contacts for userID:%v", msg.Key)
		}
		if err = c.increaseAgendaContactsJoinedValue(ctx, contact.UserID, contactsCount); err != nil {
			return errors.Wrapf(err, "can't update agenda contacts joined for userID:%v", msg.Key)
		}

		return errors.Wrapf(c.sendTryCompleteLevelsCommandMessage(ctx, msg.Key), "failed to sendTryCompleteLevelsCommandMessage, userID:%v", msg.Key)
	}), "transaction rollback")
}

func (c *contactsTableSource) upsertContacts(ctx context.Context, contact *users.Contact) (int, error) {
	before, err := c.getContacts(ctx, contact.UserID)
	if err != nil && !storage.IsErr(err, storage.ErrNotFound) {
		return 0, errors.Wrapf(err, "can't get contacts for userID:%v", contact.UserID)
	}
	var toUpsert []string
	if before != nil {
		ids := strings.Split(before.ContactUserIDs, ",")
		for _, id := range ids {
			if id == contact.ContactUserID {
				return 0, ErrDuplicate
			}
		}
		toUpsert = append(toUpsert, ids...)
	}
	toUpsert = append(toUpsert, contact.ContactUserID)
	sql := `INSERT INTO contacts(user_id, contact_user_ids) VALUES ($1, $2)
				ON CONFLICT(user_id)
				DO UPDATE
					SET contact_user_ids = EXCLUDED.contact_user_ids
				WHERE contacts.contact_user_ids != EXCLUDED.contact_user_ids`
	_, err = storage.Exec(ctx, c.db, sql, contact.UserID, strings.Join(toUpsert, ","))

	return len(toUpsert), errors.Wrapf(err, "can't insert/update contact user ids:%#v for userID:%v", toUpsert, contact.UserID)
}

func (c *contactsTableSource) getContacts(ctx context.Context, userID string) (*contacts, error) {
	sql := `SELECT * FROM contacts WHERE user_id = $1`
	res, err := storage.Get[contacts](ctx, c.db, sql, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get contact user ids for userID:%v", userID)
	}

	return res, nil
}

func (c *contactsTableSource) increaseAgendaContactsJoinedValue(ctx context.Context, userID string, contactsCount int) error {
	sql := `UPDATE levels_and_roles_progress
				SET agenda_contacts_joined = $2
				WHERE user_id = $1`
	_, err := storage.Exec(ctx, c.db, sql, userID, contactsCount)

	return errors.Wrapf(err, "can't update levels_and_roles_progress by 1 for userID:%v", userID)
}
