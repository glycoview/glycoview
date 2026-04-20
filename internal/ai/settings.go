package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/glycoview/glycoview/internal/dashboardauth"
)

const (
	settingKeyConfig = "ai.config"
)

type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// Load reads settings from the store. Missing keys return defaults; the API
// key is returned as-is (callers redact before sending to the UI).
func Load(ctx context.Context, store SettingsStore) (Settings, error) {
	defaults := DefaultSettings()
	raw, err := store.GetSetting(ctx, settingKeyConfig)
	if err != nil {
		if errors.Is(err, dashboardauth.ErrNotFound) {
			return defaults, nil
		}
		return defaults, err
	}
	var s Settings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return defaults, err
	}
	if strings.TrimSpace(s.BaseURL) == "" {
		s.BaseURL = defaults.BaseURL
	}
	if strings.TrimSpace(s.Model) == "" {
		s.Model = defaults.Model
	}
	return s, nil
}

func Save(ctx context.Context, store SettingsStore, s Settings) error {
	if strings.TrimSpace(s.BaseURL) == "" {
		s.BaseURL = DefaultSettings().BaseURL
	}
	if strings.TrimSpace(s.Model) == "" {
		s.Model = DefaultSettings().Model
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return store.SetSetting(ctx, settingKeyConfig, string(data))
}

// Redact masks the API key for display in the admin UI. Returns the first
// four and last four characters with a fixed-length middle mask so different
// key lengths render identically.
func Redact(s Settings) Settings {
	redacted := s
	redacted.APIKey = maskKey(s.APIKey)
	return redacted
}

func maskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	if len(k) <= 8 {
		return strings.Repeat("•", len(k))
	}
	return k[:4] + strings.Repeat("•", 8) + k[len(k)-4:]
}
