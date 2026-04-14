package appliance

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runnerFunc func(ctx context.Context, env map[string]string, name string, args ...string) (string, error)

func (fn runnerFunc) Run(ctx context.Context, env map[string]string, name string, args ...string) (string, error) {
	return fn(ctx, env, name, args...)
}

func TestSaveStateEncryptsSecretsAtRest(t *testing.T) {
	dir := t.TempDir()
	service := NewService(Config{
		StateDir:      dir,
		EncryptionKey: "test-agent-secret",
	})

	state := State{
		TLS: TLSConfig{
			Provider: "cloudflare",
			Env: map[string]string{
				"CF_DNS_API_TOKEN": "top-secret-token",
			},
		},
	}
	if err := service.saveState(state); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("top-secret-token")) {
		t.Fatal("state file contains plaintext secret")
	}

	loaded, err := service.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TLS.Env["CF_DNS_API_TOKEN"] != "top-secret-token" {
		t.Fatalf("loaded secret = %q", loaded.TLS.Env["CF_DNS_API_TOKEN"])
	}
}

func TestConfigureTLSPreservesStoredSecretsAndRedactsResponses(t *testing.T) {
	dir := t.TempDir()
	stackFile := filepath.Join(dir, "stack.yml")
	if err := os.WriteFile(stackFile, []byte("version: '3.9'\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService(Config{
		StateDir:      dir,
		StackFile:     stackFile,
		StackEnvFile:  filepath.Join(dir, ".env"),
		OverrideFile:  filepath.Join(dir, "override.yml"),
		EncryptionKey: "test-agent-secret",
		Runner: runnerFunc(func(ctx context.Context, env map[string]string, name string, args ...string) (string, error) {
			return "ok", nil
		}),
	})

	if _, err := service.ConfigureTLS(context.Background(), TLSConfig{
		Domain:        "glycoview.example.com",
		Email:         "admin@example.com",
		ChallengeType: "dns-01",
		Provider:      "cloudflare",
		Env: map[string]string{
			"CF_DNS_API_TOKEN": "top-secret-token",
		},
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := service.TLSConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Env["CF_DNS_API_TOKEN"]; ok {
		t.Fatal("TLSConfig exposed redacted secret")
	}

	if _, err := service.ConfigureTLS(context.Background(), TLSConfig{
		Domain:        "glycoview.example.com",
		Email:         "ops@example.com",
		ChallengeType: "dns-01",
		Provider:      "cloudflare",
		Env:           map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := service.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if stored.TLS.Env["CF_DNS_API_TOKEN"] != "top-secret-token" {
		t.Fatalf("stored secret = %q", stored.TLS.Env["CF_DNS_API_TOKEN"])
	}
	if stored.TLS.Email != "ops@example.com" {
		t.Fatalf("stored email = %q", stored.TLS.Email)
	}
}

func TestConfigureDynamicDNSPreservesSecretsAndRedactsResponses(t *testing.T) {
	dir := t.TempDir()
	service := NewService(Config{
		StateDir:      dir,
		EncryptionKey: "test-agent-secret",
	})

	if _, err := service.ConfigureDynamicDNS(context.Background(), DynamicDNSConfig{
		Enabled:    true,
		Provider:   "cloudflare",
		Zone:       "example.com",
		RecordName: "home.example.com",
		Env: map[string]string{
			"CF_DNS_API_TOKEN": "top-secret-token",
		},
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := service.DynamicDNSConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Env["CF_DNS_API_TOKEN"]; ok {
		t.Fatal("DynamicDNSConfig exposed redacted secret")
	}

	if _, err := service.ConfigureDynamicDNS(context.Background(), DynamicDNSConfig{
		Enabled:    true,
		Provider:   "cloudflare",
		Zone:       "example.com",
		RecordName: "home.example.com",
		Env:        map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := service.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if stored.DynamicDNS.Env["CF_DNS_API_TOKEN"] != "top-secret-token" {
		t.Fatalf("stored secret = %q", stored.DynamicDNS.Env["CF_DNS_API_TOKEN"])
	}
}

func TestSyncDynamicDNSUpdatesCloudflareRecord(t *testing.T) {
	dir := t.TempDir()
	var recordUpdated bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/ip":
			_, _ = w.Write([]byte("203.0.113.10"))
		case r.Method == http.MethodGet && r.URL.Path == "/client/v4/zones":
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"zone-1"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/client/v4/zones/zone-1/dns_records":
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"record-1"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/client/v4/zones/zone-1/dns_records/record-1":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"content":"203.0.113.10"`) {
				t.Fatalf("unexpected update payload: %s", string(body))
			}
			recordUpdated = true
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer api.Close()

	service := NewService(Config{
		StateDir:      dir,
		PublicIPURL:   api.URL + "/ip",
		EncryptionKey: "test-agent-secret",
		HTTPClient:    api.Client(),
	})

	if _, err := service.ConfigureDynamicDNS(context.Background(), DynamicDNSConfig{
		Enabled:    true,
		Provider:   "cloudflare",
		Zone:       "example.com",
		RecordName: "home.example.com",
		Env: map[string]string{
			"CF_DNS_API_TOKEN": "top-secret-token",
		},
	}); err != nil {
		t.Fatal(err)
	}

	originalPublicIPURL := service.cfg.PublicIPURL
	service.cfg.PublicIPURL = api.URL + "/ip"
	originalClient := service.client
	service.client = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasPrefix(req.URL.String(), "https://api.cloudflare.com/") {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(api.URL, "http://")
		}
		return originalClient.Do(req)
	})}

	if _, err := service.SyncDynamicDNS(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !recordUpdated {
		t.Fatal("record was not updated")
	}
	stored, err := service.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if stored.DynamicDNS.LastKnownIP != "203.0.113.10" {
		t.Fatalf("last known IP = %q", stored.DynamicDNS.LastKnownIP)
	}
	service.cfg.PublicIPURL = originalPublicIPURL
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
