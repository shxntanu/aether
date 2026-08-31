package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// HTTPAddress is the TCP address used by the API server.
	HTTPAddress string
	// ShutdownTimeout bounds graceful HTTP shutdown.
	ShutdownTimeout time.Duration
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// StorageProvider selects the provider package wired at startup.
	StorageProvider string
	// LocalStoragePath is the persistent root used by the local provider.
	LocalStoragePath string
	// GoogleDriveClientID identifies the vault-owner Drive OAuth client.
	GoogleDriveClientID string
	// GoogleDriveClientSecret authenticates the vault-owner Drive OAuth client.
	GoogleDriveClientSecret string
	// GoogleDriveRedirectURL is the exact Drive-owner OAuth callback.
	GoogleDriveRedirectURL string
	// GoogleDriveFolderID is the app-managed Drive folder for vault objects.
	GoogleDriveFolderID string
	// GoogleDriveRefreshToken is the deployment secret for offline Drive access.
	GoogleDriveRefreshToken string
	// GoogleClientID identifies the Google OIDC web client.
	GoogleClientID string
	// GoogleClientSecret authenticates the Google OIDC web client.
	GoogleClientSecret string
	// GoogleRedirectURL is the exact registered login callback.
	GoogleRedirectURL string
	// BootstrapAdminEmail seeds the first allowlisted administrator.
	BootstrapAdminEmail string
	// SecureCookies requires HTTPS transport for session cookies.
	SecureCookies bool
}

// Load reads and validates process configuration from AETHER_* variables.
func Load() (Config, error) {
	config := Config{
		HTTPAddress:      ":8080",
		ShutdownTimeout:  10 * time.Second,
		DatabaseURL:      "postgres://aether:aether@127.0.0.1:5432/aether?sslmode=disable",
		StorageProvider:  "local",
		LocalStoragePath: "./data/vault",
	}

	if value := os.Getenv("AETHER_HTTP_ADDRESS"); value != "" {
		config.HTTPAddress = value
	}
	if value := os.Getenv("AETHER_DATABASE_URL"); value != "" {
		config.DatabaseURL = value
	}
	if value := os.Getenv("AETHER_STORAGE_PROVIDER"); value != "" {
		config.StorageProvider = value
	}
	if value := os.Getenv("AETHER_LOCAL_STORAGE_PATH"); value != "" {
		config.LocalStoragePath = value
	}
	if config.StorageProvider != "local" && config.StorageProvider != "gdrive" {
		return Config{}, fmt.Errorf("AETHER_STORAGE_PROVIDER must be local or gdrive")
	}
	if config.StorageProvider == "local" && config.LocalStoragePath == "" {
		return Config{}, fmt.Errorf("AETHER_LOCAL_STORAGE_PATH must not be empty")
	}
	config.GoogleDriveClientID = os.Getenv("AETHER_GDRIVE_CLIENT_ID")
	config.GoogleDriveClientSecret = os.Getenv("AETHER_GDRIVE_CLIENT_SECRET")
	config.GoogleDriveRedirectURL = os.Getenv("AETHER_GDRIVE_REDIRECT_URL")
	config.GoogleDriveFolderID = os.Getenv("AETHER_GDRIVE_FOLDER_ID")
	config.GoogleDriveRefreshToken = os.Getenv("AETHER_GDRIVE_REFRESH_TOKEN")
	driveValues := []string{
		config.GoogleDriveClientID,
		config.GoogleDriveClientSecret,
		config.GoogleDriveRedirectURL,
		config.GoogleDriveFolderID,
		config.GoogleDriveRefreshToken,
	}
	configuredDriveValues := 0
	for _, value := range driveValues {
		if value != "" {
			configuredDriveValues++
		}
	}
	if configuredDriveValues != 0 && configuredDriveValues != len(driveValues) {
		return Config{}, fmt.Errorf(
			"Google Drive storage requires client ID, client secret, redirect URL, " +
				"folder ID, and refresh token",
		)
	}
	if config.StorageProvider == "gdrive" && configuredDriveValues != len(driveValues) {
		return Config{}, fmt.Errorf(
			"AETHER_STORAGE_PROVIDER=gdrive requires complete Google Drive configuration",
		)
	}
	if configuredDriveValues == len(driveValues) {
		parsed, err := url.Parse(config.GoogleDriveRedirectURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
			parsed.Path != "/auth/gdrive/callback" || parsed.RawQuery != "" ||
			parsed.Fragment != "" {
			return Config{}, fmt.Errorf(
				"AETHER_GDRIVE_REDIRECT_URL must be an absolute URL with exact path " +
					"/auth/gdrive/callback and no query or fragment",
			)
		}
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
	identityValues := []string{
		config.GoogleClientID,
		config.GoogleClientSecret,
		config.GoogleRedirectURL,
		config.BootstrapAdminEmail,
	}
	configured := 0
	for _, value := range identityValues {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(identityValues) {
		return Config{}, fmt.Errorf(
			"Google OIDC requires client ID, client secret, redirect URL, " +
				"and bootstrap administrator email",
		)
	}
	if configured == len(identityValues) {
		parsed, err := url.Parse(config.GoogleRedirectURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
			parsed.Path != "/auth/google/callback" || parsed.RawQuery != "" ||
			parsed.Fragment != "" {
			return Config{}, fmt.Errorf(
				"AETHER_GOOGLE_REDIRECT_URL must be an absolute URL with exact path " +
					"/auth/google/callback and no query or fragment",
			)
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
