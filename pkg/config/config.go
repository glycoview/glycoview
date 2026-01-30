package config

import (
	"errors"
	"os"
)

const DEFAULT_DRIVER = "sqlite"

// const DEFAULT_SQLITE_PATH = "/etc/bscout/db.sqlite"
const DEFAULT_SQLITE_PATH = "./dummy/test.db"
const DEFAULT_PORT = "8080"

type Config struct {
	Driver       string
	DBConnection string
	DBSQLitePath string
	PORT         string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Driver:       os.Getenv("DB_DRIVER"),
		DBConnection: os.Getenv("DB_POSTGRES_URI"),
		DBSQLitePath: os.Getenv("DB_SQLITE_PATH"),
		PORT:         os.Getenv("PORT"),
	}
	missing_env := ""
	if cfg.Driver == "" {
		cfg.Driver = DEFAULT_DRIVER
	}
	if cfg.Driver != "postgres" && cfg.Driver != "sqlite" {
		missing_env += "DB_DRIVER must be 'postgres' or 'sqlite'; "
	}
	if cfg.Driver == "postgres" && cfg.DBConnection == "" {
		missing_env += "DB_POSTGRES_URI is not set; "
	}
	if cfg.Driver == "sqlite" && cfg.DBSQLitePath == "" {
		cfg.DBSQLitePath = DEFAULT_SQLITE_PATH
	}
	if cfg.PORT == "" {
		cfg.PORT = DEFAULT_PORT
	}
	if missing_env != "" {
		return nil, errors.New(missing_env)
	}
	return cfg, nil
}
