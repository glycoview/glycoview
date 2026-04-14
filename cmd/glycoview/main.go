package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/better-monitoring/glycoview/internal/api"
	"github.com/better-monitoring/glycoview/internal/auth"
	"github.com/better-monitoring/glycoview/internal/config"
	"github.com/better-monitoring/glycoview/internal/dashboardauth"
	"github.com/better-monitoring/glycoview/internal/store"
	"github.com/better-monitoring/glycoview/internal/store/memory"
	postgresstore "github.com/better-monitoring/glycoview/internal/store/postgres"
)

func main() {
	cfg := config.Load()
	if err := ensureUIBuilt(cfg); err != nil {
		log.Fatalf("ui build: %v", err)
	}
	authManager := auth.New(cfg.APISecret, cfg.DefaultRoles, cfg.JWTSecret)

	var dataStore store.Store
	var accountStore dashboardauth.UserStore
	var accountCloser interface{ Close() }
	if cfg.DatabaseURL != "" {
		pgStore, err := postgresstore.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("postgres init: %v", err)
		}
		defer pgStore.Close()
		dataStore = pgStore

		pgAccounts, err := dashboardauth.NewPostgresStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("dashboard auth postgres init: %v", err)
		}
		accountStore = pgAccounts
		accountCloser = pgAccounts
	} else {
		dataStore = memory.New()
		accountStore = dashboardauth.NewMemoryStore()
	}
	if accountCloser != nil {
		defer accountCloser.Close()
	}
	dashboardAuth := dashboardauth.NewService(accountStore)
	hasUsers, err := dashboardAuth.SetupStatus(context.Background())
	if err != nil {
		log.Fatalf("setup status: %v", err)
	}
	storedAPISecret, err := dashboardAuth.CurrentInstallAPISecret(context.Background())
	if err != nil {
		log.Fatalf("install api secret init: %v", err)
	}
	switch {
	case storedAPISecret != "":
		authManager.UpdateAPISecret(storedAPISecret)
	case hasUsers:
		ensuredAPISecret, err := dashboardAuth.EnsureInstallAPISecret(context.Background(), cfg.APISecret)
		if err != nil {
			log.Fatalf("persist install api secret: %v", err)
		}
		authManager.UpdateAPISecret(ensuredAPISecret)
	case cfg.APISecret != "" && cfg.APISecret != "change-me":
		authManager.UpdateAPISecret(cfg.APISecret)
	}

	handler := api.New(cfg, dataStore, authManager, dashboardAuth)
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, handler))
}

func ensureUIBuilt(cfg config.Config) error {
	distIndex := filepath.Join("web", "dist", "index.html")
	frontendPackage := filepath.Join("frontend", "package.json")

	if !cfg.UIBuildOnStart {
		if _, err := os.Stat(distIndex); err == nil {
			return nil
		}
		return nil
	}

	if _, err := os.Stat(frontendPackage); err != nil {
		if _, distErr := os.Stat(distIndex); distErr == nil {
			return nil
		}
		return err
	}

	log.Printf("building web UI with npm --prefix frontend run build")
	cmd := exec.Command("npm", "--prefix", "frontend", "run", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
