package mcpserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Identity is the consumer-supplied subset of server metadata. Every field is
// optional; missing values fall back to the package defaults so identity can
// never be empty at the MCP handshake.
type Identity struct {
	Name    string
	Title   string
	Version string
}

// ResolveIdentity produces the single Implementation advertised by the MCP
// handshake. Consumers that call mcp.NewServer themselves can use this helper
// without going through New, which is the level-B toolkit path.
func ResolveIdentity(in Identity) mcp.Implementation {
	name := in.Name
	if name == "" {
		name = DefaultName
	}
	title := in.Title
	if title == "" {
		title = defaultTitle
	}
	version := in.Version
	if version == "" {
		version = DefaultVersion
	}
	return mcp.Implementation{Name: name, Title: title, Version: version}
}
