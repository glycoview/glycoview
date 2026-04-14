package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentClientIncludesTokenHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-GlycoView-Agent-Token"); got != "agent-secret" {
			t.Fatalf("token header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := newAgentClient(server.URL, "agent-secret")
	var body map[string]any
	if err := client.get(context.Background(), "/v1/system/status", &body); err != nil {
		t.Fatal(err)
	}
}
