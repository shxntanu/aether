package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDevelopmentDefaults(t *testing.T) {
	t.Setenv("AETHER_HTTP_ADDRESS", "")
	t.Setenv("AETHER_SHUTDOWN_TIMEOUT", "")
	t.Setenv("AETHER_DATABASE_URL", "")
	t.Setenv("AETHER_STORAGE_PROVIDER", "")
	t.Setenv("AETHER_LOCAL_STORAGE_PATH", "")
	t.Setenv("AETHER_GATEWAY_SECRET", "")
	clearIdentityEnvironment(t)
	clearDriveEnvironment(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.HTTPAddress != ":8080" {
		t.Fatalf("HTTPAddress = %q, want %q", got.HTTPAddress, ":8080")
	}
	if got.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want %s", got.ShutdownTimeout, 10*time.Second)
	}
	wantDatabaseURL := "postgres://aether:aether@127.0.0.1:5432/aether?sslmode=disable"
	if got.DatabaseURL != wantDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", got.DatabaseURL, wantDatabaseURL)
	}
	if got.StorageProvider != "local" || got.LocalStoragePath != "./data/vault" {
		t.Fatalf("storage defaults = %#v", got)
	}
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("AETHER_HTTP_ADDRESS", "127.0.0.1:9000")
	t.Setenv("AETHER_SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("AETHER_DATABASE_URL", "postgres://hosted.example/aether")
	t.Setenv("AETHER_STORAGE_PROVIDER", "local")
	t.Setenv("AETHER_LOCAL_STORAGE_PATH", "/srv/aether/vault")
	clearDriveEnvironment(t)
	t.Setenv("AETHER_GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("AETHER_GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("AETHER_GOOGLE_REDIRECT_URL", "https://vault.example/auth/google/callback")
	t.Setenv("AETHER_BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("AETHER_SECURE_COOKIES", "true")
	t.Setenv("AETHER_PUBLIC_URL", "https://vault.example")
	t.Setenv("AETHER_GATEWAY_SECRET", "01234567890123456789012345678901")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.HTTPAddress != "127.0.0.1:9000" {
		t.Fatalf("HTTPAddress = %q, want %q", got.HTTPAddress, "127.0.0.1:9000")
	}
	if got.ShutdownTimeout != 25*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want %s", got.ShutdownTimeout, 25*time.Second)
	}
	if got.DatabaseURL != "postgres://hosted.example/aether" {
		t.Fatalf("DatabaseURL = %q, want hosted provider URL", got.DatabaseURL)
	}
	if got.StorageProvider != "local" || got.LocalStoragePath != "/srv/aether/vault" {
		t.Fatalf("storage configuration = %#v", got)
	}
	if got.GoogleClientID != "client-id" || got.GoogleRedirectURL != "https://vault.example/auth/google/callback" || !got.SecureCookies {
		t.Fatalf("identity configuration = %#v", got)
	}
	if got.GatewaySecret != "01234567890123456789012345678901" {
		t.Fatalf("GatewaySecret was not loaded")
	}
}

func TestLoadRejectsShortGatewaySecret(t *testing.T) {
	t.Setenv("AETHER_GATEWAY_SECRET", "too-short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AETHER_GATEWAY_SECRET") {
		t.Fatalf("Load() error = %v, want gateway secret validation", err)
	}
}

func TestLoadAcceptsCompleteGoogleDriveConfiguration(t *testing.T) {
	clearIdentityEnvironment(t)
	clearDriveEnvironment(t)
	t.Setenv("AETHER_STORAGE_PROVIDER", "gdrive")
	t.Setenv("AETHER_GDRIVE_CLIENT_ID", "drive-client")
	t.Setenv("AETHER_GDRIVE_CLIENT_SECRET", "drive-secret")
	t.Setenv("AETHER_GDRIVE_REDIRECT_URL", "https://vault.example/auth/gdrive/callback")
	t.Setenv("AETHER_GDRIVE_FOLDER_ID", "folder-id")
	t.Setenv("AETHER_GDRIVE_REFRESH_TOKEN", "refresh-token")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.StorageProvider != "gdrive" || got.GoogleDriveFolderID != "folder-id" {
		t.Fatalf("Drive configuration = %#v", got)
	}
}

func TestLoadRejectsIncompleteGoogleDriveConfiguration(t *testing.T) {
	clearIdentityEnvironment(t)
	clearDriveEnvironment(t)
	t.Setenv("AETHER_STORAGE_PROVIDER", "gdrive")
	t.Setenv("AETHER_GDRIVE_CLIENT_ID", "drive-client")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want incomplete Drive configuration error")
	}
}

func TestLoadRejectsInexactGoogleDriveRedirect(t *testing.T) {
	clearIdentityEnvironment(t)
	clearDriveEnvironment(t)
	for key, value := range map[string]string{
		"AETHER_STORAGE_PROVIDER":     "gdrive",
		"AETHER_GDRIVE_CLIENT_ID":     "drive-client",
		"AETHER_GDRIVE_CLIENT_SECRET": "drive-secret",
		"AETHER_GDRIVE_REDIRECT_URL":  "https://vault.example/callback",
		"AETHER_GDRIVE_FOLDER_ID":     "folder-id",
		"AETHER_GDRIVE_REFRESH_TOKEN": "refresh-token",
	} {
		t.Setenv(key, value)
	}
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid Drive redirect error")
	}
}

func TestLoadRejectsPartialOrInexactOIDCConfiguration(t *testing.T) {
	tests := []struct{ name, redirect, secret, bootstrap string }{
		{name: "missing secret", redirect: "https://vault.example/auth/google/callback", bootstrap: "admin@example.com"},
		{name: "redirect query", redirect: "https://vault.example/auth/google/callback?unexpected=true", secret: "secret", bootstrap: "admin@example.com"},
		{name: "wrong callback path", redirect: "https://vault.example/callback", secret: "secret", bootstrap: "admin@example.com"},
		{name: "missing bootstrap", redirect: "https://vault.example/auth/google/callback", secret: "secret"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearIdentityEnvironment(t)
			t.Setenv("AETHER_GOOGLE_CLIENT_ID", "client")
			t.Setenv("AETHER_GOOGLE_CLIENT_SECRET", test.secret)
			t.Setenv("AETHER_GOOGLE_REDIRECT_URL", test.redirect)
			t.Setenv("AETHER_BOOTSTRAP_ADMIN_EMAIL", test.bootstrap)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want identity configuration error")
			}
		})
	}
}

func clearIdentityEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"AETHER_GOOGLE_CLIENT_ID", "AETHER_GOOGLE_CLIENT_SECRET", "AETHER_GOOGLE_REDIRECT_URL", "AETHER_BOOTSTRAP_ADMIN_EMAIL", "AETHER_SECURE_COOKIES"} {
		t.Setenv(key, "")
	}
}

func clearDriveEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"AETHER_GDRIVE_CLIENT_ID",
		"AETHER_GDRIVE_CLIENT_SECRET",
		"AETHER_GDRIVE_REDIRECT_URL",
		"AETHER_GDRIVE_FOLDER_ID",
		"AETHER_GDRIVE_REFRESH_TOKEN",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("AETHER_SHUTDOWN_TIMEOUT", "eventually")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "AETHER_SHUTDOWN_TIMEOUT") {
		t.Fatalf("Load() error = %q, want variable name", err)
	}
}

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	t.Setenv("AETHER_SHUTDOWN_TIMEOUT", "0s")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("Load() error = %q, want positive-duration guidance", err)
	}
}
