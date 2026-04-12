package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/better-monitoring/bscout/internal/auth"
	"github.com/better-monitoring/bscout/internal/store"
)

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func ReadJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	return decoder.Decode(dst)
}

func RequireRecord(storeErr error) (status int, body map[string]any) {
	switch {
	case errors.Is(storeErr, store.ErrGone):
		return http.StatusGone, map[string]any{"status": http.StatusGone}
	case errors.Is(storeErr, store.ErrNotFound):
		return http.StatusNotFound, map[string]any{"status": http.StatusNotFound}
	default:
		return http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": storeErr.Error()}
	}
}

func ParseIfModifiedSince(r *http.Request) (time.Time, bool) {
	value := strings.TrimSpace(r.Header.Get("If-Modified-Since"))
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(http.TimeFormat, value)
	return parsed, err == nil
}

func ParseIfUnmodifiedSince(r *http.Request) (time.Time, bool) {
	value := strings.TrimSpace(r.Header.Get("If-Unmodified-Since"))
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(http.TimeFormat, value)
	return parsed, err == nil
}

func LastModifiedHeader(w http.ResponseWriter, millis int64) {
	if millis <= 0 {
		return
	}
	w.Header().Set("Last-Modified", time.UnixMilli(millis).UTC().Format(http.TimeFormat))
}

func ParsePositiveInt(value string, fallback int) int {
	if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
		return parsed
	}
	return fallback
}

func SubjectName(ctx context.Context) string {
	if identity := auth.IdentityFromContext(ctx); identity != nil {
		return identity.Name
	}
	return "anonymous"
}
