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
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/app"
	"mayamcp/internal/mcpserver"
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

// TargetExtractor is the hook for domain-specific activity-log targets.
type TargetExtractor = mcpserver.TargetExtractor

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
