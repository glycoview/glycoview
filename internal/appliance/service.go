package appliance

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Config struct {
	StateDir      string
	StackName     string
	StackFile     string
	StackEnvFile  string
	OverrideFile  string
	ReleasesRepo  string
	AppImage      string
	AgentImage    string
	PublicIPURL   string
	EncryptionKey string
	HTTPClient    *http.Client
	Runner        Runner
}

type Runner interface {
	Run(ctx context.Context, env map[string]string, name string, args ...string) (string, error)
}

type shellRunner struct{}

func (shellRunner) Run(ctx context.Context, env map[string]string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	if len(env) > 0 {
		for key, value := range env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

type releasePayload struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type encryptedState struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Service struct {
	cfg    Config
	client *http.Client
	runner Runner
	now    func() time.Time
}

func NewService(cfg Config) *Service {
	if cfg.StateDir == "" {
		cfg.StateDir = "/var/lib/glycoview-agent"
	}
	if cfg.StackName == "" {
		cfg.StackName = "glycoview"
	}
	if cfg.StackFile == "" {
		cfg.StackFile = "/opt/glycoview/stack/stack.yml"
	}
	if cfg.StackEnvFile == "" {
		cfg.StackEnvFile = "/opt/glycoview/stack/.env"
	}
	if cfg.OverrideFile == "" {
		cfg.OverrideFile = filepath.Join(cfg.StateDir, "traefik.override.yml")
	}
	if cfg.ReleasesRepo == "" {
		cfg.ReleasesRepo = "glycoview/glycoview"
	}
	if cfg.AppImage == "" {
		cfg.AppImage = "ghcr.io/glycoview/glycoview"
	}
	if cfg.AgentImage == "" {
		cfg.AgentImage = "ghcr.io/glycoview/glycoview-agent"
	}
	if cfg.PublicIPURL == "" {
		cfg.PublicIPURL = "https://api.ipify.org"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Runner == nil {
		cfg.Runner = shellRunner{}
	}

	return &Service{
		cfg:    cfg,
		client: cfg.HTTPClient,
		runner: cfg.Runner,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Providers() []TLSProvider {
	return []TLSProvider{
		{
			ID:    "cloudflare",
			Label: "Cloudflare",
			Fields: []TLSField{
				{Key: "CF_DNS_API_TOKEN", Label: "API token", Secret: true},
			},
		},
		{
			ID:    "route53",
			Label: "Amazon Route53",
			Fields: []TLSField{
				{Key: "AWS_ACCESS_KEY_ID", Label: "Access key ID"},
				{Key: "AWS_SECRET_ACCESS_KEY", Label: "Secret access key", Secret: true},
				{Key: "AWS_REGION", Label: "Region", Placeholder: "eu-central-1"},
			},
		},
		{
			ID:    "hetzner",
			Label: "Hetzner DNS",
			Fields: []TLSField{
				{Key: "HETZNER_API_KEY", Label: "API key", Secret: true},
			},
		},
		{
			ID:    "digitalocean",
			Label: "DigitalOcean",
			Fields: []TLSField{
				{Key: "DO_AUTH_TOKEN", Label: "API token", Secret: true},
			},
		},
		{
			ID:    "gandi",
			Label: "Gandi v5",
			Fields: []TLSField{
				{Key: "GANDIV5_API_KEY", Label: "API key", Secret: true},
			},
		},
		{
			ID:    "ovh",
			Label: "OVH",
			Fields: []TLSField{
				{Key: "OVH_ENDPOINT", Label: "Endpoint", Placeholder: "ovh-eu"},
				{Key: "OVH_APPLICATION_KEY", Label: "Application key", Secret: true},
				{Key: "OVH_APPLICATION_SECRET", Label: "Application secret", Secret: true},
				{Key: "OVH_CONSUMER_KEY", Label: "Consumer key", Secret: true},
			},
		},
	}
}

func (s *Service) DynamicDNSProviders() []DynamicDNSProvider {
	return []DynamicDNSProvider{
		{
			ID:    "cloudflare",
			Label: "Cloudflare",
			Fields: []TLSField{
				{Key: "CF_DNS_API_TOKEN", Label: "API token", Secret: true},
			},
		},
	}
}

func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	state, err := s.loadState()
	if err != nil {
		return StatusResponse{}, err
	}
	env, err := s.loadEnv()
	if err != nil {
		return StatusResponse{}, err
	}

	currentTag := firstNonEmpty(env["GLYCOVIEW_TAG"], "latest")
	currentAgentTag := firstNonEmpty(env["GLYCOVIEW_AGENT_TAG"], "latest")
	dockerManaged := s.dockerAvailable(ctx)

	return StatusResponse{
		Service:           "glycoview-agent",
		DockerManaged:     dockerManaged,
		StackName:         s.cfg.StackName,
		StackFile:         s.cfg.StackFile,
		StackEnvFile:      s.cfg.StackEnvFile,
		CurrentTag:        currentTag,
		CurrentImage:      s.cfg.AppImage + ":" + currentTag,
		CurrentAgentTag:   currentAgentTag,
		CurrentAgentImage: s.cfg.AgentImage + ":" + currentAgentTag,
		LastAction:        state.Update.LastAction,
		LastMessage:       state.Update.LastMessage,
		LastActionAt:      state.Update.LastActionAt,
		TLS:               s.redactedTLSConfig(state.TLS),
		DynamicDNS:        s.redactedDynamicDNSConfig(state.DynamicDNS),
	}, nil
}

func (s *Service) TLSConfig(_ context.Context) (TLSConfig, error) {
	state, err := s.loadState()
	if err != nil {
		return TLSConfig{}, err
	}
	return s.redactedTLSConfig(state.TLS), nil
}

func (s *Service) DynamicDNSConfig(_ context.Context) (DynamicDNSConfig, error) {
	state, err := s.loadState()
	if err != nil {
		return DynamicDNSConfig{}, err
	}
	return s.redactedDynamicDNSConfig(state.DynamicDNS), nil
}

func (s *Service) ConfigureTLS(ctx context.Context, cfg TLSConfig) (ActionResponse, error) {
	cfg.Domain = strings.TrimSpace(cfg.Domain)
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.ChallengeType = strings.TrimSpace(cfg.ChallengeType)
	if cfg.ChallengeType == "" {
		cfg.ChallengeType = "http-01"
	}
	if cfg.Domain == "" {
		return ActionResponse{}, errors.New("domain is required")
	}
	if cfg.Email == "" {
		return ActionResponse{}, errors.New("email is required")
	}
	if cfg.ChallengeType != "http-01" && cfg.ChallengeType != "dns-01" {
		return ActionResponse{}, errors.New("challengeType must be http-01 or dns-01")
	}

	allowedFields := map[string]struct{}{}
	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
	}
	if cfg.ChallengeType == "dns-01" {
		if cfg.Provider == "" {
			return ActionResponse{}, errors.New("provider is required for dns-01")
		}
		var found bool
		for _, provider := range s.Providers() {
			if provider.ID != cfg.Provider {
				continue
			}
			found = true
			for _, field := range provider.Fields {
				allowedFields[field.Key] = struct{}{}
				value := strings.TrimSpace(cfg.Env[field.Key])
				if value == "" && state.TLS.Provider == cfg.Provider {
					value = strings.TrimSpace(state.TLS.Env[field.Key])
				}
				if value == "" {
					return ActionResponse{}, fmt.Errorf("%s is required", field.Key)
				}
				cfg.Env[field.Key] = value
			}
			break
		}
		if !found {
			return ActionResponse{}, errors.New("unsupported provider")
		}
	}
	env, err := s.loadEnv()
	if err != nil {
		return ActionResponse{}, err
	}

	for key := range state.TLS.Env {
		delete(env, key)
	}
	delete(env, "GLYCOVIEW_ACME_DNS_PROVIDER")

	env["GLYCOVIEW_DOMAIN"] = cfg.Domain
	env["LETSENCRYPT_EMAIL"] = cfg.Email
	if cfg.ChallengeType == "dns-01" {
		env["GLYCOVIEW_ACME_DNS_PROVIDER"] = cfg.Provider
	}

	filteredEnv := map[string]string{}
	for key, value := range cfg.Env {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if len(allowedFields) > 0 {
			if _, ok := allowedFields[key]; !ok {
				continue
			}
		}
		env[key] = value
		filteredEnv[key] = value
	}

	cfg.Env = filteredEnv
	cfg.ConfiguredAt = s.now()
	state.TLS = cfg

	if err := s.writeEnv(env); err != nil {
		return ActionResponse{}, err
	}
	if err := s.writeOverride(cfg); err != nil {
		return ActionResponse{}, err
	}
	if err := s.deployStack(ctx, env); err != nil {
		state.Update.LastAction = "tls-configure"
		state.Update.LastMessage = err.Error()
		state.Update.LastActionAt = s.now()
		_ = s.saveState(state)
		return ActionResponse{}, err
	}

	state.TLS.AppliedAt = s.now()
	state.Update.LastAction = "tls-configure"
	state.Update.LastMessage = "TLS configuration applied"
	state.Update.LastActionAt = s.now()
	if err := s.saveState(state); err != nil {
		return ActionResponse{}, err
	}
	return ActionResponse{
		Status:    "ok",
		Message:   "TLS configuration applied",
		AppliedAt: state.TLS.AppliedAt,
	}, nil
}

func (s *Service) ConfigureDynamicDNS(_ context.Context, cfg DynamicDNSConfig) (ActionResponse, error) {
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Zone = strings.TrimSpace(cfg.Zone)
	cfg.RecordName = strings.TrimSpace(cfg.RecordName)
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = 5
	}

	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
	}

	if !cfg.Enabled {
		cfg.Env = map[string]string{}
		cfg.ConfiguredAt = s.now()
		state.DynamicDNS = cfg
		if err := s.saveState(state); err != nil {
			return ActionResponse{}, err
		}
		return ActionResponse{Status: "ok", Message: "Dynamic DNS disabled", AppliedAt: cfg.ConfiguredAt}, nil
	}

	if cfg.Provider == "" {
		return ActionResponse{}, errors.New("provider is required")
	}
	if cfg.Zone == "" {
		return ActionResponse{}, errors.New("zone is required")
	}
	if cfg.RecordName == "" {
		return ActionResponse{}, errors.New("recordName is required")
	}

	allowedFields := map[string]struct{}{}
	var providerFound bool
	for _, provider := range s.DynamicDNSProviders() {
		if provider.ID != cfg.Provider {
			continue
		}
		providerFound = true
		for _, field := range provider.Fields {
			allowedFields[field.Key] = struct{}{}
			value := strings.TrimSpace(cfg.Env[field.Key])
			if value == "" && state.DynamicDNS.Provider == cfg.Provider {
				value = strings.TrimSpace(state.DynamicDNS.Env[field.Key])
			}
			if value == "" {
				return ActionResponse{}, fmt.Errorf("%s is required", field.Key)
			}
			cfg.Env[field.Key] = value
		}
	}
	if !providerFound {
		return ActionResponse{}, errors.New("unsupported provider")
	}

	filteredEnv := map[string]string{}
	for key, value := range cfg.Env {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, ok := allowedFields[key]; ok {
			filteredEnv[key] = value
		}
	}
	cfg.Env = filteredEnv
	cfg.ConfiguredAt = s.now()
	cfg.LastError = ""
	state.DynamicDNS = cfg
	if err := s.saveState(state); err != nil {
		return ActionResponse{}, err
	}
	return ActionResponse{Status: "ok", Message: "Dynamic DNS settings saved", AppliedAt: cfg.ConfiguredAt}, nil
}

func (s *Service) SyncDynamicDNS(ctx context.Context) (ActionResponse, error) {
	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
	}
	cfg := state.DynamicDNS
	if !cfg.Enabled {
		return ActionResponse{Status: "ok", Message: "Dynamic DNS is disabled"}, nil
	}
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = 5
	}

	ip, err := s.fetchPublicIP(ctx)
	cfg.LastCheckedAt = s.now()
	if err != nil {
		cfg.LastError = err.Error()
		state.DynamicDNS = cfg
		_ = s.saveState(state)
		return ActionResponse{}, err
	}

	if ip == cfg.LastKnownIP {
		cfg.LastError = ""
		state.DynamicDNS = cfg
		if err := s.saveState(state); err != nil {
			return ActionResponse{}, err
		}
		return ActionResponse{
			Status:    "ok",
			Message:   "Public IP unchanged",
			AppliedAt: cfg.LastCheckedAt,
		}, nil
	}

	if err := s.updateDynamicDNSRecord(ctx, cfg, ip); err != nil {
		cfg.LastError = err.Error()
		state.DynamicDNS = cfg
		_ = s.saveState(state)
		return ActionResponse{}, err
	}

	cfg.LastKnownIP = ip
	cfg.LastSyncedAt = s.now()
	cfg.LastCheckedAt = cfg.LastSyncedAt
	cfg.LastError = ""
	state.DynamicDNS = cfg
	if err := s.saveState(state); err != nil {
		return ActionResponse{}, err
	}
	return ActionResponse{
		Status:    "ok",
		Message:   "Dynamic DNS synced",
		AppliedAt: cfg.LastSyncedAt,
	}, nil
}

