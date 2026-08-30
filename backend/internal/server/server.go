package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func Serve(ctx context.Context, listener net.Listener, handler http.Handler, shutdownTimeout time.Duration) error {
	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
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
