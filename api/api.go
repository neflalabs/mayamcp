// Package api is the public, importable façade of MayaMCP.
//
// The reusable implementation lives in internal/app and internal/mcpserver.
// Go's internal package rule prevents other modules from importing those
// packages directly, so this package re-exports a small, stable surface. It
// contains no logic of its own: it only aliases types and delegates to the
// internal packages, which keeps the façade in sync as the core evolves.
package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/internal/app"
	"github.com/neflalabs/mayamcp/internal/mcpserver"
)

// Config is the input to NewCore. It is an alias of app.Config, so fields
// added to the core config are available here without duplication.
type Config = app.Config

// Core is the transport-agnostic runtime returned by NewCore.
type Core = app.Core

// Host is the domain contract used by the template tools.
type Host = mcpserver.Host

// Options configures server identity and optional activity logging.
type Options = mcpserver.Options

// Identity is the metadata consumed by ResolveIdentity.
type Identity = mcpserver.Identity

// LogOptions configures a logger created by NewLogger.
type LogOptions = mcpserver.LogOptions

// TargetExtractor is the hook for domain-specific activity-log targets.
type TargetExtractor = mcpserver.TargetExtractor

// Handler is the typed handler shape accepted by AddTextTool and AddUntypedTool.
type Handler[In, Out any] func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)

// NewCore prepares an MCP server and its domain host without starting a
// listener or wiring a transport.
func NewCore(cfg Config) *Core {
	return app.NewCore(cfg)
}

// New builds an MCP server for host. The caller decides how to serve the
// returned server (stdio, Streamable HTTP, in-memory, ...).
func New(host Host, version string, opts Options) *mcp.Server {
	return mcpserver.New(host, version, opts)
}

// NewWithOptions builds a server without a separate version argument. The
// version comes from Options.Version, then host.Version(), then the default.
func NewWithOptions(host Host, opts Options) *mcp.Server {
	return mcpserver.New(host, "", opts)
}

// ResolveIdentity produces the server identity advertised at initialize. Use
// it when the consumer builds mcp.NewServer itself (level-B toolkit path).
func ResolveIdentity(in Identity) mcp.Implementation {
	return mcpserver.ResolveIdentity(in)
}

// NewLogger builds the standard slog-based logger used by the core. Level and
// formatting options only apply when no existing Logger is supplied.
func NewLogger(o LogOptions) *slog.Logger {
	return mcpserver.NewLogger(o)
}

// ActivityMiddleware is the log middleware used by the core. Consumers that
// own mcp.NewServer add it themselves with Server.AddReceivingMiddleware.
func ActivityMiddleware(logger *slog.Logger, extract TargetExtractor) mcp.Middleware {
	return mcpserver.ActivityMiddleware(logger, extract)
}

// ActivitySize is the measured byte/token size of a tool result.
type ActivitySize = mcpserver.ActivitySize

// MeasureActivity reports the exact byte counts and bytes/4 token estimates
// for a tool result. It is shared with consumers that own their middleware.
func MeasureActivity(res mcp.Result) (ActivitySize, bool) {
	return mcpserver.MeasureActivity(res)
}

// SanitizeTarget applies the core's printable, bounded target sanitizer.
func SanitizeTarget(s string) string {
	return mcpserver.SanitizeTarget(s)
}

// AddTextTool registers a tool with content only: no outputSchema and no
// structuredContent. Use it for text-heavy results such as files or logs.
func AddTextTool[In, Out any](s *mcp.Server, t *mcp.Tool, h Handler[In, Out]) {
	mcpserver.AddTextTool(s, t, mcpserver.Handler[In, Out](h))
}

// AddUntypedTool registers a tool with structuredContent but no outputSchema.
// Use it for short confirmations read by programmatic clients.
func AddUntypedTool[In, Out any](s *mcp.Server, t *mcp.Tool, h Handler[In, Out]) {
	mcpserver.AddUntypedTool(s, t, mcpserver.Handler[In, Out](h))
}

// ActivityPrefix is the message prefix a log consumer can use to recognise
// tool-activity records written by the core.
const ActivityPrefix = mcpserver.ActivityPrefix

// ActivityLine formats one activity entry for display or testing.
func ActivityLine(tool, target string, ok bool, detail string, d time.Duration) string {
	return mcpserver.ActivityLine(tool, target, ok, detail, d)
}

// CountTools reports how many tools s exposes. The caller should bound ctx.
func CountTools(ctx context.Context, s *mcp.Server) (int, error) {
	return mcpserver.CountTools(ctx, s)
}
