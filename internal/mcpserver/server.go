// Package mcpserver is the transport-agnostic MCP core of MayaMCP. It knows
// nothing about HTTP, stdio, auth or a UI: it builds an *mcp.Server from a
// small domain contract and a set of options.
package mcpserver

import (
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// DefaultName and DefaultVersion are the identity defaults shared with the
	// app package. Keeping them here (the lower layer) avoids an import cycle
	// and makes it impossible for the two layers to drift apart.
	DefaultName    = "mayamcp"
	DefaultVersion = "dev"

	// defaultName is kept as the package-local spelling used by helpers.
	defaultName  = DefaultName
	defaultTitle = "MayaMCP"
)

const defaultInstructions = `You are connected to %s, a small MCP starter server.

Use the available tools to answer requests. When adapting this template, replace
the starter tools with your domain and provide explicit Instructions describing
them; the transport and server wiring do not need to change.`

// Options configures identity and optional activity logging. Defaults keep
// the template usable out of the box; a consumer can override any of them
// without editing this package.
type Options struct {
	Name         string
	Title        string
	Instructions string
	// Version overrides the version argument and host.Version(). Empty keeps
	// the resolution order documented on New.
	Version string
	// TargetExtractor supplies domain-specific activity-log targets. Its
	// output is always sanitized with cleanTarget by the middleware.
	TargetExtractor TargetExtractor
	Logger          *slog.Logger
}

// New builds an MCP server for host. The same server can be served over stdio
// or over Streamable HTTP by the caller.
//
// The server identity is resolved exactly once. Names come from opts.Name,
// then host.Name(), then DefaultName. Versions come from opts.Version, then
// the version argument, then host.Version(), then DefaultVersion. The resolved
// values feed both initialize.serverInfo and describe_server.
//
// A nil host is safe: New substitutes fallbackHost, so describe_server reports
// the resolved identity instead of panicking at call time.
func New(host Host, version string, opts Options) *mcp.Server {
	name := opts.Name
	if name == "" && host != nil {
		name = host.Name()
	}
	if name == "" {
		name = DefaultName
	}
	resolvedVersion := opts.Version
	if resolvedVersion == "" {
		resolvedVersion = version
	}
	if resolvedVersion == "" && host != nil {
		resolvedVersion = host.Version()
	}
	if resolvedVersion == "" {
		resolvedVersion = DefaultVersion
	}
	title := opts.Title
	if title == "" {
		title = defaultTitle
	}
	instructions := opts.Instructions
	if instructions == "" {
		instructions = fmt.Sprintf(defaultInstructions, name)
	}

	// One identity for every observer: describe_server receives the same
	// resolved values the MCP handshake advertises. The consumer's Host is
	// never wrapped, so extending it with domain methods stays safe.
	s := mcp.NewServer(
		&mcp.Implementation{Name: name, Title: title, Version: resolvedVersion},
		&mcp.ServerOptions{Instructions: instructions},
	)
	addPing(s)
	addDescribeServer(s, name, resolvedVersion)
	if opts.Logger != nil {
		s.AddReceivingMiddleware(activityMiddleware(opts.Logger, opts.TargetExtractor))
	}
	return s
}
