package entry

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/oklog/ulid/v2"
	"github.com/uptrace/bun"
)

type IRepository interface {
	InsertEntries(entries []entities.Entry) error
	Find(
		ctx context.Context,
		spec *common.QuerySpec,
	) ([]entities.Entry, error)
	Delete(ctx context.Context, spec *common.QuerySpec) error
}

type Repository struct {
	db       *bun.DB
	fieldMap map[string]string
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{
		db: db,
		fieldMap: map[string]string{
			"_id":         "id",
			"type":        "type",
			"date_string": "date_string",
			"date":        "date",
			"sgv":         "sgv",
			"direction":   "direction",
			"noise":       "noise",
			"filtered":    "filtered",
			"unfiltered":  "unfiltered",
			"rssi":        "rssi",
		}}
}

func (r *Repository) InsertEntries(entries []entities.Entry) error {
	for i := range entries {
		entries[i].ID = ulid.Make().String()
	}
	_, err := r.db.NewInsert().Model(&entries).Exec(context.Background())
	return err
}

func (r *Repository) Find(
	ctx context.Context,
	spec *common.QuerySpec,
) ([]entities.Entry, error) {

	q := r.db.NewSelect().Model((*entities.Entry)(nil))

	for _, f := range spec.Filters {
		col := r.fieldMap[f.Field]
		q = q.Where("? "+string(f.Op)+" ?", bun.Ident(col), f.Value)
	}

	if spec.Limit > 0 {
		q = q.Limit(spec.Limit)
	}
	if spec.Offset > 0 {
		q = q.Offset(spec.Offset)
	}

	var entries []entities.Entry
	err := q.Scan(ctx, &entries)
	return entries, err
}

func (r *Repository) Delete(ctx context.Context, spec *common.QuerySpec) error {
	q := r.db.NewDelete().Model((*entities.Entry)(nil))

	for _, f := range spec.Filters {
		col := r.fieldMap[f.Field]
		q = q.Where("? "+string(f.Op)+" ?", bun.Ident(col), f.Value)
	}

	_, err := q.Exec(ctx)
	return err
}
