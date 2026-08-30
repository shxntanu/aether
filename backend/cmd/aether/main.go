package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/shxntanu/aether/backend/internal/config"
	"github.com/shxntanu/aether/backend/internal/httpapi"
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

	log.Printf("Aether API listening on %s", listener.Addr())
	if err := server.Serve(ctx, listener, httpapi.NewRouter(), settings.ShutdownTimeout); err != nil {
		log.Fatalf("serve HTTP: %v", err)
	}
}
