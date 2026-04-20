package api

import (
	"net/http"

	nsv1 "github.com/glycoview/nightscout-api/api/v1"
	nsv3 "github.com/glycoview/nightscout-api/api/v3"
	nsconfig "github.com/glycoview/nightscout-api/config"
	nsdeps "github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/glycoview/glycoview/internal/auth"
	"github.com/glycoview/glycoview/internal/config"
	"github.com/glycoview/glycoview/internal/dashboardauth"
	"github.com/glycoview/glycoview/internal/store"
	"github.com/glycoview/glycoview/internal/ui"
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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	nsdep := nsdeps.Dependencies{
		Config: nsconfig.Config{
			APISecret:    s.Config.APISecret,
			JWTSecret:    s.Config.JWTSecret,
			Enable:       append([]string(nil), s.Config.Enable...),
			DefaultRoles: append([]string(nil), s.Config.DefaultRoles...),
			API3MaxLimit: s.Config.API3MaxLimit,
			AppVersion:   s.Config.AppVersion,
		}.WithDefaults(),
		Store: s.Store,
		Auth:  s.Auth,
	}

	r.Mount("/api", nsv1.NewNightscoutV1Router(nsdep))
	r.Mount("/api/v1", nsv1.NewNightscoutV1Router(nsdep))
	r.Mount("/api/v3", nsv3.NewNightscoutV3Router(nsdep))

	s.mountAIRoutes(r)

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
