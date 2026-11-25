package mcpserver

// Registration helpers that decide which result payloads a tool advertises.
// addTextTool keeps only the canonical text content, so text-heavy tools do
// not pay for an outputSchema in tools/list. Plain mcp.AddTool keeps both the
// text and an outputSchema for small typed records.

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolHandler[In, Out any] func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)

// addTextTool registers a tool with content only: no outputSchema and no
// structuredContent.
func addTextTool[In, Out any](s *mcp.Server, t *mcp.Tool, h toolHandler[In, Out]) {
	mcp.AddTool(s, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		res, _, err := h(ctx, req, in)
		return res, nil, err
	})
}
