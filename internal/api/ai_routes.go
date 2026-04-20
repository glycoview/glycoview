package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/glycoview/nightscout-api/httpx"
	"github.com/go-chi/chi/v5"

	"github.com/glycoview/glycoview/internal/ai"
	"github.com/glycoview/glycoview/internal/dashboardauth"
	"github.com/glycoview/glycoview/internal/ui"
)

// mountAIRoutes attaches /app/api/ai/* to r. Kept in the api package rather
// than in ui so that ai can depend on ui without an import cycle.
func (s *Server) mountAIRoutes(r chi.Router) {
	uiService := ui.Service{Config: s.Config, Store: s.Store}
	svc := ai.NewService(ai.Deps{UI: uiService}, s.AppAuth)

	adminOnly := func(next func(http.ResponseWriter, *http.Request, dashboardauth.UserSummary)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, err := s.AppAuth.RequireRole(r.Context(), dashboardauth.SessionTokenFromRequest(r), dashboardauth.RoleAdmin)
			if err != nil {
				writeAuthErrorJSON(w, err)
				return
			}
			next(w, r, user)
		}
	}
	anyUser := func(next func(http.ResponseWriter, *http.Request, dashboardauth.UserSummary)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, err := s.AppAuth.CurrentUser(r.Context(), dashboardauth.SessionTokenFromRequest(r))
			if err != nil {
				writeAuthErrorJSON(w, err)
				return
			}
			next(w, r, user)
		}
	}

	r.Get("/app/api/ai/settings", adminOnly(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		loaded, err := ai.Load(r.Context(), s.AppAuth)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ai.Redact(loaded))
	}))
	r.Put("/app/api/ai/settings", adminOnly(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		var body ai.Settings
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		// Empty or masked API key on PUT means "keep existing" so the UI can
		// PUT the redacted object back without losing the saved key.
		if strings.TrimSpace(body.APIKey) == "" || strings.Contains(body.APIKey, "•") {
			existing, err := ai.Load(r.Context(), s.AppAuth)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
				return
			}
			body.APIKey = existing.APIKey
		}
		if err := ai.Save(r.Context(), s.AppAuth, body); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		saved, _ := ai.Load(r.Context(), s.AppAuth)
		httpx.WriteJSON(w, http.StatusOK, ai.Redact(saved))
	}))
	r.Post("/app/api/ai/chat", anyUser(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		var req ai.ChatRequest
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		streamAIChat(w, r, svc, req)
	}))
}

func streamAIChat(w http.ResponseWriter, r *http.Request, svc *ai.Service, req ai.ChatRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	emit := func(event string, data any) {
		payload, err := json.Marshal(data)
		if err != nil {
			payload = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
		flusher.Flush()
	}

	if err := svc.RunChat(r.Context(), req, emit); err != nil {
		emit(ai.EventError, map[string]any{"message": err.Error()})
	}
}

func writeAuthErrorJSON(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dashboardauth.ErrInvalidCredentials), errors.Is(err, dashboardauth.ErrSessionExpired):
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": 401, "message": "Unauthorized"})
	case errors.Is(err, dashboardauth.ErrRoleNotAllowed):
		httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": 403, "message": "Role not permitted"})
	default:
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": err.Error()})
	}
}
