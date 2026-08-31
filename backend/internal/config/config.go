package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddress         string
	ShutdownTimeout     time.Duration
	DatabaseURL         string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	BootstrapAdminEmail string
	SecureCookies       bool
}

func Load() (Config, error) {
	config := Config{
		HTTPAddress:     ":8080",
		ShutdownTimeout: 10 * time.Second,
		DatabaseURL:     "postgres://aether:aether@127.0.0.1:5432/aether?sslmode=disable",
	}

	if value := os.Getenv("AETHER_HTTP_ADDRESS"); value != "" {
		config.HTTPAddress = value
	}
	if value := os.Getenv("AETHER_DATABASE_URL"); value != "" {
		config.DatabaseURL = value
	}
	config.GoogleClientID = os.Getenv("AETHER_GOOGLE_CLIENT_ID")
	config.GoogleClientSecret = os.Getenv("AETHER_GOOGLE_CLIENT_SECRET")
	config.GoogleRedirectURL = os.Getenv("AETHER_GOOGLE_REDIRECT_URL")
	config.BootstrapAdminEmail = os.Getenv("AETHER_BOOTSTRAP_ADMIN_EMAIL")
	if value := os.Getenv("AETHER_SECURE_COOKIES"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse AETHER_SECURE_COOKIES: %w", err)
		}
		config.SecureCookies = parsed
	}
	identityValues := []string{config.GoogleClientID, config.GoogleClientSecret, config.GoogleRedirectURL, config.BootstrapAdminEmail}
	configured := 0
	for _, value := range identityValues {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(identityValues) {
		return Config{}, fmt.Errorf("Google OIDC requires client ID, client secret, redirect URL, and bootstrap administrator email")
	}
	if configured == len(identityValues) {
		parsed, err := url.Parse(config.GoogleRedirectURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "/auth/google/callback" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return Config{}, fmt.Errorf("AETHER_GOOGLE_REDIRECT_URL must be an absolute URL with exact path /auth/google/callback and no query or fragment")
		}
	}

	if value := os.Getenv("AETHER_SHUTDOWN_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse AETHER_SHUTDOWN_TIMEOUT: %w", err)
		}
		if timeout <= 0 {
			return Config{}, fmt.Errorf("AETHER_SHUTDOWN_TIMEOUT must be positive")
		}
		config.ShutdownTimeout = timeout
	}

	return config, nil
}
