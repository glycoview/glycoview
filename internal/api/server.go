package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/better-monitoring/glycoview/internal/auth"
	"github.com/better-monitoring/glycoview/internal/config"
	"github.com/better-monitoring/glycoview/internal/dashboardauth"
	"github.com/better-monitoring/glycoview/internal/httpx"
	v1 "github.com/better-monitoring/glycoview/internal/nightscout/v1"
	v3 "github.com/better-monitoring/glycoview/internal/nightscout/v3"
	"github.com/better-monitoring/glycoview/internal/store"
	"github.com/better-monitoring/glycoview/internal/ui"
)

type Server struct {
	Config  config.Config
	Store   store.Store
	Auth    *auth.Manager
	AppAuth *dashboardauth.Service
}

func New(cfg config.Config, dataStore store.Store, authManager *auth.Manager, appAuth *dashboardauth.Service) http.Handler {
	server := &Server{
		Config:  cfg,
		Store:   dataStore,
		Auth:    authManager,
		AppAuth: appAuth,
	}
	return server.routes()
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Mount("/api", v1.NewRouter(v1.Dependencies{
		Config: s.Config,
		Store:  s.Store,
		Auth:   s.Auth,
	}))
	r.Mount("/api/v1", v1.NewRouter(v1.Dependencies{
		Config: s.Config,
		Store:  s.Store,
		Auth:   s.Auth,
	}))
	r.Mount("/api/v3", v3.NewRouter(v3.Dependencies{
		Config: s.Config,
		Store:  s.Store,
		Auth:   s.Auth,
	}))

	r.Get("/api/v2/authorization/request/{accessToken}", func(w http.ResponseWriter, r *http.Request) {
		token, err := s.Auth.IssueJWT(chi.URLParam(r, "accessToken"))
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Unauthorized"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"token": token})
	})

	r.Mount("/", ui.NewRouter(ui.Dependencies{
		Config:  s.Config,
		Store:   s.Store,
		Auth:    s.Auth,
		AppAuth: s.AppAuth,
	}))

	return r
}
