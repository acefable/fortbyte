// Package config provides server configuration loaded from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds server configuration.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   []byte
	MasterKey   []byte
	LogLevel    string
	APIURL      string
}

// Load reads configuration from environment variables and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("PORT", "8080"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	cfg.APIURL = getEnv("API_URL", "")

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwt := os.Getenv("JWT_SECRET")
	if jwt == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwt) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(jwt))
	}
	cfg.JWTSecret = []byte(jwt)

	mk := os.Getenv("MASTER_KEY")
	if mk == "" {
		return nil, fmt.Errorf("MASTER_KEY is required")
	}
	if len(mk) < 32 {
		return nil, fmt.Errorf("MASTER_KEY must be at least 32 bytes, got %d", len(mk))
	}
	cfg.MasterKey = []byte(mk)

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
