package app

import (
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/database"
	"github.com/better-monitoring/bscout/internal/repository"
	"github.com/better-monitoring/bscout/internal/service"
)

type Dependencies struct {
	EntryService *service.EntryService
}

func BuildDependencies(cfg *config.Config) Dependencies {
	db, err := database.ConnectDB(cfg)
	if err != nil {
		panic(err)
	}
	entryRepo := repository.NewEntryRepository(db)
	entryService := service.NewEntryService(entryRepo)

	return Dependencies{
		EntryService: entryService,
	}
}
