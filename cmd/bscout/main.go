package main

import (
	"context"
	"log"
	"net/http"

	"github.com/better-monitoring/bscout/internal/api"
	"github.com/better-monitoring/bscout/internal/auth"
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/store"
	"github.com/better-monitoring/bscout/internal/store/memory"
	postgresstore "github.com/better-monitoring/bscout/internal/store/postgres"
)

func main() {
	cfg := config.Load()
	authManager := auth.New(cfg.APISecret, cfg.DefaultRoles, cfg.JWTSecret)

	var dataStore store.Store
	if cfg.DatabaseURL != "" {
		pgStore, err := postgresstore.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("postgres init: %v", err)
		}
		defer pgStore.Close()
		dataStore = pgStore
	} else {
		dataStore = memory.New()
	}

	handler := api.New(cfg, dataStore, authManager)
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, handler))
}
