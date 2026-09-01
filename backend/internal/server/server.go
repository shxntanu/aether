package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// Serve runs handler on listener with transport timeouts and graceful
// shutdown bound to ctx and shutdownTimeout.
func Serve(
	ctx context.Context,
	listener net.Listener,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	httpServer := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadTimeout:       11 * time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}

	watchCtx, stopWatching := context.WithCancel(ctx)
	defer stopWatching()

	shutdownResult := make(chan error, 1)
	go func() {
		<-watchCtx.Done()
		if ctx.Err() == nil {
			shutdownResult <- nil
			return
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		shutdownResult <- httpServer.Shutdown(shutdownCtx)
	}()

	err := httpServer.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	if ctx.Err() != nil {
		if shutdownErr := <-shutdownResult; err == nil {
			err = shutdownErr
		}
	}

	return err
}
