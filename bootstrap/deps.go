package bootstrap

import (
	"github.com/better-monitoring/bscout/pkg/config"
	"github.com/better-monitoring/bscout/pkg/database"
	"github.com/better-monitoring/bscout/pkg/devicestatus"
	"github.com/better-monitoring/bscout/pkg/entry"
	"github.com/better-monitoring/bscout/pkg/food"
	"github.com/better-monitoring/bscout/pkg/profile"
	"github.com/better-monitoring/bscout/pkg/settings"
	"github.com/better-monitoring/bscout/pkg/treatment"
)

type Dependencies struct {
	EntryService        *entry.Service
	DeviceStatusService *devicestatus.Service
	FoodService         *food.Service
	ProfileService      *profile.Service
	SettingsService     *settings.Service
	TreatmentService    *treatment.Service
}

func buildDependencies(cfg *config.Config) Dependencies {
	db, err := database.ConnectDB(cfg)
	if err != nil {
		panic(err)
	}

	entryRepo := entry.NewRepository(db)
	entryService := entry.NewService(entryRepo)

	dsRepo := devicestatus.NewRepository(db)
	dsService := devicestatus.NewService(dsRepo)

	foodRepo := food.NewRepository(db)
	foodService := food.NewService(foodRepo)

	profileRepo := profile.NewRepository(db)
	profileService := profile.NewService(profileRepo)

	settingsRepo := settings.NewRepository(db)
	settingsService := settings.NewService(settingsRepo)

	treatmentRepo := treatment.NewRepository(db)
	treatmentService := treatment.NewService(treatmentRepo)

	return Dependencies{
		EntryService:        entryService,
		DeviceStatusService: dsService,
		FoodService:         foodService,
		ProfileService:      profileService,
		SettingsService:     settingsService,
		TreatmentService:    treatmentService,
	}
}
