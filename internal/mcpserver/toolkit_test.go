package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResolveIdentityDefaultsAndOverrides(t *testing.T) {
	cases := []struct {
		name      string
		in        Identity
		wantName  string
		wantTitle string
		wantVer   string
	}{
		{"empty", Identity{}, DefaultName, defaultTitle, DefaultVersion},
		{"name only", Identity{Name: "n"}, "n", defaultTitle, DefaultVersion},
		{"title only", Identity{Title: "T"}, DefaultName, "T", DefaultVersion},
		{"version only", Identity{Version: "v2"}, DefaultName, defaultTitle, "v2"},
		{"all", Identity{Name: "n", Title: "T", Version: "v2"}, "n", "T", "v2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveIdentity(tc.in)
			if got.Name != tc.wantName || got.Title != tc.wantTitle || got.Version != tc.wantVer {
				t.Fatalf("ResolveIdentity(%+v) = %+v", tc.in, got)
			}
		})
	}
}

func TestNewLoggerBranches(t *testing.T) {
	existing := slog.New(slog.NewTextHandler(io.Discard, nil))
	if got := NewLogger(LogOptions{Logger: existing}); got != existing {
		t.Fatal("supplied Logger was replaced")
	}

	var jsonBuf bytes.Buffer
	jsonLogger := NewLogger(LogOptions{Writer: &jsonBuf, JSON: true, Level: slog.LevelWarn})
	jsonLogger.Info("hidden")
	jsonLogger.Warn("shown")
	out := jsonBuf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("Info passed the level guard: %q", out)
	}
	if !strings.Contains(out, `"msg":"shown"`) {
		t.Errorf("JSON handler not in use: %q", out)
	}

	var textBuf bytes.Buffer
	textLogger := NewLogger(LogOptions{Writer: &textBuf})
	textLogger.Info("plain")
	if !strings.Contains(textBuf.String(), "plain") {
		t.Errorf("text handler not in use: %q", textBuf.String())
	}

	// No writer and no logger: loggers are built against stderr.
	_ = NewLogger(LogOptions{})
}

func TestAddUntypedToolDirect(t *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "register-test", Version: "v1"}, nil)
	AddUntypedTool(srv, &mcp.Tool{Name: "confirm", Description: "Confirm."},
		func(_ context.Context, _ *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, string, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "ok", nil
		})
	AddUntypedTool(srv, &mcp.Tool{Name: "boom", Description: "Fail."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
			return nil, "", errors.New("boom")
		})

	session := identitySession(t, srv)
	res, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(res.Tools))
	}
	for _, tool := range res.Tools {
		if tool.OutputSchema != nil {
			t.Errorf("tool %q has an outputSchema", tool.Name)
		}
	}
	call, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "confirm"})
	if err != nil {
		t.Fatalf("CallTool confirm: %v", err)
	}
	if call.StructuredContent == nil {
		t.Error("structuredContent was dropped")
	}
	failed, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "boom"})
	if err == nil && !failed.IsError {
		t.Error("handler error was not propagated")
	}
}

func TestActivityMiddlewareDirect(t *testing.T) {
	var buf bytes.Buffer
	srv := mcp.NewServer(&mcp.Implementation{Name: "middleware-test", Version: "v1"}, nil)
	AddTextTool(srv, &mcp.Tool{Name: "probe", Description: "Probe."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "", nil
		})
	extractor := func(_ string, _ json.RawMessage) string { return "chosen-target" }
	srv.AddReceivingMiddleware(ActivityMiddleware(
		NewLogger(LogOptions{Writer: &buf}), extractor,
	))

	session := identitySession(t, srv)
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "probe"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !strings.Contains(buf.String(), ActivityPrefix+"probe chosen-target") {
		t.Errorf("activity line missing: %q", buf.String())
	}
}

func TestNewHookBranchesDirect(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogOptions{Writer: &buf})
	called := 0
	srv := New(nil, "", Options{
		Name:   "hook-server",
		Logger: logger,
		Register: func(s *mcp.Server) {
			called++
			AddTextTool(s, &mcp.Tool{Name: "custom", Description: "Custom."},
				func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, string, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, "", nil
				})
		},
		RegisterResources: func(s *mcp.Server) {
			called++
			s.AddResource(&mcp.Resource{Name: "notes", URI: "memo://notes"},
				func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
					return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "note"}}}, nil
				})
		},
		Middleware: []mcp.Middleware{
			func(next mcp.MethodHandler) mcp.MethodHandler {
				return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
					called++
					return next(ctx, method, req)
				}
			},
		},
	})

	session := identitySession(t, srv)
	tools, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "custom" {
		t.Fatalf("tools = %+v, want only custom", tools.Tools)
	}
	resources, err := session.ListResources(context.Background(), &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources.Resources))
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "custom"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if called < 3 {
		t.Errorf("hooks called only %d times", called)
	}
	if !strings.Contains(buf.String(), ActivityPrefix+"custom") {
		t.Errorf("activity middleware did not run: %q", buf.String())
	}

	// DisableActivityMiddleware keeps identity and hooks but skips logging.
	var quietBuf bytes.Buffer
	quiet := New(nil, "", Options{
		Name:                      "quiet-server",
		Logger:                    NewLogger(LogOptions{Writer: &quietBuf}),
		DisableActivityMiddleware: true,
	})
	quietSession := identitySession(t, quiet)
	if _, err := quietSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "ping"}); err != nil {
		t.Fatalf("CallTool ping: %v", err)
	}
	if quietBuf.Len() != 0 {
		t.Errorf("activity log written despite DisableActivityMiddleware: %q", quietBuf.String())
	}
}