func (s *Service) CheckUpdate(ctx context.Context) (UpdateCheckResponse, error) {
	env, err := s.loadEnv()
	if err != nil {
		return UpdateCheckResponse{}, err
	}
	currentTag := firstNonEmpty(env["GLYCOVIEW_TAG"], "latest")
	latestTag, releaseURL, err := s.fetchLatestRelease(ctx)
	if err != nil {
		return UpdateCheckResponse{}, err
	}

	state, err := s.loadState()
	if err != nil {
		return UpdateCheckResponse{}, err
	}
	state.Update.CurrentTag = currentTag
	state.Update.CurrentAgentTag = firstNonEmpty(env["GLYCOVIEW_AGENT_TAG"], "latest")
	state.Update.LastCheckedTag = latestTag
	state.Update.LastCheckedAt = s.now()
	_ = s.saveState(state)

	return UpdateCheckResponse{
		CurrentTag:      currentTag,
		LatestTag:       latestTag,
		UpdateAvailable: latestTag != "" && latestTag != currentTag,
		ReleaseURL:      releaseURL,
		CheckedAt:       state.Update.LastCheckedAt,
		Source:          "github-releases",
	}, nil
}

func (s *Service) ApplyUpdate(ctx context.Context, req ApplyUpdateRequest) (ActionResponse, error) {
	req.Tag = strings.TrimSpace(req.Tag)
	if req.Tag == "" {
		return ActionResponse{}, errors.New("tag is required")
	}

	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
	}
	env, err := s.loadEnv()
	if err != nil {
		return ActionResponse{}, err
	}

	currentTag := firstNonEmpty(env["GLYCOVIEW_TAG"], "latest")
	currentAgentTag := firstNonEmpty(env["GLYCOVIEW_AGENT_TAG"], "latest")
	state.Update.PreviousTag = currentTag
	state.Update.CurrentTag = req.Tag
	if req.IncludeAgent {
		state.Update.PreviousAgentTag = currentAgentTag
		state.Update.CurrentAgentTag = req.Tag
		env["GLYCOVIEW_AGENT_TAG"] = req.Tag
	}

	env["GLYCOVIEW_TAG"] = req.Tag
	if err := s.writeEnv(env); err != nil {
		return ActionResponse{}, err
	}
	if err := s.writeOverride(state.TLS); err != nil {
		return ActionResponse{}, err
	}
	if err := s.deployStack(ctx, env); err != nil {
		state.Update.LastAction = "update-apply"
		state.Update.LastMessage = err.Error()
		state.Update.LastActionAt = s.now()
		_ = s.saveState(state)
		return ActionResponse{}, err
	}

	state.Update.LastAction = "update-apply"
	state.Update.LastMessage = "Update applied"
	state.Update.LastActionAt = s.now()
	if err := s.saveState(state); err != nil {
		return ActionResponse{}, err
	}
	return ActionResponse{
		Status:          "ok",
		Message:         "Update applied",
		CurrentTag:      req.Tag,
		CurrentAgentTag: firstNonEmpty(env["GLYCOVIEW_AGENT_TAG"], currentAgentTag),
		AppliedAt:       state.Update.LastActionAt,
	}, nil
}

