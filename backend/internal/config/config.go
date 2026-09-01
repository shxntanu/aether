package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains the validated process settings needed to start Aether.
type Config struct {
	// HTTPAddress is the TCP address used by the API server.
	HTTPAddress string
	// ShutdownTimeout bounds graceful HTTP shutdown.
	ShutdownTimeout time.Duration
	// PublicURL is the normalized external origin used for browser-facing checks.
	PublicURL *url.URL
	// TrustedProxyRanges enumerates proxies allowed to supply forwarding headers.
	TrustedProxyRanges []*net.IPNet
	// RequestTimeout bounds ordinary HTTP request handling.
	RequestTimeout time.Duration
	// UploadTimeout bounds upload request handling.
	UploadTimeout time.Duration
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
		HTTPAddress:     ":8080",
		ShutdownTimeout: 10 * time.Second,
		RequestTimeout:  30 * time.Second,
		UploadTimeout:   10 * time.Minute,
		DatabaseURL: "postgres://aether:aether@127.0.0.1:5432/" +
			"aether?sslmode=disable",
		StorageProvider:  "local",
		LocalStoragePath: "./data/vault",
	}

	if value := os.Getenv("AETHER_HTTP_ADDRESS"); value != "" {
		config.HTTPAddress = value
	}
	if value := os.Getenv("AETHER_DATABASE_URL"); value != "" {
		config.DatabaseURL = value
	}
	publicURL, err := parsePublicURL(os.Getenv("AETHER_PUBLIC_URL"))
	if err != nil {
		return Config{}, err
	}
	config.PublicURL = publicURL
	trustedProxyRanges, err := parseTrustedProxyRanges(
		os.Getenv("AETHER_TRUSTED_PROXY_CIDRS"),
	)
	if err != nil {
		return Config{}, err
	}
	config.TrustedProxyRanges = trustedProxyRanges
	if value := os.Getenv("AETHER_REQUEST_TIMEOUT"); value != "" {
		timeout, err := parsePositiveDuration("AETHER_REQUEST_TIMEOUT", value)
		if err != nil {
			return Config{}, err
		}
		config.RequestTimeout = timeout
	}
	if value := os.Getenv("AETHER_UPLOAD_TIMEOUT"); value != "" {
		timeout, err := parsePositiveDuration("AETHER_UPLOAD_TIMEOUT", value)
		if err != nil {
			return Config{}, err
		}
		config.UploadTimeout = timeout
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
		if config.PublicURL == nil {
			return Config{}, fmt.Errorf(
				"AETHER_PUBLIC_URL must be configured when Google OIDC is enabled",
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

func parsePublicURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse AETHER_PUBLIC_URL: %w", err)
	}
	if !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf(
			"AETHER_PUBLIC_URL must be an absolute URL with no userinfo, query, " +
				"or fragment",
		)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("AETHER_PUBLIC_URL must use http or https")
	}
	if parsed.Scheme != "https" && !isLoopbackHost(parsed.Hostname()) {
		return nil, fmt.Errorf(
			"AETHER_PUBLIC_URL must use https unless it targets localhost or a " +
				"loopback address",
		)
	}
	normalized := *parsed
	normalized.Path = "/"
	normalized.RawPath = ""
	normalized.ForceQuery = false
	normalized.RawQuery = ""
	normalized.Fragment = ""
	return &normalized, nil
}

func parseTrustedProxyRanges(raw string) ([]*net.IPNet, error) {
	if raw == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ",")
	ranges := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		candidate := strings.TrimSpace(entry)
		if candidate == "" {
			return nil, fmt.Errorf(
				"AETHER_TRUSTED_PROXY_CIDRS must not contain empty entries",
			)
		}
		if !strings.Contains(candidate, "/") {
			return nil, fmt.Errorf(
				"AETHER_TRUSTED_PROXY_CIDRS entries must be CIDR ranges, not " +
					"individual IPs",
			)
		}
		_, network, err := net.ParseCIDR(candidate)
		if err != nil {
			return nil, fmt.Errorf(
				"parse AETHER_TRUSTED_PROXY_CIDRS entry %q: %w",
				candidate,
				err,
			)
		}
		ranges = append(ranges, network)
	}
	return ranges, nil
}

func parsePositiveDuration(name, raw string) (time.Duration, error) {
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return duration, nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
