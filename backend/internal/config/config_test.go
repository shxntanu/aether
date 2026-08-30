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
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("AETHER_HTTP_ADDRESS", "127.0.0.1:9000")
	t.Setenv("AETHER_SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("AETHER_DATABASE_URL", "postgres://hosted.example/aether")

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
