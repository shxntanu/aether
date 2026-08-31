package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shxntanu/aether/backend/internal/catalog/postgres"
	"github.com/shxntanu/aether/backend/internal/config"
	"github.com/shxntanu/aether/backend/internal/httpapi"
	"github.com/shxntanu/aether/backend/internal/identity"
	"github.com/shxntanu/aether/backend/internal/server"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	listener, err := net.Listen("tcp", settings.HTTPAddress)
	if err != nil {
		log.Fatalf("listen on %s: %v", settings.HTTPAddress, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	router := httpapi.NewRouter()
	if settings.GoogleClientID != "" {
		store, err := postgres.Open(ctx, settings.DatabaseURL)
		if err != nil {
			log.Fatalf("open catalog: %v", err)
		}
		defer func() { _ = store.Close() }()
		provider, err := identity.NewGoogleProvider(ctx, settings.GoogleClientID, settings.GoogleClientSecret, settings.GoogleRedirectURL)
		if err != nil {
			log.Fatalf("configure Google OIDC: %v", err)
		}
		identityService := identity.NewService(store, time.Now, newSecret)
		if _, err := identityService.BootstrapAdmin(ctx, settings.BootstrapAdminEmail); err != nil {
			log.Fatalf("bootstrap administrator: %v", err)
		}
		oidcService := identity.NewOIDCService(store, provider, time.Now, newSecret)
		router = httpapi.NewRouter(httpapi.Options{Identity: identityService, OIDC: oidcService, SecureCookies: settings.SecureCookies})
	}
	log.Printf("Aether API listening on %s", listener.Addr())
	if err := server.Serve(ctx, listener, router, settings.ShutdownTimeout); err != nil {
		log.Fatalf("serve HTTP: %v", err)
	}
}

func newSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("secure random source unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
