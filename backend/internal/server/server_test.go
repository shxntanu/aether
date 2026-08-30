package server

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestServeStopsWhenContextIsCancelled(t *testing.T) {
	listener, client := newPipeListener(t)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), time.Second)
	}()

	if _, err := client.Write([]byte("GET / HTTP/1.1\r\nHost: aether.test\r\n\r\n")); err != nil {
		cancel()
		t.Fatalf("write request: %v", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(client), &http.Request{Method: http.MethodGet})
	if err != nil {
		cancel()
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		cancel()
		t.Fatalf("response status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	_ = response.Body.Close()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after context cancellation")
	}
}

type pipeListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func newPipeListener(t *testing.T) (*pipeListener, net.Conn) {
	t.Helper()
	serverConnection, clientConnection := net.Pipe()
	listener := &pipeListener{
		connections: make(chan net.Conn, 1),
		closed:      make(chan struct{}),
	}
	listener.connections <- serverConnection
	return listener, clientConnection
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case connection := <-l.connections:
		return connection, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return pipeAddress("memory")
}

type pipeAddress string

func (a pipeAddress) Network() string { return string(a) }
func (a pipeAddress) String() string  { return string(a) }