func (s *Service) Rollback(ctx context.Context) (ActionResponse, error) {
	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
	}
	if state.Update.PreviousTag == "" {
		return ActionResponse{}, errors.New("no previous tag is available")
	}
	env, err := s.loadEnv()
	if err != nil {
		return ActionResponse{}, err
	}
	currentTag := firstNonEmpty(env["GLYCOVIEW_TAG"], "latest")
	env["GLYCOVIEW_TAG"] = state.Update.PreviousTag
	if state.Update.PreviousAgentTag != "" {
		env["GLYCOVIEW_AGENT_TAG"] = state.Update.PreviousAgentTag
	}

	if err := s.writeEnv(env); err != nil {
		return ActionResponse{}, err
	}
	if err := s.writeOverride(state.TLS); err != nil {
		return ActionResponse{}, err
	}
	if err := s.deployStack(ctx, env); err != nil {
		state.Update.LastAction = "update-rollback"
		state.Update.LastMessage = err.Error()
		state.Update.LastActionAt = s.now()
		_ = s.saveState(state)
		return ActionResponse{}, err
	}

	newTag := env["GLYCOVIEW_TAG"]
	previousTag := currentTag
	state.Update.CurrentTag = newTag
	state.Update.PreviousTag = previousTag
	if previousAgent := env["GLYCOVIEW_AGENT_TAG"]; previousAgent != "" {
		state.Update.CurrentAgentTag = previousAgent
		state.Update.PreviousAgentTag = firstNonEmpty(state.Update.CurrentAgentTag, previousAgent)
	}
	state.Update.LastAction = "update-rollback"
	state.Update.LastMessage = "Rollback applied"
	state.Update.LastActionAt = s.now()
	if err := s.saveState(state); err != nil {
		return ActionResponse{}, err
	}
	return ActionResponse{
		Status:          "ok",
		Message:         "Rollback applied",
		CurrentTag:      newTag,
		CurrentAgentTag: env["GLYCOVIEW_AGENT_TAG"],
		AppliedAt:       state.Update.LastActionAt,
	}, nil
}

