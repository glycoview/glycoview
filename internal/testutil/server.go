package testutil

import (
	"net/http"
	"net/http/httptest"

	"github.com/better-monitoring/glycoview/internal/api"
	"github.com/better-monitoring/glycoview/internal/auth"
	"github.com/better-monitoring/glycoview/internal/config"
	"github.com/better-monitoring/glycoview/internal/dashboardauth"
	"github.com/better-monitoring/glycoview/internal/store/memory"
)

type Harness struct {
	Config  *config.Config
	Store   *memory.Store
	Auth    *auth.Manager
	AppAuth *dashboardauth.Service
	Server  *httptest.Server
}

func NewHarness(defaultRoles ...string) *Harness {
	cfg := config.Config{
		APISecret:    "this is my long pass phrase",
		JWTSecret:    "this is my long pass phrase",
		Enable:       []string{"careportal", "rawbg", "api"},
		DefaultRoles: defaultRoles,
		API3MaxLimit: 1000,
	}
	return NewHarnessWithConfig(cfg)
}

func NewHarnessWithConfig(cfg config.Config) *Harness {
	authManager := auth.New(cfg.APISecret, cfg.DefaultRoles, cfg.JWTSecret)
	store := memory.New()
	appAuth := dashboardauth.NewService(dashboardauth.NewMemoryStore())
	server := httptest.NewServer(api.New(cfg, store, authManager, appAuth))
	return &Harness{
		Config:  &cfg,
		Store:   store,
		Auth:    authManager,
		AppAuth: appAuth,
		Server:  server,
	}
}

func (h *Harness) Close() {
	if h.Server != nil {
		h.Server.Close()
	}
}

func (h *Harness) Client() *http.Client {
	return h.Server.Client()
}
