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
	defaultName          = "mayamcp"
	defaultTitle         = "MayaMCP"
	defaultMetaNamespace = "mayamcp"
)

const defaultInstructions = `You are connected to %s, a small MCP starter server.

Use ping to check that the connection works, then describe_server to learn the
server name and version. Replace these tools with your domain when adapting
this template; the transport and server wiring do not need to change.`

// Options configures identity and optional activity logging. Defaults keep
// the template usable out of the box; a consumer can override any of them
// without editing this package.
type Options struct {
	Name          string
	Title         string
	Instructions  string
	MetaNamespace string
	Logger        *slog.Logger
}

// New builds an MCP server for host. The same server can be served over stdio
// or over Streamable HTTP by the caller.
func New(host Host, version string, opts Options) *mcp.Server {
	name, title, namespace := opts.Name, opts.Title, opts.MetaNamespace
	if name == "" {
		name = defaultName
	}
	if title == "" {
		title = defaultTitle
	}
	if namespace == "" {
		namespace = defaultMetaNamespace
	}
	instructions := opts.Instructions
	if instructions == "" {
		instructions = fmt.Sprintf(defaultInstructions, name)
	}

	s := mcp.NewServer(
		&mcp.Implementation{Name: name, Title: title, Version: version},
		&mcp.ServerOptions{Instructions: instructions},
	)
	addPing(s)
	addDescribeServer(s, host)
	if opts.Logger != nil {
		s.AddReceivingMiddleware(activityMiddleware(opts.Logger, namespace))
	}
	return s
}
