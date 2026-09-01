package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shxntanu/aether/backend/internal/audit"
	"github.com/shxntanu/aether/backend/internal/catalog/postgres"
	"github.com/shxntanu/aether/backend/internal/config"
	"github.com/shxntanu/aether/backend/internal/httpapi"
	"github.com/shxntanu/aether/backend/internal/identity"
	"github.com/shxntanu/aether/backend/internal/server"
	"github.com/shxntanu/aether/backend/internal/storage"
	"github.com/shxntanu/aether/backend/internal/storage/gdrive"
	localstorage "github.com/shxntanu/aether/backend/internal/storage/local"
	"github.com/shxntanu/aether/backend/internal/vault"
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
		auditRecorder := audit.NewRecorder(
			store,
			time.Now,
			func() (string, error) { return newSecret(), nil },
		)
		provider, err := identity.NewGoogleProvider(
			ctx,
			settings.GoogleClientID,
			settings.GoogleClientSecret,
			settings.GoogleRedirectURL,
		)
		if err != nil {
			log.Fatalf("configure Google OIDC: %v", err)
		}
		identityService := identity.NewService(store, time.Now, newSecret, auditRecorder)
		if _, err := identityService.BootstrapAdmin(ctx, settings.BootstrapAdminEmail); err != nil {
			log.Fatalf("bootstrap administrator: %v", err)
		}
		oidcService := identity.NewOIDCService(store, provider, time.Now, newSecret)
		objects, err := configureObjectStore(ctx, settings)
		if err != nil {
			log.Fatalf("configure %s object storage: %v", settings.StorageProvider, err)
		}
		vaultService := vault.NewService(store, objects, auditRecorder)
		router = httpapi.NewRouter(httpapi.Options{
			Identity:      identityService,
			OIDC:          oidcService,
			Vault:         vaultService,
			SecureCookies: settings.SecureCookies,
			Audit:         auditRecorder,
		})
	}
	log.Printf("Aether API listening on %s", listener.Addr())
	if err := server.Serve(ctx, listener, router, settings.ShutdownTimeout); err != nil {
		log.Fatalf("serve HTTP: %v", err)
	}
}

func configureObjectStore(
	ctx context.Context,
	settings config.Config,
) (storage.ObjectStore, error) {
	switch settings.StorageProvider {
	case "local":
		return localstorage.New(settings.LocalStoragePath)
	case "gdrive":
		return gdrive.New(ctx, gdrive.Config{
			ClientID:     settings.GoogleDriveClientID,
			ClientSecret: settings.GoogleDriveClientSecret,
			RedirectURL:  settings.GoogleDriveRedirectURL,
			FolderID:     settings.GoogleDriveFolderID,
			RefreshToken: settings.GoogleDriveRefreshToken,
		})
	default:
		return nil, fmt.Errorf("unsupported storage provider %q", settings.StorageProvider)
	}
}

func newSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("secure random source unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
