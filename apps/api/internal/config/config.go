package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the API server configuration.
type Config struct {
	Port                string
	LogLevel            slog.Level
	DatabaseURL         string
	ReconcileInterval   time.Duration
	KubeconfigPath      string
	KnativeNamespace    string
	ResendAPIKey        string
	EmailFrom           string
	InviteExpiration    time.Duration
	DevMode             bool
	DevOrgUID           string
	RegistryURL         string
	RegistryJWTSecret   string
	RegistryTokenExpiry time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	_ = godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	reconcileInterval := 5 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			reconcileInterval = d
		}
	}

	knativeNamespace := os.Getenv("KNATIVE_NAMESPACE")
	if knativeNamespace == "" {
		knativeNamespace = "default"
	}

	emailFrom := os.Getenv("EMAIL_FROM")
	if emailFrom == "" {
		emailFrom = "noreply@maxcloud.dev"
	}

	inviteExpiration := 168 * time.Hour // 7 Tage
	if v := os.Getenv("INVITE_EXPIRATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			inviteExpiration = d
		}
	}

	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = "registry.maxcloud.dev"
	}

	registryTokenExpiry := 1 * time.Hour
	if v := os.Getenv("REGISTRY_TOKEN_EXPIRY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			registryTokenExpiry = d
		}
	}

	return &Config{
		Port:                port,
		LogLevel:            slog.LevelInfo,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		ReconcileInterval:   reconcileInterval,
		KubeconfigPath:      os.Getenv("KUBECONFIG"),
		KnativeNamespace:    knativeNamespace,
		ResendAPIKey:        os.Getenv("RESEND_API_KEY"),
		EmailFrom:           emailFrom,
		InviteExpiration:    inviteExpiration,
		DevMode:             os.Getenv("DEV_MODE") == "true",
		DevOrgUID:           os.Getenv("DEV_ORG_UID"),
		RegistryURL:         registryURL,
		RegistryJWTSecret:   os.Getenv("REGISTRY_JWT_SECRET"),
		RegistryTokenExpiry: registryTokenExpiry,
	}
}
