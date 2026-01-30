package bootstrap

import (
	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/config"
	"github.com/better-monitoring/bscout/pkg/database"
	"github.com/better-monitoring/bscout/pkg/entry"
)

type Dependencies struct {
	EntryService *entry.Service
}

func buildDependencies(cfg *config.Config) Dependencies {
	db, err := database.ConnectDB(cfg)
	if err != nil {
		panic(err)
	}
	entryRepo := entry.NewRepository(db)
	queryHelperService := common.NewQueryHelper()
	entryService := entry.NewService(entryRepo, queryHelperService)

	return Dependencies{
		EntryService: entryService,
	}
}
