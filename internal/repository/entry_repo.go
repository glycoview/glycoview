package repository

import (
	"context"

	"github.com/better-monitoring/bscout/internal/model"
	"github.com/uptrace/bun"
)

type IEntryRepository interface {
	InsertEntry(entry model.Entry) error
}

type EntryRepository struct {
	db *bun.DB
}

func NewEntryRepository(db *bun.DB) *EntryRepository {
	return &EntryRepository{db: db}
}

func (r *EntryRepository) InsertEntry(entry model.Entry) error {
	_, err := r.db.NewInsert().Model(&entry).Exec(context.Background())
	return err
}
