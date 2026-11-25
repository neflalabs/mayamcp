package mcpserver

// Small helpers shared by the tool files: text results and annotation presets.

import "github.com/modelcontextprotocol/go-sdk/mcp"

// textResult wraps text as the human-readable part of a tool result.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// readOnly marks a tool that never changes state.
func readOnly() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true}
}
