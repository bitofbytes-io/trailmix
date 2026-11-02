// Package config reads process configuration from the environment and from
// Docker secret files.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config is everything the process needs to listen and reach Postgres.
type Config struct {
	Port          string
	DatabaseURL   string
	LogLevel      string
	SecureCookies bool
}

// Load reads configuration. DATABASE_URL is required. A direct environment
// value wins, then DATABASE_URL_FILE, then /run/secrets/trailmix_database_url.
func Load() (*Config, error) {
	var err error
	cfg := &Config{}
	if cfg.Port, err = getEnv("PORT", "4600"); err != nil {
		return nil, err
	}
	if cfg.DatabaseURL, err = LoadDatabaseURL(); err != nil {
		return nil, err
	}
	if cfg.LogLevel, err = getEnv("LOG_LEVEL", "info"); err != nil {
		return nil, err
	}
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}

	secure, err := getEnv("SECURE_COOKIES", "true")
	if err != nil {
		return nil, err
	}
	cfg.SecureCookies = secure != "false"

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

// LoadDatabaseURL reads the database URL the same way Load does. The migrate
// command uses it so a one-off job can receive only the database secret.
func LoadDatabaseURL() (string, error) {
	return getEnvOrFile("DATABASE_URL", "/run/secrets/trailmix_database_url")
}

func getEnv(key, defaultVal string) (string, error) {
	if filePath := os.Getenv(key + "_FILE"); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s_FILE (%s): %w", key, filePath, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if val := os.Getenv(key); val != "" {
		return val, nil
	}
	return defaultVal, nil
}

func getEnvOrFile(key, defaultPath string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	fileKey := key + "_FILE"
	if path := os.Getenv(fileKey); path != "" {
		return readSecret(path, fileKey)
	}
	if defaultPath != "" {
		return readSecret(defaultPath, key)
	}
	return "", nil
}

func readSecret(path, name string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("config: reading %s (%s): %w", name, path, err)
	}
	value := strings.TrimSpace(string(contents))
	if value == "" {
		return "", fmt.Errorf("config: %s (%s) is empty", name, path)
	}
	return value, nil
}
