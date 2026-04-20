package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	// Embed Go's IANA timezone database so time.LoadLocation works on
	// minimal images (alpine, distroless) that don't ship /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/glycoview/glycoview/internal/api"
	"github.com/glycoview/glycoview/internal/auth"
	"github.com/glycoview/glycoview/internal/config"
	"github.com/glycoview/glycoview/internal/dashboardauth"
	"github.com/glycoview/glycoview/internal/goals"
	"github.com/glycoview/glycoview/internal/store"
	"github.com/glycoview/glycoview/internal/store/memory"
	postgresstore "github.com/glycoview/glycoview/internal/store/postgres"
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
	var goalsStore goals.Store
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

		pgGoals, err := goals.NewPostgresStore(context.Background(), pgStore.Pool())
		if err != nil {
			log.Fatalf("goals postgres init: %v", err)
		}
		goalsStore = pgGoals
	} else {
		dataStore = memory.New()
		accountStore = dashboardauth.NewMemoryStore()
		goalsStore = goals.NewMemoryStore()
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

	goalsService := &goals.Service{
		Store:   goalsStore,
		Samples: goals.NightscoutStoreSource{Store: dataStore},
	}
	handler := api.New(cfg, dataStore, authManager, dashboardAuth, goalsService)
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, handler))
}

func ensureUIBuilt(cfg config.Config) error {
	distIndex := filepath.Join("frontend", "dist", "index.html")
	frontendPackage := filepath.Join("frontend", "package.json")
	frontendModules := filepath.Join("frontend", "node_modules")

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

	if _, err := os.Stat(frontendModules); err != nil {
		log.Printf("installing frontend dependencies with npm --prefix frontend ci")
		cmd := exec.Command("npm", "--prefix", "frontend", "ci")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if runErr := cmd.Run(); runErr != nil {
			return runErr
		}
	}

	log.Printf("building web UI into frontend/dist")
	cmd := exec.Command("npm", "run", "build:ui")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
