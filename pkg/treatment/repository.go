package treatment

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/uptrace/bun"
)

type IRepository interface {
	Insert(items []entities.Treatment) error
	Find(ctx context.Context, spec *common.QuerySpec) ([]entities.Treatment, error)
	Delete(ctx context.Context, spec *common.QuerySpec) error
}

type Repository struct{ db *bun.DB }

func NewRepository(db *bun.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Insert(items []entities.Treatment) error {
	_, err := r.db.NewInsert().Model(&items).Exec(context.Background())
	return err
}

func (r *Repository) Find(ctx context.Context, spec *common.QuerySpec) ([]entities.Treatment, error) {
	q := r.db.NewSelect().Model((*entities.Treatment)(nil))
	var out []entities.Treatment
	err := q.Scan(ctx, &out)
	return out, err
}

func (r *Repository) Delete(ctx context.Context, spec *common.QuerySpec) error {
	q := r.db.NewDelete().Model((*entities.Treatment)(nil))
	_, err := q.Exec(ctx)
	return err
}
