package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type response map[string]any

func main() {
	addr := os.Getenv("BSCOUT_AGENT_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, response{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/v1/system/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, response{
			"status":        "ok",
			"service":       "bscout-agent",
			"dockerManaged": true,
			"updates": response{
				"supported": false,
				"message":   "update orchestration is not implemented yet",
			},
			"tls": response{
				"supported": false,
				"message":   "TLS provider orchestration is not implemented yet",
			},
		})
	})
	mux.HandleFunc("/v1/update/check", notImplemented("update checking is not implemented yet"))
	mux.HandleFunc("/v1/update/apply", notImplemented("update orchestration is not implemented yet"))
	mux.HandleFunc("/v1/update/rollback", notImplemented("rollback orchestration is not implemented yet"))
	mux.HandleFunc("/v1/tls/providers", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, response{
			"providers": []string{
				"cloudflare",
				"route53",
				"hetzner",
				"digitalocean",
				"ovh",
				"gandi",
				"gcloud",
			},
			"mode": "planned",
		})
	})
	mux.HandleFunc("/v1/tls/configure", notImplemented("TLS orchestration is not implemented yet"))

	log.Printf("bscout-agent listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func notImplemented(message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, response{
			"status":  http.StatusNotImplemented,
			"message": message,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
