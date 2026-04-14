package appliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Config struct {
	StateDir     string
	StackName    string
	StackFile    string
	StackEnvFile string
	OverrideFile string
	ReleasesRepo string
	AppImage     string
	AgentImage   string
	HTTPClient   *http.Client
	Runner       Runner
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
		TLS:               state.TLS,
	}, nil
}

func (s *Service) TLSConfig(_ context.Context) (TLSConfig, error) {
	state, err := s.loadState()
	if err != nil {
		return TLSConfig{}, err
	}
	return state.TLS, nil
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
				if strings.TrimSpace(cfg.Env[field.Key]) == "" {
					return ActionResponse{}, fmt.Errorf("%s is required", field.Key)
				}
			}
			break
		}
		if !found {
			return ActionResponse{}, errors.New("unsupported provider")
		}
	}

	state, err := s.loadState()
	if err != nil {
		return ActionResponse{}, err
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
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.TLS.Env == nil {
		state.TLS.Env = map[string]string{}
	}
	return state, nil
}

func (s *Service) saveState(state State) error {
	if state.TLS.Env == nil {
		state.TLS.Env = map[string]string{}
	}
	if err := os.MkdirAll(s.cfg.StateDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
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
