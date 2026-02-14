package config

import (
	"log/slog"
	"os"
)

// Config holds the API server configuration.
type Config struct {
	Port     string
	LogLevel slog.Level
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:     port,
		LogLevel: slog.LevelInfo,
	}
}
