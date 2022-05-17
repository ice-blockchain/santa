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
	return nil
}

func (b *badge) Badge() *Badge {
	return &Badge{
		Name: b.Name,
		Type: b.BadgeType,
		ProgressInterval: struct {
			Left  uint64 `json:"left" example:"11"`
			Right uint64 `json:"right" example:"22"`
		}{
			Left:  b.FromInclusive,
			Right: b.ToInclusive,
		},
	}
}
