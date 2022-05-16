package achievements

import (
	"context"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func (r *repository) AddBadge(ctx context.Context, b *Badge) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "add user failed because context failed")
	}

	sql := `INSERT INTO badges (NAME,  TYPE,        FROM_INCLUSIVE, TO_INCLUSIVE)
                       VALUES (:name, :badge_type, :from_inclusive, :to_inclusive)`

	params := map[string]interface{}{
		"name":           b.Name,
		"badge_type":     b.Type,
		"from_inclusive": b.ProgressInterval.Left,
		"to_inclusive":   b.ProgressInterval.Right,
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to add badge %#v", b)
	}
	return nil // TODO: send badge to message broker, will any service consume them?

}
