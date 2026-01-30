package devicestatus

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
	"github.com/uptrace/bun"
)

type IRepository interface {
	Insert(items []entities.DeviceStatus) error
	Find(ctx context.Context, spec *common.QuerySpec) ([]entities.DeviceStatus, error)
	Delete(ctx context.Context, spec *common.QuerySpec) error
}

type Repository struct {
	db       *bun.DB
	fieldMap map[string]string
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db, fieldMap: map[string]string{
		"_id":        "id",
		"date":       "date",
		"device":     "device",
		"created_at": "created_at",
	}}
}

func (r *Repository) Insert(items []entities.DeviceStatus) error {
	_, err := r.db.NewInsert().Model(&items).Exec(context.Background())
	return err
}

func (r *Repository) Find(ctx context.Context, spec *common.QuerySpec) ([]entities.DeviceStatus, error) {
	q := r.db.NewSelect().Model((*entities.DeviceStatus)(nil))

	if spec.Limit > 0 {
		q = q.Limit(spec.Limit)
	}
	if spec.Offset > 0 {
		q = q.Offset(spec.Offset)
	}

	var out []entities.DeviceStatus
	err := q.Scan(ctx, &out)
	return out, err
}

func (r *Repository) Delete(ctx context.Context, spec *common.QuerySpec) error {
	q := r.db.NewDelete().Model((*entities.DeviceStatus)(nil))
	_, err := q.Exec(ctx)
	return err
}
