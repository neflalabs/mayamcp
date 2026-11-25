package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type describeServerArgs struct{}

// describeServerOut is a small typed record, so plain mcp.AddTool keeps both
// the canonical text content and an outputSchema for programmatic clients.
type describeServerOut struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// addDescribeServer reports the identity resolved by New, so its answer always
// matches initialize.serverInfo.
func addDescribeServer(s *mcp.Server, name, version string) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_server",
		Description: "Return the server name and version from the domain layer.",
		Annotations: readOnly(),
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ describeServerArgs) (*mcp.CallToolResult, describeServerOut, error) {
		out := describeServerOut{Name: name, Version: version}
		return textResult(fmt.Sprintf("%s %s", out.Name, out.Version)), out, nil
	})
}
