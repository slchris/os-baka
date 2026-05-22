package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.Server.Port == "" {
		t.Error("Server.Port should have a default value")
	}
	if cfg.Database.URL == "" {
		t.Error("Database.URL should have a default value")
	}
	if cfg.Server.SecretKey == "" {
		t.Error("SecretKey should have a default value")
	}
}

func TestLoadDefaultValues(t *testing.T) {
	cfg := Load()

	if cfg.Server.Port != "8000" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "8000")
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "debug")
	}
	if cfg.Server.SecretKey != "change-this-in-production" {
		t.Errorf("SecretKey = %q, want %q", cfg.Server.SecretKey, "change-this-in-production")
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("PORT", "9999")

	cfg := Load()

	if cfg.Server.Port != "9999" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9999")
	}
}

func TestLoadSecretKeyEnvOverride(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret-12345")

	cfg := Load()

	if cfg.Server.SecretKey != "test-secret-12345" {
		t.Errorf("SecretKey = %q, want %q", cfg.Server.SecretKey, "test-secret-12345")
	}
}

func TestLoadDatabaseURLEnvOverride(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://test:test@testhost:5432/testdb")

	cfg := Load()

	if cfg.Database.URL != "postgresql://test:test@testhost:5432/testdb" {
		t.Errorf("Database.URL = %q, want the env value", cfg.Database.URL)
	}
}

func TestLoadGinModeEnvOverride(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv("SECRET_KEY", "test-secret-that-is-at-least-32-chars-long")

	cfg := Load()

	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "release")
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `server:
  port: "7777"
  mode: "release"
  secret_key: "yaml-secret-that-is-at-least-32-characters-long"
database:
  url: "postgresql://yaml:yaml@yamlhost:5432/yamldb"
cors:
  allowed_origins:
    - "http://localhost:3000"
    - "http://example.com"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Change to the temporary directory so Load() finds config.yaml
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	cfg := Load()

	if cfg.Server.Port != "7777" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "7777")
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "release")
	}
	if cfg.Server.SecretKey != "yaml-secret-that-is-at-least-32-characters-long" {
		t.Errorf("SecretKey = %q, want %q", cfg.Server.SecretKey, "yaml-secret-that-is-at-least-32-characters-long")
	}
	if cfg.Database.URL != "postgresql://yaml:yaml@yamlhost:5432/yamldb" {
		t.Errorf("Database.URL = %q, want yaml db value", cfg.Database.URL)
	}
	if len(cfg.Cors.AllowedOrigins) != 2 {
		t.Errorf("Cors.AllowedOrigins len = %d, want 2", len(cfg.Cors.AllowedOrigins))
	}
}

func TestLoadEnvOverridesYAML(t *testing.T) {
	// Create a YAML config, then override with env
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `server:
  port: "7777"
  secret_key: "yaml-secret"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	// Env should override YAML
	t.Setenv("PORT", "8888")
	t.Setenv("SECRET_KEY", "env-secret")

	cfg := Load()

	if cfg.Server.Port != "8888" {
		t.Errorf("Server.Port = %q, want %q (env should override YAML)", cfg.Server.Port, "8888")
	}
	if cfg.Server.SecretKey != "env-secret" {
		t.Errorf("SecretKey = %q, want %q (env should override YAML)", cfg.Server.SecretKey, "env-secret")
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	t.Setenv("TEST_GETENV_VAR", "hello")

	got := getEnv("TEST_GETENV_VAR", "default")
	if got != "hello" {
		t.Errorf("getEnv() = %q, want %q", got, "hello")
	}

	// Test with non-existing env var
	got = getEnv("NON_EXISTING_VAR_12345", "fallback")
	if got != "fallback" {
		t.Errorf("getEnv() = %q, want %q", got, "fallback")
	}
}
