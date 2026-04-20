package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/glycoview/nightscout-api/httpx"
	"github.com/go-chi/chi/v5"

	"github.com/glycoview/glycoview/internal/dashboardauth"
	"github.com/glycoview/glycoview/internal/goals"
)

// mountGoalsRoutes attaches /app/api/goals/* to r. Any authenticated user can
// list and view progress; create/edit/delete also require an authenticated
// session (we don't restrict to admin here — a patient should be able to set
// and track their own goals alongside the clinician).
func (s *Server) mountGoalsRoutes(r chi.Router) {
	if s.Goals == nil {
		return
	}

	user := func(next func(http.ResponseWriter, *http.Request, dashboardauth.UserSummary)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u, err := s.AppAuth.CurrentUser(r.Context(), dashboardauth.SessionTokenFromRequest(r))
			if err != nil {
				writeAuthErrorJSON(w, err)
				return
			}
			next(w, r, u)
		}
	}

	r.Get("/app/api/goals", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		tz := parseTimeZoneOrUTC(r.URL.Query().Get("tz"))
		includeArchived := r.URL.Query().Get("includeArchived") == "true"
		list, err := s.Goals.List(r.Context(), includeArchived, tz)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"goals": list})
	}))

	r.Post("/app/api/goals", user(func(w http.ResponseWriter, r *http.Request, u dashboardauth.UserSummary) {
		var g goals.Goal
		if err := httpx.ReadJSON(r, &g); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		g.CreatedBy = u.ID
		if strings.TrimSpace(g.StartDate) == "" {
			g.StartDate = time.Now().UTC().Format("2006-01-02")
		}
		wp, err := s.Goals.Create(r.Context(), g)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, wp)
	}))

	r.Get("/app/api/goals/{id}", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		tz := parseTimeZoneOrUTC(r.URL.Query().Get("tz"))
		wp, err := s.Goals.Get(r.Context(), chi.URLParam(r, "id"), tz)
		if err != nil {
			if errors.Is(err, goals.ErrNotFound) {
				httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": 404, "message": "not found"})
				return
			}
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, wp)
	}))

	r.Put("/app/api/goals/{id}", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		var g goals.Goal
		if err := httpx.ReadJSON(r, &g); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		g.ID = chi.URLParam(r, "id")
		wp, err := s.Goals.Update(r.Context(), g)
		if err != nil {
			if errors.Is(err, goals.ErrNotFound) {
				httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": 404, "message": "not found"})
				return
			}
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, wp)
	}))

	r.Post("/app/api/goals/{id}/status", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		var body struct {
			Status string `json:"status"`
		}
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		wp, err := s.Goals.SetStatus(r.Context(), chi.URLParam(r, "id"), goals.Status(body.Status))
		if err != nil {
			if errors.Is(err, goals.ErrNotFound) {
				httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": 404, "message": "not found"})
				return
			}
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, wp)
	}))

	r.Delete("/app/api/goals/{id}", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		if err := s.Goals.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
			if errors.Is(err, goals.ErrNotFound) {
				httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": 404, "message": "not found"})
				return
			}
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}))

	r.Post("/app/api/goals/preview", user(func(w http.ResponseWriter, r *http.Request, _ dashboardauth.UserSummary) {
		var body struct {
			Predicate  goals.Predicate `json:"predicate"`
			TargetDate string          `json:"targetDate,omitempty"`
			TZ         string          `json:"tz,omitempty"`
		}
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		tz := parseTimeZoneOrUTC(body.TZ)
		progress, err := s.Goals.Preview(r.Context(), body.Predicate, body.TargetDate, tz)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": 500, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, progress)
	}))
}

// parseTimeZoneOrUTC mirrors the ui-router helper but local to api so we don't
// pull an import cycle.
func parseTimeZoneOrUTC(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" || name == "UTC" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.UTC
}

var _ = json.Marshal // keep json imported for future use
