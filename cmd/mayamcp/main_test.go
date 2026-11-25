package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/app"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewHTTPServer(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	srv := newHTTPServer("127.0.0.1:8080", handler)

	if srv.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want %q", srv.Addr, "127.0.0.1:8080")
	}
	if srv.Handler == nil {
		t.Fatal("Handler is nil")
	}
	if got, want := reflect.ValueOf(srv.Handler).Pointer(), reflect.ValueOf(handler).Pointer(); got != want {
		t.Errorf("Handler does not match the supplied handler")
	}
	if srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout = %v, want > 0", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Errorf("ReadTimeout = %v, want > 0", srv.ReadTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Errorf("IdleTimeout = %v, want > 0", srv.IdleTimeout)
	}
	// Streamable HTTP uses long-lived SSE streams; a write timeout would cut them off.
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v, want 0", srv.WriteTimeout)
	}
}

func TestRunHTTPShutdownOnContextDone(t *testing.T) {
	shutdownCalled := make(chan struct{})
	srv := newHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.RegisterOnShutdown(func() { close(shutdownCalled) })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runHTTP(ctx, srv, testLogger()) }()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runHTTP: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runHTTP did not return after context cancellation")
	}
	select {
	case <-shutdownCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("srv.Shutdown was not called")
	}
}

func TestRunHTTPReturnsListenError(t *testing.T) {
	srv := newHTTPServer("not a valid address", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := runHTTP(context.Background(), srv, testLogger()); err == nil {
		t.Fatal("runHTTP returned nil for an invalid listen address")
	}
}

func TestLimitBody(t *testing.T) {
	handler := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("small body accepted", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ok":true}`))
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("oversized body rejected", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1)))
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rr.Code)
		}
	})
}

func TestHTTPHandlerHealthAndMCPRoundTrip(t *testing.T) {
	core := app.NewCore(app.Config{Name: "http-test", Version: "v1", LogWriter: io.Discard})
	srv := httptest.NewServer(newMCPHandler(core))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200", resp.StatusCode)
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:             srv.URL + "/",
		HTTPClient:           srv.Client(),
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "http-test-client", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("CallTool ping: %v", err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("ping content blocks = %d, want 1", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok || text.Text != "pong" {
		t.Fatalf("ping content = %#v, want pong", res.Content[0])
	}
}

func TestWarnIfNonLoopback(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	warnIfNonLoopback("127.0.0.1:8080", logger)
	if strings.Contains(buf.String(), "non-loopback") {
		t.Errorf("loopback address produced a warning: %q", buf.String())
	}

	warnIfNonLoopback("0.0.0.0:8080", logger)
	if !strings.Contains(buf.String(), "non-loopback") {
		t.Errorf("non-loopback address produced no warning: %q", buf.String())
	}
}
