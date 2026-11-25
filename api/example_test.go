package api_test

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/api"
)

// ExampleNew_customTool shows how a consumer adds a parameterized tool without
// changing the built-in tool contract. It is verified by `go test ./...`.
func ExampleNew_customTool() {
	type echoArgs struct {
		Label string `json:"label" jsonschema:"Text to echo"`
	}

	srv := api.NewWithOptions(testHost{"example-server", "v1"}, api.Options{})
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "echo",
		Description: "Echo a short label.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in echoArgs) (*mcp.CallToolResult, any, error) {
		if in.Label == "" {
			// Domain/validation error: the model can see and correct it.
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "label is required"}},
			}, nil, nil
		}
		// Protocol errors use `return nil, err` instead.
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: in.Label}},
		}, nil, nil
	})

	serverT, clientT := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		panic(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "example-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"label": "hello"},
	})
	if err != nil {
		panic(err)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		panic("expected text content")
	}
	fmt.Println(text.Text)
	// Output: hello
}
