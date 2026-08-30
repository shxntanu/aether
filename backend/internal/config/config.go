package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddress     string
	ShutdownTimeout time.Duration
	DatabaseURL     string
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
