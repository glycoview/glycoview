package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAgentTokenProtectsControlEndpoints(t *testing.T) {
	handler := withAgentToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "agent-secret")

	req := httptest.NewRequest(http.MethodGet, "/v1/system/status", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/system/status", nil)
	req.Header.Set("X-GlycoView-Agent-Token", "agent-secret")
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("healthz status = %d", resp.Code)
	}
}
