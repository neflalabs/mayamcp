// Command mayamcp serves the MCP core over stdio (default) or Streamable HTTP.
// Authentication and TLS are left to the embedding application or proxy:
// the template deliberately keeps the transport layer thin.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/app"
	"mayamcp/internal/mcpserver"
)

// defaultVersion is overridable at build time with -ldflags
// "-X main.defaultVersion=v2026.11.2".
var defaultVersion = mcpserver.DefaultVersion

func main() {
	transport := flag.String("transport", "stdio", "transport: stdio (default) or http")
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address (http transport only)")
	name := flag.String("name", mcpserver.DefaultName, "server identity name")
	version := flag.String("version", defaultVersion, "server version")
	flag.Parse()

	core := app.NewCore(app.Config{Name: *name, Version: *version})

	switch *transport {
	case "stdio":
		if err := serveStdio(core); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "http":
		if err := serveHTTP(core, *addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown transport %q (want stdio or http)\n", *transport)
		os.Exit(2)
	}
}

// serveStdio runs the MCP protocol over stdin/stdout until the parent closes
// the pipe or the process receives SIGINT/SIGTERM. stdout is reserved for the
// protocol; the logger writes to stderr.
func serveStdio(core *app.Core) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := core.Server.Run(ctx, &mcp.StdioTransport{})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// newHTTPServer builds the HTTP server with bounded read and idle timeouts.
// WriteTimeout is deliberately left at zero: Streamable HTTP uses long-lived
// SSE response streams, and a write deadline would terminate those streams.
func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// maxRequestBodyBytes bounds MCP POST bodies. Streamable HTTP requests are
// small JSON documents; an unbounded body would only waste memory.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// limitBody wraps h with a per-request body limit.
func limitBody(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		h.ServeHTTP(w, r)
	})
}

// runHTTP serves until ctx is done or ListenAndServe fails, then shuts down
// gracefully so in-flight requests can finish. Kept separate from signal
// wiring so shutdown can be tested without sending a signal to the test process.
func runHTTP(ctx context.Context, srv *http.Server, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down HTTP server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newMCPHandler mounts the Streamable HTTP endpoint at / and a minimal
// liveness endpoint at /healthz. Auth/TLS remain the embedding app's job.
func newMCPHandler(core *app.Core) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return core.Server },
		&mcp.StreamableHTTPOptions{Logger: core.Logger, Stateless: true},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	mux.Handle("/", limitBody(handler))
	return mux
}

// warnIfNonLoopback flags an exposed bind address: this template ships no
// auth or TLS, so the operator must provide them at the proxy layer.
func warnIfNonLoopback(addr string, logger *slog.Logger) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return
	}
	logger.Warn("non-loopback bind without built-in auth/TLS", "addr", addr)
}

// serveHTTP runs the HTTP transport until a fatal error or SIGINT/SIGTERM,
// then shuts the server down gracefully so in-flight requests can finish.
func serveHTTP(core *app.Core, addr string) error {
	srv := newHTTPServer(addr, newMCPHandler(core))
	warnIfNonLoopback(addr, core.Logger)

	// stdout stays reserved for MCP; signal handling and logging never use it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	core.Logger.Info("serving MCP over HTTP", "addr", addr)
	return runHTTP(ctx, srv, core.Logger)
}
