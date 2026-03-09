package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func newTestServer(handler http.Handler) (*http.Server, string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	return srv, "http://" + ln.Addr().String()
}

func TestGracefulShutdown_InFlightRequestCompletes(t *testing.T) {
	started := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	srv, addr := newTestServer(handler)

	result := make(chan error, 1)
	go func() {
		resp, err := http.Get(addr)
		if err != nil {
			result <- err
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		result <- nil
	}()

	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	if err := <-result; err != nil {
		t.Fatalf("in-flight request failed: %v", err)
	}
}

func TestGracefulShutdown_RejectsNewConnections(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv, addr := newTestServer(handler)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	_, err := http.Get(addr)
	if err == nil {
		t.Fatal("expected connection to fail after shutdown, but it succeeded")
	}
}
