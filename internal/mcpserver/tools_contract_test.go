package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/internal/mcpserver"
)

// Regenerate with: go test ./internal/mcpserver -run TestToolContract -args -update
var updateGolden = flag.Bool("update", false, "rewrite testdata/tools.golden.json")

const toolsGolden = "testdata/tools.golden.json"

type testHost struct{}

func (testHost) Name() string    { return "test-server" }
func (testHost) Version() string { return "v1" }

func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcpserver.New(testHost{}, "v1", mcpserver.Options{})
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func listTools(t *testing.T) []*mcp.Tool {
	t.Helper()
	var tools []*mcp.Tool
	session := newTestSession(t)
	params := &mcp.ListToolsParams{}
	for {
		res, err := session.ListTools(context.Background(), params)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == "" {
			break
		}
		params.Cursor = res.NextCursor
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	return tools
}

func TestToolContract(t *testing.T) {
	tools := listTools(t)
	got, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	got = append(got, '\n')

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(toolsGolden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(toolsGolden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d tools)", toolsGolden, len(tools))
		return
	}
	want, err := os.ReadFile(toolsGolden)
	if err != nil {
		t.Fatalf("read %s: %v (run with -args -update to create it)", toolsGolden, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("tool contract changed (%d tools); if intended, rerun with -args -update and review the diff", len(tools))
	}
}

func TestToolCallResults(t *testing.T) {
	session := newTestSession(t)
	ctx := context.Background()

	ping, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("CallTool ping: %v", err)
	}
	if len(ping.Content) != 1 {
		t.Fatalf("ping content blocks = %d, want 1", len(ping.Content))
	}
	text, ok := ping.Content[0].(*mcp.TextContent)
	if !ok || text.Text != "pong" {
		t.Fatalf("ping content = %#v, want text \"pong\"", ping.Content[0])
	}
	if ping.StructuredContent != nil {
		t.Errorf("ping structuredContent = %#v, want nil", ping.StructuredContent)
	}

	desc, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe_server"})
	if err != nil {
		t.Fatalf("CallTool describe_server: %v", err)
	}
	if len(desc.Content) != 1 {
		t.Fatalf("describe_server content blocks = %d, want 1", len(desc.Content))
	}
	text, ok = desc.Content[0].(*mcp.TextContent)
	if !ok || text.Text != "test-server v1" {
		t.Fatalf("describe_server content = %#v, want text \"test-server v1\"", desc.Content[0])
	}
	raw, err := json.Marshal(desc.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structuredContent: %v", err)
	}
	var out struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structuredContent: %v", err)
	}
	if out.Name != "test-server" || out.Version != "v1" {
		t.Errorf("describe_server structuredContent = %+v", out)
	}
}
