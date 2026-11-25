package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/internal/app"
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
	if srv.MaxHeaderBytes <= 0 || srv.MaxHeaderBytes > 1<<20 {
		t.Errorf("MaxHeaderBytes = %d, want a bounded positive value", srv.MaxHeaderBytes)
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

func TestParseLogLevel(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"Warn", slog.LevelWarn},
		{" warning ", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		got, err := parseLogLevel(tc.in)
		if err != nil {
			t.Errorf("parseLogLevel(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	for _, in := range []string{"", "trace", "verbose"} {
		if _, err := parseLogLevel(in); err == nil {
			t.Errorf("parseLogLevel(%q) returned nil error", in)
		}
	}
}

func TestServeHTTPInvalidAddr(t *testing.T) {
	core := app.NewCore(app.Config{Name: "bad-addr", Version: "v1", LogWriter: io.Discard})
	if err := serveHTTP(core, "not a valid address"); err == nil {
		t.Fatal("serveHTTP returned nil for an invalid listen address")
	}
}

func TestNewCoreFromFlags(t *testing.T) {
	core, err := newCore("flag-server", "v7", "warn", true)
	if err != nil {
		t.Fatalf("newCore: %v", err)
	}
	if core.Host.Name() != "flag-server" || core.Host.Version() != "v7" {
		t.Errorf("identity = %s %s", core.Host.Name(), core.Host.Version())
	}
	if _, err := newCore("a", "b", "trace", false); err == nil {
		t.Error("invalid log level was accepted")
	}
}

func TestWarnIfNonLoopbackWildcardAndIPv6(t *testing.T) {
	for _, addr := range []string{":8080", "[::]:8080"} {
		t.Run(addr, func(t *testing.T) {
			var buf bytes.Buffer
			warnIfNonLoopback(addr, slog.New(slog.NewTextHandler(&buf, nil)))
			if !strings.Contains(buf.String(), "non-loopback") {
				t.Errorf("wildcard %q produced no warning: %q", addr, buf.String())
			}
		})
	}

	for _, addr := range []string{"[::1]:8080", "127.0.0.2:8080", "LOCALHOST:8080", "localhost.:8080"} {
		t.Run(addr, func(t *testing.T) {
			var buf bytes.Buffer
			warnIfNonLoopback(addr, slog.New(slog.NewTextHandler(&buf, nil)))
			if strings.Contains(buf.String(), "non-loopback") {
				t.Errorf("loopback %q produced a warning: %q", addr, buf.String())
			}
		})
	}
}

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		stdioErr error
		httpErr  error
		wantCode int
		wantText string
		wantHTTP bool
	}{
		{name: "stdio success", wantCode: 0},
		{name: "http success", args: []string{"-transport=http", "-addr=127.0.0.1:0"}, wantCode: 0, wantHTTP: true},
		{name: "invalid log level", args: []string{"-log-level=trace"}, wantCode: 2, wantText: "invalid -log-level"},
		{name: "invalid transport", args: []string{"-transport=bogus"}, wantCode: 2, wantText: "unknown transport"},
		{name: "unknown flag", args: []string{"-not-a-flag"}, wantCode: 2, wantText: "not defined"},
		{name: "stdio failure", stdioErr: errors.New("stdio failed"), wantCode: 1, wantText: "stdio failed"},
		{name: "http failure", args: []string{"-transport=http"}, httpErr: errors.New("http failed"), wantCode: 1, wantText: "http failed", wantHTTP: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			stdioCalls, httpCalls := 0, 0
			got := run(tc.args, &stderr,
				func(*app.Core) error { stdioCalls++; return tc.stdioErr },
				func(*app.Core, string) error { httpCalls++; return tc.httpErr },
			)
			if got != tc.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", got, tc.wantCode, stderr.String())
			}
			if tc.wantText != "" && !strings.Contains(stderr.String(), tc.wantText) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tc.wantText)
			}
			if tc.wantHTTP {
				if httpCalls != 1 || stdioCalls != 0 {
					t.Errorf("calls: stdio=%d http=%d, want only http", stdioCalls, httpCalls)
				}
			} else if tc.wantCode == 0 || tc.name == "stdio failure" {
				if stdioCalls != 1 || httpCalls != 0 {
					t.Errorf("calls: stdio=%d http=%d, want only stdio", stdioCalls, httpCalls)
				}
			}
		})
	}
}

// stubTransport fails immediately so serveStdio can be tested without
// touching real stdin/stdout.
type stubTransport struct{ err error }

func (t stubTransport) Connect(context.Context) (mcp.Connection, error) {
	return nil, t.err
}

func TestServeStdioTransport(t *testing.T) {
	core := app.NewCore(app.Config{Name: "stdio-test", Version: "v1", LogWriter: io.Discard})
	if err := serveStdio(core, stubTransport{err: context.Canceled}); err != nil {
		t.Errorf("canceled transport = %v, want nil", err)
	}
	want := errors.New("connect failed")
	if err := serveStdio(core, stubTransport{err: want}); !errors.Is(err, want) {
		t.Errorf("failed transport = %v, want %v", err, want)
	}
}

func TestRunHTTPServerClosed(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = srv.Close()
	}()
	if err := runHTTP(context.Background(), srv, testLogger()); err != nil {
		t.Fatalf("runHTTP after Close = %v, want nil", err)
	}
}
