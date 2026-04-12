package testutil

import (
	"net/http"
	"net/http/httptest"

	"github.com/better-monitoring/bscout/internal/api"
	"github.com/better-monitoring/bscout/internal/auth"
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/store/memory"
)

type Harness struct {
	Config *config.Config
	Store  *memory.Store
	Auth   *auth.Manager
	Server *httptest.Server
}

func NewHarness(defaultRoles ...string) *Harness {
	cfg := config.Config{
		APISecret:    "this is my long pass phrase",
		JWTSecret:    "this is my long pass phrase",
		Enable:       []string{"careportal", "rawbg", "api"},
		DefaultRoles: defaultRoles,
		API3MaxLimit: 1000,
	}
	authManager := auth.New(cfg.APISecret, cfg.DefaultRoles, cfg.JWTSecret)
	store := memory.New()
	server := httptest.NewServer(api.New(cfg, store, authManager))
	return &Harness{
		Config: &cfg,
		Store:  store,
		Auth:   authManager,
		Server: server,
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
