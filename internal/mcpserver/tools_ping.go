package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type pingArgs struct{}

// addPing registers a minimal liveness tool. It is the simplest example of
// addTextTool: content only, no outputSchema cost in tools/list.
func addPing(s *mcp.Server) {
	AddTextTool(s, &mcp.Tool{
		Name:        "ping",
		Description: "Return pong. Use it to verify the transport and the tool contract.",
		Annotations: readOnly(),
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ pingArgs) (*mcp.CallToolResult, any, error) {
		return textResult("pong"), nil, nil
	})
}
