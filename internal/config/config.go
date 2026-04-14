package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr           string
	AppVersion     string
	APISecret      string
	JWTSecret      string
	DatabaseURL    string
	UIBuildOnStart bool
	Enable         []string
	DefaultRoles   []string
	API3MaxLimit   int
}

func Load() Config {
	return Config{
		Addr:           envOrDefault("ADDR", ":8080"),
		AppVersion:     envOrDefault("APP_VERSION", "0.1.0"),
		APISecret:      envOrDefault("API_SECRET", "change-me"),
		JWTSecret:      envOrDefault("JWT_SECRET", envOrDefault("API_SECRET", "change-me")),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		UIBuildOnStart: envBoolOrDefault("UI_BUILD_ON_START", true),
		Enable:         splitCSV(envOrDefault("ENABLE", "careportal,api,rawbg")),
		DefaultRoles:   splitCSV(envOrDefault("AUTH_DEFAULT_ROLES", "readable")),
		API3MaxLimit:   envIntOrDefault("API3_MAX_LIMIT", 1000),
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envIntOrDefault(key string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}
