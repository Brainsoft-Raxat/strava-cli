package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

func withTempConfigDir(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	return func() { os.Setenv("XDG_CONFIG_HOME", orig) }
}

func TestLoadSave_RoundTrip(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	cfg := &config.Config{
		ClientID:     "my-id",
		ClientSecret: "my-secret",
		Tokens: config.Tokens{
			AccessToken:  "acc",
			RefreshToken: "ref",
			ExpiresAt:    9999999999,
		},
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ClientID != cfg.ClientID {
		t.Errorf("ClientID = %q, want %q", loaded.ClientID, cfg.ClientID)
	}
	if loaded.Tokens.AccessToken != cfg.Tokens.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.Tokens.AccessToken, cfg.Tokens.AccessToken)
	}
	if loaded.Tokens.ExpiresAt != cfg.Tokens.ExpiresAt {
		t.Errorf("ExpiresAt = %d, want %d", loaded.Tokens.ExpiresAt, cfg.Tokens.ExpiresAt)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if cfg.ClientID != "" {
		t.Errorf("expected empty config, got ClientID=%q", cfg.ClientID)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	// Point to a non-existent subdir to ensure MkdirAll works.
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "nested", "deep"))
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	cfg := &config.Config{ClientID: "test"}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save with nested dir: %v", err)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	cfg := &config.Config{ClientID: "perm-test", ClientSecret: "shh"}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dir, _ := config.Dir()
	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("config.json permissions = %o, want 0600", mode)
	}
}
