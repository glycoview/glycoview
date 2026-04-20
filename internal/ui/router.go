package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glycoview/nightscout-api/httpx"
	"github.com/go-chi/chi/v5"

	"github.com/glycoview/glycoview/internal/appliance"
	"github.com/glycoview/glycoview/internal/auth"
	"github.com/glycoview/glycoview/internal/config"
	"github.com/glycoview/glycoview/internal/dashboardauth"
	"github.com/glycoview/glycoview/internal/store"
)

type Dependencies struct {
	Config  config.Config
	Store   store.Store
	Auth    *auth.Manager
	AppAuth *dashboardauth.Service
}

func NewRouter(dep Dependencies) http.Handler {
	service := Service{
		Config: dep.Config,
		Store:  dep.Store,
	}
	agent := newAgentClient(dep.Config.AgentURL, dep.Config.AgentToken)

	r := chi.NewRouter()
	r.Route("/app/api", func(r chi.Router) {
		r.Get("/auth/status", func(w http.ResponseWriter, r *http.Request) {
			hasUsers, err := dep.AppAuth.SetupStatus(r.Context())
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
				return
			}
			token := dashboardauth.SessionTokenFromRequest(r)
			body := map[string]any{
				"setupRequired": !hasUsers,
				"authenticated": false,
				"appVersion":    dep.Config.AppVersion,
			}
			if token != "" {
				user, err := dep.AppAuth.CurrentUser(r.Context(), token)
				if err == nil {
					body["authenticated"] = true
					body["user"] = user
				}
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		})
		r.Post("/auth/setup", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Username    string `json:"username"`
				Password    string `json:"password"`
				DisplayName string `json:"displayName"`
			}
			if err := httpx.ReadJSON(r, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			user, token, apiSecret, err := dep.AppAuth.Bootstrap(r.Context(), body.Username, body.Password, body.DisplayName, dep.Config.APISecret)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			dep.Auth.UpdateAPISecret(apiSecret)
			dep.AppAuth.SetSessionCookie(w, r, token)
			httpx.WriteJSON(w, http.StatusCreated, map[string]any{"user": user, "apiSecret": apiSecret})
		})
		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := httpx.ReadJSON(r, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			user, token, err := dep.AppAuth.Login(r.Context(), body.Username, body.Password)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			dep.AppAuth.SetSessionCookie(w, r, token)
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": user})
		})
		r.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			_ = dep.AppAuth.Logout(r.Context(), dashboardauth.SessionTokenFromRequest(r))
			dep.AppAuth.ClearSessionCookie(w, r)
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		})
		r.Get("/auth/install-secret", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			apiSecret, err := dep.AppAuth.CurrentInstallAPISecret(r.Context())
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"apiSecret": apiSecret})
		}))
		r.Get("/users", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			users, err := dep.AppAuth.ListUsers(r.Context())
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"users": users})
		}))
		r.Post("/users", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body struct {
				Username    string `json:"username"`
				Password    string `json:"password"`
				DisplayName string `json:"displayName"`
				Role        string `json:"role"`
			}
			if err := httpx.ReadJSON(r, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			user, err := dep.AppAuth.CreateUser(r.Context(), body.Username, body.Password, body.DisplayName, body.Role)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			httpx.WriteJSON(w, http.StatusCreated, map[string]any{"user": user})
		}))
		r.Patch("/users/{id}", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body struct {
				DisplayName string `json:"displayName"`
				Password    string `json:"password"`
				Role        string `json:"role"`
				Active      *bool  `json:"active"`
			}
			if err := httpx.ReadJSON(r, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			user, err := dep.AppAuth.UpdateUser(r.Context(), chi.URLParam(r, "id"), body.DisplayName, body.Role, body.Active, body.Password)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": user})
		}))
		r.Get("/settings/status", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.StatusResponse
			if err := agent.get(r.Context(), "/v1/system/status", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/settings/updates/check", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.UpdateCheckResponse
			if err := agent.get(r.Context(), "/v1/update/check", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Post("/settings/updates/apply", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var payload appliance.ApplyUpdateRequest
			if err := httpx.ReadJSON(r, &payload); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			var body appliance.ActionResponse
			if err := agent.post(r.Context(), "/v1/update/apply", payload, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Post("/settings/updates/rollback", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.ActionResponse
			if err := agent.post(r.Context(), "/v1/update/rollback", map[string]any{}, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/settings/tls/providers", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body struct {
				Providers  []appliance.TLSProvider    `json:"providers"`
				Challenges []appliance.ChallengeOption `json:"challenges,omitempty"`
			}
			if err := agent.get(r.Context(), "/v1/tls/providers", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/settings/tls/config", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.TLSConfig
			if err := agent.get(r.Context(), "/v1/tls/config", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/settings/dyndns/providers", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body struct {
				Providers []appliance.DynamicDNSProvider `json:"providers"`
			}
			if err := agent.get(r.Context(), "/v1/dyndns/providers", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/settings/dyndns/config", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.DynamicDNSConfig
			if err := agent.get(r.Context(), "/v1/dyndns/config", &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Post("/settings/dyndns/configure", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var payload appliance.DynamicDNSConfig
			if err := httpx.ReadJSON(r, &payload); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			var body appliance.ActionResponse
			if err := agent.post(r.Context(), "/v1/dyndns/configure", payload, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Post("/settings/dyndns/sync", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var body appliance.ActionResponse
			if err := agent.post(r.Context(), "/v1/dyndns/sync", map[string]any{}, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Post("/settings/tls/configure", requireSessionRole(dep.AppAuth, dashboardauth.RoleAdmin, func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
			var payload appliance.TLSConfig
			if err := httpx.ReadJSON(r, &payload); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
				return
			}
			var body appliance.ActionResponse
			if err := agent.post(r.Context(), "/v1/tls/configure", payload, &body); err != nil {
				httpx.WriteJSON(w, http.StatusBadGateway, map[string]any{"status": 502, "message": err.Error()})
				return
			}
			httpx.WriteJSON(w, http.StatusOK, body)
		}))
		r.Get("/overview", authApp(dep.Auth, dep.AppAuth, func(w http.ResponseWriter, r *http.Request) {
			days := queryInt(r, "days", 14)
			body, err := service.Overview(r.Context(), time.Now().UTC(), days)
			writeAppJSON(w, body, err)
		}))
		r.Get("/daily", authApp(dep.Auth, dep.AppAuth, func(w http.ResponseWriter, r *http.Request) {
			day := parseDateParam(r.URL.Query().Get("date"), time.Now().UTC())
			body, err := service.Daily(r.Context(), day)
			writeAppJSON(w, body, err)
		}))
		r.Get("/trends", authApp(dep.Auth, dep.AppAuth, func(w http.ResponseWriter, r *http.Request) {
			days := queryInt(r, "days", 14)
			body, err := service.Trends(r.Context(), time.Now().UTC(), days)
			writeAppJSON(w, body, err)
		}))
		r.Get("/profile", authApp(dep.Auth, dep.AppAuth, func(w http.ResponseWriter, r *http.Request) {
			body, err := service.Profile(r.Context())
			writeAppJSON(w, body, err)
		}))
		r.Get("/devices", authApp(dep.Auth, dep.AppAuth, func(w http.ResponseWriter, r *http.Request) {
			body, err := service.Devices(r.Context(), time.Now().UTC())
			writeAppJSON(w, body, err)
		}))
	})

	r.Mount("/", newAssetsHandler())
	return r
}

func authApp(authManager *auth.Manager, appAuth *dashboardauth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if appAuth != nil {
			sessionToken := dashboardauth.SessionTokenFromRequest(r)
			if sessionToken != "" {
				user, err := appAuth.CurrentUser(r.Context(), sessionToken)
				if err != nil {
					httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Unauthorized"})
					return
				}
				if user.Role == dashboardauth.RoleAdmin || user.Role == dashboardauth.RoleDoctor {
					next(w, r)
					return
				}
				httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": "Role not permitted"})
				return
			}
		}
		identity, err := authManager.AuthenticateRequest(r)
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Unauthorized"})
			return
		}
		if !authManager.HasPermission(*identity, "api:entries:read") {
			httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": "Missing permission api:entries:read"})
			return
		}
		next(w, r.WithContext(r.Context()))
	}
}

func requireSessionRole(appAuth *dashboardauth.Service, role string, next func(http.ResponseWriter, *http.Request, dashboardauth.UserSummary)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := appAuth.RequireRole(r.Context(), dashboardauth.SessionTokenFromRequest(r), role)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		next(w, r, user)
	}
}


func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dashboardauth.ErrSetupAlreadyDone):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{"status": 409, "message": "Setup has already been completed"})
	case errors.Is(err, dashboardauth.ErrInvalidCredentials), errors.Is(err, dashboardauth.ErrSessionExpired):
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": 401, "message": "Invalid username or password"})
	case errors.Is(err, dashboardauth.ErrRoleNotAllowed):
		httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": 403, "message": "Role not permitted"})
	default:
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": err.Error()})
	}
}

func writeAppJSON(w http.ResponseWriter, body any, err error) {
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": 500, "message": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseDateParam(value string, fallback time.Time) time.Time {
	if value == "" {
		return fallback
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return fallback
	}
	return parsed
}

func isAssetPath(path string) bool {
	return strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/favicon") ||
		strings.HasPrefix(path, "/icons")
}
