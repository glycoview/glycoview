package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/glycoview/glycoview/internal/appliance"
	"github.com/glycoview/nightscout-api/httpx"
)

func main() {
	addr := envOrDefault("GLYCOVIEW_AGENT_ADDR", ":8090")
	token := strings.TrimSpace(os.Getenv("GLYCOVIEW_AGENT_TOKEN"))
	stateKey := strings.TrimSpace(os.Getenv("GLYCOVIEW_AGENT_STATE_KEY"))
	if stateKey == "" {
		stateKey = token
	}
	service := appliance.NewService(appliance.Config{
		StateDir:      envOrDefault("GLYCOVIEW_AGENT_STATE_DIR", "/var/lib/glycoview-agent"),
		StackName:     envOrDefault("GLYCOVIEW_STACK_NAME", "glycoview"),
		StackFile:     envOrDefault("GLYCOVIEW_STACK_FILE", "/opt/glycoview/stack/stack.yml"),
		StackEnvFile:  envOrDefault("GLYCOVIEW_STACK_ENV_FILE", "/opt/glycoview/stack/.env"),
		OverrideFile:  envOrDefault("GLYCOVIEW_TRAEFIK_OVERRIDE_FILE", "/var/lib/glycoview-agent/traefik.override.yml"),
		ReleasesRepo:  envOrDefault("GLYCOVIEW_RELEASES_REPO", "glycoview/glycoview"),
		AppImage:      envOrDefault("GLYCOVIEW_IMAGE_REPO", "ghcr.io/glycoview/glycoview"),
		AgentImage:    envOrDefault("GLYCOVIEW_AGENT_IMAGE_REPO", "ghcr.io/glycoview/glycoview-agent"),
		EncryptionKey: stateKey,
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
	mux.HandleFunc("/v1/dyndns/providers", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"providers": service.DynamicDNSProviders()})
	}))
	mux.HandleFunc("/v1/dyndns/config", methodHandler(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		body, err := service.DynamicDNSConfig(r.Context())
		writeResponse(w, body, err)
	}))
	mux.HandleFunc("/v1/dyndns/configure", methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		var body appliance.DynamicDNSConfig
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "message": "Invalid JSON body"})
			return
		}
		result, err := service.ConfigureDynamicDNS(r.Context(), body)
		writeResponse(w, result, err)
	}))
	mux.HandleFunc("/v1/dyndns/sync", methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		result, err := service.SyncDynamicDNS(r.Context())
		writeResponse(w, result, err)
	}))

	server := &http.Server{
		Addr:              addr,
		Handler:           withAgentToken(mux, token),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go startDynamicDNSSyncLoop(service)

	log.Printf("glycoview-agent listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func startDynamicDNSSyncLoop(service *appliance.Service) {
	ctx := context.Background()
	run := func() {
		if _, err := service.SyncDynamicDNS(ctx); err != nil {
			log.Printf("dynamic DNS sync failed: %v", err)
		}
	}
	run()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		run()
	}
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

func withAgentToken(next http.Handler, token string) http.Handler {
	token = strings.TrimSpace(token)
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-GlycoView-Agent-Token") != token {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
