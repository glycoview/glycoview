package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/better-monitoring/glycoview/internal/appliance"
	"github.com/better-monitoring/glycoview/internal/httpx"
)

func main() {
	addr := envOrDefault("GLYCOVIEW_AGENT_ADDR", ":8090")
	service := appliance.NewService(appliance.Config{
		StateDir:     envOrDefault("GLYCOVIEW_AGENT_STATE_DIR", "/var/lib/glycoview-agent"),
		StackName:    envOrDefault("GLYCOVIEW_STACK_NAME", "glycoview"),
		StackFile:    envOrDefault("GLYCOVIEW_STACK_FILE", "/opt/glycoview/stack/stack.yml"),
		StackEnvFile: envOrDefault("GLYCOVIEW_STACK_ENV_FILE", "/opt/glycoview/stack/.env"),
		OverrideFile: envOrDefault("GLYCOVIEW_TRAEFIK_OVERRIDE_FILE", "/var/lib/glycoview-agent/traefik.override.yml"),
		ReleasesRepo: envOrDefault("GLYCOVIEW_RELEASES_REPO", "glycoview/glycoview"),
		AppImage:     envOrDefault("GLYCOVIEW_IMAGE_REPO", "ghcr.io/glycoview/glycoview"),
		AgentImage:   envOrDefault("GLYCOVIEW_AGENT_IMAGE_REPO", "ghcr.io/glycoview/glycoview-agent"),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/v1/system/status", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		body, err := service.Status(r.Context())
		writeResponse(w, body, err)
	}))
	mux.HandleFunc("/v1/update/check", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		body, err := service.CheckUpdate(r.Context())
		writeResponse(w, body, err)
	}))
	mux.HandleFunc("/v1/update/apply", methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body appliance.ApplyUpdateRequest
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		result, err := service.ApplyUpdate(r.Context(), body)
		writeResponse(w, result, err)
	}))
	mux.HandleFunc("/v1/update/rollback", methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Rollback(r.Context())
		writeResponse(w, result, err)
	}))
	mux.HandleFunc("/v1/tls/providers", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"providers": service.Providers()})
	}))
	mux.HandleFunc("/v1/tls/config", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		body, err := service.TLSConfig(r.Context())
		writeResponse(w, body, err)
	}))
	mux.HandleFunc("/v1/tls/configure", methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body appliance.TLSConfig
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		result, err := service.ConfigureTLS(r.Context(), body)
		writeResponse(w, result, err)
	}))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("glycoview-agent listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func methodHandler(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			httpx.WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"status": http.StatusMethodNotAllowed, "message": "Method not allowed"})
			return
		}
		next(w, r)
	}
}

func writeResponse(w http.ResponseWriter, body any, err error) {
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, body)
		return
	}
	status := http.StatusBadRequest
	if errors.Is(err, os.ErrNotExist) {
		status = http.StatusNotFound
	}
	httpx.WriteJSON(w, status, map[string]any{"status": status, "message": err.Error()})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
