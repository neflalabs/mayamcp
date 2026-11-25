// Package consumer_test proves that an external Go module can build an MCP
// server using only github.com/neflalabs/mayamcp/api, without importing internal packages and
// without inheriting the template's starter tools.
package consumer_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/api"
)

type host struct{}

func (host) Name() string    { return "consumer-server" }
func (host) Version() string { return "v4.2.0" }

type input struct {
	Label string `json:"label" jsonschema:"Short label"`
}

func TestConsumerCanOwnLifecycle(t *testing.T) {
	var logs bytes.Buffer
	srv := api.NewWithOptions(host{}, api.Options{
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "echo", Description: "Echo a label."},
				func(_ context.Context, _ *mcp.CallToolRequest, in input) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Label}}}, in.Label, nil
				})
		},
		RegisterResources: func(s *mcp.Server) {
			s.AddResource(&mcp.Resource{Name: "notes", URI: "memo://notes"},
				func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
					return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "note"}}}, nil
				})
		},
		Middleware: []mcp.Middleware{
			func(next mcp.MethodHandler) mcp.MethodHandler {
				return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
					return next(ctx, method, req)
				}
			},
		},
	})

	serverT, clientT := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "consumer-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "echo" {
		t.Fatalf("tools = %+v, want only echo", tools.Tools)
	}
	res, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(res.Resources) != 1 {
		t.Fatalf("resources = %+v, want one resource", res.Resources)
	}

	// Attach the exported activity middleware to the consumer-owned server.
	srv.AddReceivingMiddleware(api.ActivityMiddleware(
		api.NewLogger(api.LogOptions{Writer: &logs, Level: slog.LevelInfo}), nil,
	))
	call, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"label": "Bearer secret-token"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	_ = call
	if !strings.Contains(logs.String(), api.ActivityPrefix+"echo") {
		t.Errorf("activity line missing: %q", logs.String())
	}
	if strings.Contains(logs.String(), "secret-token") {
		t.Errorf("secret leaked into activity log: %q", logs.String())
	}
}
