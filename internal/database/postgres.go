package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/model"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/schema"
)

func createSchemas(db *bun.DB) error {
	models := []any{
		(*model.Entry)(nil),
	}
	ctx := context.Background()
	for _, model := range models {
		_, err := db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func ConnectDB(cfg config.Config) (*bun.DB, error) {
	var sqldb *sql.DB
	var dialect schema.Dialect

	switch cfg.Driver {
	case "sqlite":
		sqldb, err := sql.Open(sqliteshim.ShimName, fmt.Sprintf("file:%s?cache=shared&mode=rwc", cfg.DBSQLitePath))
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite: %w", err)
		}
		if err := sqldb.Ping(); err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite at %s: %w", cfg.DBSQLitePath, err)
		}
		dialect = sqlitedialect.New()
	case "postgres":
		sqldb = sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithDSN(cfg.DBConnection),
		))
		dialect = pgdialect.New()
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db := bun.NewDB(sqldb, dialect)
	if err := createSchemas(db); err != nil {
		return nil, err
	}
	return db, nil
}
