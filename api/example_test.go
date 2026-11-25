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

// ExampleNewCore shows how a consumer assembles identity, hooks and logging
// without starting a listener.
func ExampleNewCore() {
	core := api.NewCore(api.Config{
		Name:    "example-core",
		Version: "v1",
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "hello", Description: "Say hello."},
				func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hello"}}}, "", nil
				})
		},
	})
	fmt.Println(core.Host.Name(), core.Host.Version())
	// Output: example-core v1
}

// ExampleAddTextTool shows the registration helper that omits outputSchema
// for text-heavy results.
func ExampleAddTextTool() {
	type args struct {
		Label string `json:"label" jsonschema:"Label to echo"`
	}
	srv := api.NewWithOptions(testHost{"text-server", "v1"}, api.Options{
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "echo", Description: "Echo a label."},
				func(_ context.Context, _ *mcp.CallToolRequest, in args) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Label}}}, in.Label, nil
				})
		},
	})
	n, err := api.CountTools(context.Background(), srv)
	if err != nil {
		panic(err)
	}
	fmt.Println(n)
	// Output: 1
}
