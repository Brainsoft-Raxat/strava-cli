// Package config manages persistent configuration and token storage for strava-cli.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirName  = "strava-cli"
	fileName = "config.json"
)

// Tokens holds the OAuth2 token pair and metadata.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // Unix timestamp
	TokenType    string `json:"token_type,omitempty"`
}

// PendingAuth holds state between step 1 and step 2 of a remote (two-step) login.
type PendingAuth struct {
	State       string `json:"state"`
	RedirectURI string `json:"redirect_uri"`
	ExpiresAt   int64  `json:"expires_at"` // Unix timestamp
}

// Config is the full persisted configuration.
type Config struct {
	ClientID     string       `json:"client_id"`
	ClientSecret string       `json:"client_secret"`
	RedirectURI  string       `json:"redirect_uri,omitempty"`
	Tokens       Tokens       `json:"tokens,omitempty"`
	PendingAuth  *PendingAuth `json:"pending_auth,omitempty"`
}

// Dir returns the path to the config directory (~/.config/strava-cli/).
// The STRAVA_CONFIG_DIR environment variable overrides the default location;
// set it in tests to avoid touching the real config on disk.
func Dir() (string, error) {
	if override := os.Getenv("STRAVA_CONFIG_DIR"); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(base, dirName), nil
}

// Load reads config from disk. Returns an empty Config if the file doesn't exist yet.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}
