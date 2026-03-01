package config

import (
	"os"
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

func TestLoadEnvOverride(t *testing.T) {
	os.Setenv("PORT", "9999")
	defer os.Unsetenv("PORT")

	cfg := Load()

	if cfg.Server.Port != "9999" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9999")
	}
}

func TestLoadSecretKeyEnvOverride(t *testing.T) {
	os.Setenv("SECRET_KEY", "test-secret-12345")
	defer os.Unsetenv("SECRET_KEY")

	cfg := Load()

	if cfg.Server.SecretKey != "test-secret-12345" {
		t.Errorf("SecretKey = %q, want %q", cfg.Server.SecretKey, "test-secret-12345")
	}
}
