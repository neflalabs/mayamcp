package mcpserver

// Registration helpers that decide which result payloads a tool advertises.
// addTextTool keeps only the canonical text content, so text-heavy tools do
// not pay for an outputSchema in tools/list. addUntypedTool keeps the typed
// value as structuredContent but still omits outputSchema. Plain mcp.AddTool
// keeps both the text and an outputSchema for small typed records.

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler is the typed handler shape shared by the registration helpers.
// It is exported so consumers outside the module can use AddTextTool and
// AddUntypedTool through the api façade.
type Handler[In, Out any] func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)

// AddTextTool registers a tool with content only: no outputSchema and no
// structuredContent. The handler's typed value is dropped.
func AddTextTool[In, Out any](s *mcp.Server, t *mcp.Tool, h Handler[In, Out]) {
	mcp.AddTool(s, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		res, _, err := h(ctx, req, in)
		return res, nil, err
	})
}

// AddUntypedTool registers a tool without an outputSchema while still sending
// the handler's value as structuredContent. It suits short confirmations that
// programmatic clients read as JSON but that should not cost tools/list tokens.
func AddUntypedTool[In, Out any](s *mcp.Server, t *mcp.Tool, h Handler[In, Out]) {
	mcp.AddTool(s, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		res, out, err := h(ctx, req, in)
		if err != nil {
			return res, nil, err
		}
		return res, out, nil
	})
}
