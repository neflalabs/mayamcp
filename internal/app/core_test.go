package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/internal/mcpserver"
)

func TestNewCoreWithoutTransport(t *testing.T) {
	core := NewCore(Config{Name: "test-server", Version: "v1", LogWriter: io.Discard})
	if core.Server == nil || core.Host == nil || core.Logger == nil {
		t.Fatal("core is missing server, host or logger")
	}
	if core.Host.Name() != "test-server" || core.Host.Version() != "v1" {
		t.Fatalf("host = %s %s", core.Host.Name(), core.Host.Version())
	}
}

func TestNewCoreDefaults(t *testing.T) {
	core := NewCore(Config{LogWriter: io.Discard})
	if core.Host.Name() != mcpserver.DefaultName || core.Host.Version() != mcpserver.DefaultVersion {
		t.Fatalf("defaults = %s %s", core.Host.Name(), core.Host.Version())
	}
}

// connectSession wires a Core.Server to a client over in-memory transports.
func connectSession(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	serverSession, err := s.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callToolCalls a tool and decodes its structuredContent into T.
func callTool[T any](t *testing.T, session *mcp.ClientSession, name string) (T, *mcp.CallToolResult) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%s) returned an error result: %+v", name, res.Content)
	}
	var out T
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structuredContent: %v", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structuredContent: %v", err)
	}
	return out, res
}

func TestNewCoreIdentityIsSingleSourceOfTruth(t *testing.T) {
	const (
		name         = "custom-server"
		version      = "v1.2.3"
		title        = "Custom Server Title"
		instructions = "Custom instructions for this server."
	)
	core := NewCore(Config{
		LogWriter:    io.Discard,
		Name:         name,
		Version:      version,
		Title:        title,
		Instructions: instructions,
	})
	session := connectSession(t, core.Server)

	serverInfo := session.InitializeResult().ServerInfo
	if serverInfo == nil {
		t.Fatal("initialize.serverInfo is nil")
	}
	if serverInfo.Name != name || serverInfo.Title != title || serverInfo.Version != version {
		t.Errorf("serverInfo = %+v, want name=%q title=%q version=%q", serverInfo, name, title, version)
	}
	if got := session.InitializeResult().Instructions; got != instructions {
		t.Errorf("instructions = %q, want %q", got, instructions)
	}

	// describe_server must report the same identity as the handshake.
	type described struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	got, res := callTool[described](t, session, "describe_server")
	if got.Name != serverInfo.Name || got.Version != serverInfo.Version {
		t.Errorf("describe_server = %+v, serverInfo = %+v: identity diverged", got, serverInfo)
	}
	var text string
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(*mcp.TextContent); ok {
			text = tc.Text
		}
	}
	if want := serverInfo.Name + " " + serverInfo.Version; text != want {
		t.Errorf("describe_server text = %q, want %q", text, want)
	}
}

func TestNewCoreDefaultsAreAdvertised(t *testing.T) {
	core := NewCore(Config{
		LogWriter: io.Discard,
		Title:     "Defaulted Title",
	})
	session := connectSession(t, core.Server)

	info := session.InitializeResult().ServerInfo
	if info == nil {
		t.Fatal("initialize.serverInfo is nil")
	}
	if info.Name != mcpserver.DefaultName || info.Version != mcpserver.DefaultVersion {
		t.Errorf("serverInfo = %+v, want default name=%q version=%q", info, mcpserver.DefaultName, mcpserver.DefaultVersion)
	}
	if info.Title != "Defaulted Title" {
		t.Errorf("title = %q, want %q", info.Title, "Defaulted Title")
	}
	instructions := session.InitializeResult().Instructions
	if !strings.Contains(instructions, mcpserver.DefaultName) {
		t.Errorf("default instructions = %q, want it to mention %q", instructions, mcpserver.DefaultName)
	}
}

// syncBuffer is a thread-safe sink for slog output produced by the server's
// middleware goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestNewCoreOptionsPassthrough verifies Config options through observable
// behavior over an in-memory session, without a white-box options snapshot.
func TestNewCoreOptionsPassthrough(t *testing.T) {
	var logs syncBuffer
	cfg := Config{
		Name:         "custom-server",
		Version:      "v1",
		Title:        "Custom Server Title",
		Instructions: "Custom instructions for this server.",
		LogWriter:    &logs,
	}
	core := NewCore(cfg)
	session := connectSession(t, core.Server)

	init := session.InitializeResult()
	if info := init.ServerInfo; info == nil || info.Name != cfg.Name || info.Title != cfg.Title || info.Version != cfg.Version {
		t.Errorf("serverInfo = %+v, want %s / %s / %s", info, cfg.Name, cfg.Title, cfg.Version)
	}
	if init.Instructions != cfg.Instructions {
		t.Errorf("instructions = %q, want %q", init.Instructions, cfg.Instructions)
	}
	callTool[struct{}](t, session, "ping")
	if !strings.Contains(logs.String(), "tool: ping") {
		t.Errorf("activity log missing from Config.LogWriter: %q", logs.String())
	}
}

func TestNewCoreUsesSuppliedLogger(t *testing.T) {
	var logs syncBuffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	core := NewCore(Config{Logger: logger})
	if core.Logger != logger {
		t.Fatal("supplied Logger was replaced")
	}
	session := connectSession(t, core.Server)
	callTool[struct{}](t, session, "ping")
	if !strings.Contains(logs.String(), "tool: ping") {
		t.Errorf("activity log missing from supplied Logger: %q", logs.String())
	}
}

func TestNewCoreLogJSONAndLevel(t *testing.T) {
	var logs syncBuffer
	core := NewCore(Config{
		LogWriter: &logs,
		LogJSON:   true,
		LogLevel:  slog.LevelWarn,
	})
	session := connectSession(t, core.Server)
	callTool[struct{}](t, session, "ping")
	if strings.Contains(logs.String(), "tool: ping") {
		t.Errorf("info-level activity was not filtered: %q", logs.String())
	}

	core.Logger.Warn("warn message")
	if !strings.Contains(logs.String(), `"level":"WARN"`) || !strings.Contains(logs.String(), "warn message") {
		t.Errorf("JSON warn record missing: %q", logs.String())
	}
}

func TestNewCoreLogWriterReplacesLogger(t *testing.T) {
	var supplied, writer syncBuffer
	core := NewCore(Config{
		Logger:    slog.New(slog.NewTextHandler(&supplied, nil)),
		LogWriter: &writer,
	})
	session := connectSession(t, core.Server)
	callTool[struct{}](t, session, "ping")

	if strings.Contains(supplied.String(), "tool: ") {
		t.Errorf("supplied Logger was used despite LogWriter: %q", supplied.String())
	}
	if !strings.Contains(writer.String(), "tool: ping") {
		t.Errorf("LogWriter received no activity log: %q", writer.String())
	}
}