func (s *Service) fetchLatestRelease(ctx context.Context) (string, string, error) {
	url := "https://api.github.com/repos/" + s.cfg.ReleasesRepo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "glycoview-agent")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("release check failed with status %d", resp.StatusCode)
	}
	var payload releasePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(payload.TagName), strings.TrimSpace(payload.HTMLURL), nil
}

func (s *Service) deployStack(ctx context.Context, env map[string]string) error {
	if _, err := os.Stat(s.cfg.StackFile); err != nil {
		return fmt.Errorf("stack file not found: %s", s.cfg.StackFile)
	}
	if _, err := os.Stat(s.cfg.OverrideFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	args := []string{"stack", "deploy", "--with-registry-auth", "-c", s.cfg.StackFile}
	if _, err := os.Stat(s.cfg.OverrideFile); err == nil {
		args = append(args, "-c", s.cfg.OverrideFile)
	}
	args = append(args, s.cfg.StackName)
	_, err := s.runner.Run(ctx, env, "docker", args...)
	return err
}

func (s *Service) dockerAvailable(ctx context.Context) bool {
	_, err := s.runner.Run(ctx, nil, "docker", "info", "--format", "{{.ServerVersion}}")
	return err == nil
}

func (s *Service) loadState() (State, error) {
	path := filepath.Join(s.cfg.StateDir, "state.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var envelope encryptedState
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Algorithm != "" && envelope.Ciphertext != "" {
		data, err = s.decryptState(data)
		if err != nil {
			return State{}, err
		}
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.TLS.Env == nil {
		state.TLS.Env = map[string]string{}
	}
	if state.DynamicDNS.Env == nil {
		state.DynamicDNS.Env = map[string]string{}
	}
	return state, nil
}

func (s *Service) saveState(state State) error {
	if state.TLS.Env == nil {
		state.TLS.Env = map[string]string{}
	}
	if state.DynamicDNS.Env == nil {
		state.DynamicDNS.Env = map[string]string{}
	}
	if err := os.MkdirAll(s.cfg.StateDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if len(s.encryptionKey()) > 0 {
		data, err = s.encryptState(data)
		if err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(s.cfg.StateDir, "state.json"), data, 0o600)
}

func (s *Service) loadEnv() (map[string]string, error) {
	env := map[string]string{}
	data, err := os.ReadFile(s.cfg.StackEnvFile)
	if errors.Is(err, os.ErrNotExist) {
		return env, nil
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, nil
}

func (s *Service) writeEnv(env map[string]string) error {
	dir := filepath.Dir(s.cfg.StackEnvFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+sanitizeEnvValue(env[key]))
	}
	return os.WriteFile(s.cfg.StackEnvFile, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func (s *Service) writeOverride(cfg TLSConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.cfg.OverrideFile), 0o755); err != nil {
		return err
	}
	command := []string{
		"      - --certificatesresolvers.letsencrypt.acme.email=${LETSENCRYPT_EMAIL}",
		"      - --certificatesresolvers.letsencrypt.acme.storage=/data/acme.json",
	}
	environmentLines := []string{}
	switch cfg.ChallengeType {
	case "dns-01":
		command = append(command,
			"      - --certificatesresolvers.letsencrypt.acme.dnschallenge.provider=${GLYCOVIEW_ACME_DNS_PROVIDER}",
			"      - --certificatesresolvers.letsencrypt.acme.dnschallenge.delaybeforecheck=10",
		)
		keys := make([]string, 0, len(cfg.Env))
		for key := range cfg.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			environmentLines = append(environmentLines, fmt.Sprintf("      %s: ${%s}", key, key))
		}
	default:
		command = append(command, "      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web")
	}

	builder := strings.Builder{}
	builder.WriteString("version: \"3.9\"\n\nservices:\n  traefik:\n    command:\n")
	builder.WriteString(strings.Join(command, "\n"))
	builder.WriteString("\n")
	if len(environmentLines) > 0 {
		builder.WriteString("    environment:\n")
		builder.WriteString(strings.Join(environmentLines, "\n"))
		builder.WriteString("\n")
	}
	return os.WriteFile(s.cfg.OverrideFile, []byte(builder.String()), 0o600)
}

func (s *Service) fetchPublicIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.PublicIPURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "glycoview-agent")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("public IP lookup failed with status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(data))
	if ip == "" {
		return "", errors.New("public IP lookup returned empty response")
	}
	return ip, nil
}

func (s *Service) updateDynamicDNSRecord(ctx context.Context, cfg DynamicDNSConfig, ip string) error {
	switch cfg.Provider {
	case "cloudflare":
		return s.updateCloudflareRecord(ctx, cfg, ip)
	default:
		return fmt.Errorf("dynamic DNS provider %q is not implemented", cfg.Provider)
	}
}

func (s *Service) updateCloudflareRecord(ctx context.Context, cfg DynamicDNSConfig, ip string) error {
	token := strings.TrimSpace(cfg.Env["CF_DNS_API_TOKEN"])
	if token == "" {
		return errors.New("CF_DNS_API_TOKEN is required")
	}

	zoneID, err := s.cloudflareZoneID(ctx, token, cfg.Zone)
	if err != nil {
		return err
	}
	recordID, err := s.cloudflareRecordID(ctx, token, zoneID, cfg.RecordName)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"type":    "A",
		"name":    cfg.RecordName,
		"content": ip,
		"ttl":     1,
		"proxied": false,
	}

	if recordID == "" {
		return s.cloudflareRequest(ctx, token, http.MethodPost, "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records", payload, nil)
	}
	return s.cloudflareRequest(ctx, token, http.MethodPut, "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records/"+recordID, payload, nil)
}

func (s *Service) cloudflareZoneID(ctx context.Context, token, zone string) (string, error) {
	var body struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := s.cloudflareRequest(ctx, token, http.MethodGet, "https://api.cloudflare.com/client/v4/zones?name="+url.QueryEscape(zone), nil, &body); err != nil {
		return "", err
	}
	if len(body.Result) == 0 || strings.TrimSpace(body.Result[0].ID) == "" {
		return "", fmt.Errorf("cloudflare zone %q not found", zone)
	}
	return body.Result[0].ID, nil
}

func (s *Service) cloudflareRecordID(ctx context.Context, token, zoneID, recordName string) (string, error) {
	var body struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := s.cloudflareRequest(ctx, token, http.MethodGet, "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records?type=A&name="+url.QueryEscape(recordName), nil, &body); err != nil {
		return "", err
	}
	if len(body.Result) == 0 {
		return "", nil
	}
	return strings.TrimSpace(body.Result[0].ID), nil
}

func (s *Service) cloudflareRequest(ctx context.Context, token, method, url string, body any, dst any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "glycoview-agent")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope struct {
		Success bool             `json:"success"`
		Errors  []map[string]any `json:"errors"`
	}
	if dst != nil {
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return err
		}
		payload, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || !envelope.Success {
			return fmt.Errorf("cloudflare API request failed with status %d", resp.StatusCode)
		}
		return json.Unmarshal(payload, dst)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloudflare API request failed with status %d", resp.StatusCode)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &envelope); err == nil && !envelope.Success {
		return fmt.Errorf("cloudflare API request failed")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func sanitizeEnvValue(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
}

func (s *Service) redactedTLSConfig(cfg TLSConfig) TLSConfig {
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	provider, ok := s.providerByID(cfg.Provider)
	if !ok {
		return cfg
	}
	redacted := map[string]string{}
	for key, value := range cfg.Env {
		redacted[key] = value
	}
	for _, field := range provider.Fields {
		if field.Secret {
			delete(redacted, field.Key)
		}
	}
	cfg.Env = redacted
	return cfg
}

func (s *Service) redactedDynamicDNSConfig(cfg DynamicDNSConfig) DynamicDNSConfig {
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	provider, ok := s.dynamicDNSProviderByID(cfg.Provider)
	if !ok {
		return cfg
	}
	redacted := map[string]string{}
	for key, value := range cfg.Env {
		redacted[key] = value
	}
	for _, field := range provider.Fields {
		if field.Secret {
			delete(redacted, field.Key)
		}
	}
	cfg.Env = redacted
	return cfg
}

func (s *Service) providerByID(id string) (TLSProvider, bool) {
	for _, provider := range s.Providers() {
		if provider.ID == id {
			return provider, true
		}
	}
	return TLSProvider{}, false
}

func (s *Service) dynamicDNSProviderByID(id string) (DynamicDNSProvider, bool) {
	for _, provider := range s.DynamicDNSProviders() {
		if provider.ID == id {
			return provider, true
		}
	}
	return DynamicDNSProvider{}, false
}

func (s *Service) encryptionKey() []byte {
	secret := strings.TrimSpace(s.cfg.EncryptionKey)
	if secret == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func (s *Service) encryptState(plaintext []byte) ([]byte, error) {
	key := s.encryptionKey()
	if len(key) == 0 {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	payload := encryptedState{
		Version:    1,
		Algorithm:  "aes-256-gcm",
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(aead.Seal(nil, nonce, plaintext, nil)),
	}
	return json.MarshalIndent(payload, "", "  ")
}

func (s *Service) decryptState(data []byte) ([]byte, error) {
	key := s.encryptionKey()
	if len(key) == 0 {
		return nil, errors.New("encrypted appliance state requires GLYCOVIEW_AGENT_STATE_KEY or GLYCOVIEW_AGENT_TOKEN")
	}
	var payload encryptedState
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Algorithm != "aes-256-gcm" {
		return nil, fmt.Errorf("unsupported state encryption algorithm %q", payload.Algorithm)
	}
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ciphertext, nil)
}
