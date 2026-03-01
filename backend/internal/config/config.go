package config

import (
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port      string `yaml:"port"`
		Mode      string `yaml:"mode"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"server"`

	Database struct {
		URL          string `yaml:"url"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		MaxOpenConns int    `yaml:"max_open_conns"`
	} `yaml:"database"`

	Cors struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"cors"`

	Vault struct {
		Enabled    bool   `yaml:"enabled"`
		Address    string `yaml:"address"`
		Token      string `yaml:"token"`
		MountPath  string `yaml:"mount_path"`
		PathPrefix string `yaml:"path_prefix"`
	} `yaml:"vault"`
}

func Load() *Config {
	cfg := &Config{}

	// Default values
	cfg.Server.Port = "8000"
	cfg.Server.Mode = "debug"
	cfg.Server.SecretKey = "change-this-in-production"
	cfg.Database.URL = getEnv("DATABASE_URL", "postgresql://osbaka:password@localhost:5432/osbaka")

	// Try loading from config.yaml
	// Check standard locations: ./config.yaml, ../config.yaml, /etc/osbaka/config.yaml
	locations := []string{"config.yaml", "../config.yaml", "backend/config.yaml"}

	var file *os.File
	var err error

	for _, loc := range locations {
		file, err = os.Open(loc)
		if err == nil {
			slog.Info("Loading config", "path", loc)
			break
		}
	}

	if file != nil {
		defer file.Close()
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(cfg); err != nil {
			slog.Error("Failed to decode config file", "error", err)
		}
	} else {
		slog.Info("No config file found, using defaults/env vars")
	}

	// Override with Env vars if present (highest precedence)
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		cfg.Server.SecretKey = secretKey
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		cfg.Server.Mode = mode
	}

	// Vault config from env vars
	if vaultEnabled := os.Getenv("VAULT_ENABLED"); vaultEnabled != "" {
		cfg.Vault.Enabled = strings.EqualFold(vaultEnabled, "true") || vaultEnabled == "1"
	}
	if vaultAddr := os.Getenv("VAULT_ADDR"); vaultAddr != "" {
		cfg.Vault.Address = vaultAddr
	}
	if vaultToken := os.Getenv("VAULT_TOKEN"); vaultToken != "" {
		cfg.Vault.Token = vaultToken
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
