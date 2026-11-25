package api_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/api"
)

type echoInput struct {
	Label string `json:"label" jsonschema:"Label to echo"`
}

func connectServer(t *testing.T, srv *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverT, clientT := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "toolkit-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func listToolNames(t *testing.T, session *mcp.ClientSession) map[string]*mcp.Tool {
	t.Helper()
	res, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	out := make(map[string]*mcp.Tool, len(res.Tools))
	for _, tool := range res.Tools {
		out[tool.Name] = tool
	}
	return out
}

func TestResolveIdentityThroughFacade(t *testing.T) {
	got := api.ResolveIdentity(api.Identity{})
	if got.Name != "mayamcp" || got.Title != "MayaMCP" || got.Version != "dev" {
		t.Fatalf("defaults = %+v", got)
	}
	got = api.ResolveIdentity(api.Identity{Name: "consumer", Title: "Consumer", Version: "v9"})
	if got.Name != "consumer" || got.Title != "Consumer" || got.Version != "v9" {
		t.Fatalf("overrides = %+v", got)
	}
}

func TestNewLoggerThroughFacade(t *testing.T) {
	var buf bytes.Buffer
	logger := api.NewLogger(api.LogOptions{Writer: &buf, Level: slog.LevelWarn, JSON: true})
	logger.Info("hidden")
	logger.Warn("shown")
	line := buf.String()
	if strings.Contains(line, "hidden") {
		t.Errorf("Info record passed the level guard: %q", line)
	}
	if !strings.Contains(line, `"msg":"shown"`) {
		t.Errorf("JSON handler not in use: %q", line)
	}
}

func TestRegistrationHelpersThroughFacade(t *testing.T) {
	srv := api.NewWithOptions(testHost{"toolkit-server", "v1"}, api.Options{
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "text_only", Description: "Text result."},
				func(_ context.Context, _ *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Label}}}, in.Label, nil
				})
			api.AddUntypedTool(s, &mcp.Tool{Name: "untyped", Description: "Structured result."},
				func(_ context.Context, _ *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Label}}}, in.Label, nil
				})
		},
	})

	session := connectServer(t, srv)
	tools := listToolNames(t, session)
	if len(tools) != 2 {
		t.Fatalf("tools = %v, want exactly the two registered tools", tools)
	}
	for _, name := range []string{"text_only", "untyped"} {
		tool, ok := tools[name]
		if !ok {
			t.Fatalf("tool %q missing", name)
		}
		if tool.OutputSchema != nil {
			t.Errorf("tool %q has an outputSchema", name)
		}
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "untyped", Arguments: map[string]any{"label": "hello"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.StructuredContent == nil {
		t.Error("AddUntypedTool dropped structuredContent")
	}
	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "text_only", Arguments: map[string]any{"label": "hello"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.StructuredContent != nil {
		t.Error("AddTextTool leaked structuredContent")
	}
}

func TestRegisterReplacesStarterTools(t *testing.T) {
	srv := api.NewWithOptions(testHost{"custom", "v1"}, api.Options{
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "custom_tool", Description: "Consumer tool."},
				func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "", nil
				})
		},
	})
	tools := listToolNames(t, connectServer(t, srv))
	if len(tools) != 1 || tools["custom_tool"] == nil {
		t.Fatalf("tools = %v, want only custom_tool", tools)
	}
}

func TestNewCoreHooks(t *testing.T) {
	var order []string
	var buf bytes.Buffer
	core := api.NewCore(api.Config{
		Name:      "hooks-server",
		Version:   "v1",
		LogWriter: &buf,
		Register: func(s *mcp.Server) {
			order = append(order, "register")
			api.AddTextTool(s, &mcp.Tool{Name: "custom_tool", Description: "Consumer tool."},
				func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "", nil
				})
		},
		RegisterResources: func(s *mcp.Server) {
			order = append(order, "resources")
			s.AddResource(&mcp.Resource{Name: "notes", URI: "memo://notes"},
				func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
					return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "note"}}}, nil
				})
		},
		Middleware: []mcp.Middleware{
			func(next mcp.MethodHandler) mcp.MethodHandler {
				return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
					order = append(order, "middleware:"+method)
					return next(ctx, method, req)
				}
			},
		},
	})

	session := connectServer(t, core.Server)
	tools := listToolNames(t, session)
	if len(tools) != 1 || tools["custom_tool"] == nil {
		t.Fatalf("tools = %v, want only custom_tool", tools)
	}
	res, err := session.ListResources(context.Background(), &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(res.Resources) != 1 || res.Resources[0].URI != "memo://notes" {
		t.Fatalf("resources = %+v", res.Resources)
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "custom_tool"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !strings.Contains(strings.Join(order, ","), "middleware:tools/call") {
		t.Errorf("custom middleware did not run: %v", order)
	}
	if !strings.Contains(buf.String(), api.ActivityPrefix+"custom_tool") {
		t.Errorf("activity middleware missing from log: %q", buf.String())
	}
}

func TestActivityMiddlewareThroughFacade(t *testing.T) {
	var buf bytes.Buffer
	srv := mcp.NewServer(&mcp.Implementation{Name: "standalone", Version: "v1"}, nil)
	api.AddTextTool(srv, &mcp.Tool{Name: "standalone_tool", Description: "Consumer-owned server."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "", nil
		})
	srv.AddReceivingMiddleware(api.ActivityMiddleware(
		api.NewLogger(api.LogOptions{Writer: &buf}), nil,
	))

	session := connectServer(t, srv)
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "standalone_tool"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !strings.Contains(buf.String(), api.ActivityPrefix+"standalone_tool") {
		t.Errorf("activity line missing: %q", buf.String())
	}
}
