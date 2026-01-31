package treatment

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/oklog/ulid/v2"
	"github.com/uptrace/bun"
)

type IRepository interface {
	Insert(entries []entities.Treatment) error
	Find(
		ctx context.Context,
		spec *common.QuerySpec,
	) ([]entities.Treatment, error)
	GetOne(ctx context.Context, spec *common.QuerySpec) (*entities.Treatment, error)
	Delete(ctx context.Context, spec *common.QuerySpec) error
}

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Insert(entries []entities.Treatment) error {
	for i := range entries {
		entries[i].ID = ulid.Make().String()
	}
	_, err := r.db.NewInsert().Model(&entries).Exec(context.Background())
	return err
}

func (r *Repository) GetOne(ctx context.Context, spec *common.QuerySpec) (*entities.Treatment, error) {
	q := r.db.NewSelect().Model((*entities.Treatment)(nil))
	for _, f := range spec.Filters {
		q = q.Where("? "+string(f.Op)+" ?", bun.Ident(f.Field), f.Value)
	}
	var entries []entities.Treatment
	err := q.Limit(1).Scan(ctx, &entries)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return &entries[0], nil
}

func (r *Repository) Find(
	ctx context.Context,
	spec *common.QuerySpec,
) ([]entities.Treatment, error) {

	q := r.db.NewSelect().Model((*entities.Treatment)(nil))

	for _, f := range spec.Filters {
		q = q.Where("? "+string(f.Op)+" ?", bun.Ident(f.Field), f.Value)
	}

	if spec.Limit > 0 {
		q = q.Limit(spec.Limit)
	}
	if spec.Offset > 0 {
		q = q.Offset(spec.Offset)
	}

	var entries []entities.Treatment
	err := q.Scan(ctx, &entries)
	return entries, err
}

func (r *Repository) Delete(ctx context.Context, spec *common.QuerySpec) error {
	q := r.db.NewDelete().Model((*entities.Treatment)(nil))

	for _, f := range spec.Filters {
		q = q.Where("? "+string(f.Op)+" ?", bun.Ident(f.Field), f.Value)
	}

	_, err := q.Exec(ctx)
	return err
}
