package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port string

	// Database settings
	DatabaseURL string

	// Redis settings
	RedisURL string

	// Security settings
	HMACKey   string
	JWTSecret string

	// Rate limiting defaults
	DefaultDailyLimit   int
	DefaultMonthlyLimit int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://regstrava:regstrava@localhost:5432/regstrava?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		HMACKey:             os.Getenv("HMAC_KEY"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		DefaultDailyLimit:   1000,
		DefaultMonthlyLimit: 20000,
	}

	// Validate required settings
	if cfg.HMACKey == "" {
		return nil, fmt.Errorf("HMAC_KEY environment variable is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
