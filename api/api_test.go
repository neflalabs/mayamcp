package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/api"
)

// testHost implements api.Host, demonstrating that a consumer implementing
// only the public contract can build a server.
type testHost struct {
	name    string
	version string
}

func (h testHost) Name() string    { return h.name }
func (h testHost) Version() string { return h.version }

// TestAliases verifies that the public alias types are usable by a consumer
// that imports only github.com/neflalabs/mayamcp/api.
func TestAliases(t *testing.T) {
	var _ api.Host = testHost{}

	var cfg api.Config
	cfg.Name = "alias-server"
	cfg.Version = "v2"
	var opts api.Options
	opts.Name = "alias-name"

	core := api.NewCore(cfg)
	if core == nil {
		t.Fatal("NewCore returned nil")
	}
	if _, ok := any(core).(*api.Core); !ok {
		t.Fatalf("core type = %T, want *api.Core", core)
	}

	srv := api.New(testHost{"alias-server", "v2"}, "v2", opts)
	if srv == nil {
		t.Fatal("New returned nil server")
	}
}

// TestNewCoreHostIdentity verifies NewCore returns a populated Core whose host
// reflects the supplied config.
func TestNewCoreHostIdentity(t *testing.T) {
	core := api.NewCore(api.Config{Name: "identity-server", Version: "v3"})
	if core == nil {
		t.Fatal("NewCore returned nil")
	}
	if core.Host == nil {
		t.Fatal("Core.Host is nil")
	}
	if got := core.Host.Name(); got != "identity-server" {
		t.Errorf("Host.Name() = %q, want %q", got, "identity-server")
	}
	if got := core.Host.Version(); got != "v3" {
		t.Errorf("Host.Version() = %q, want %q", got, "v3")
	}
	if core.Server == nil {
		t.Fatal("Core.Server is nil")
	}
	if core.Logger == nil {
		t.Fatal("Core.Logger is nil")
	}
}

// TestNewServerOverInMemoryTransport verifies that a server built through the
// façade can be connected to and have its tools listed.
func TestNewServerOverInMemoryTransport(t *testing.T) {
	srv := api.New(testHost{"memory-server", "v1"}, "v1", api.Options{})
	if srv == nil {
		t.Fatal("New returned nil server")
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatal("ListTools returned no tools")
	}

	seen := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		seen[tool.Name] = true
	}
	for _, want := range []string{"ping", "describe_server"} {
		if !seen[want] {
			t.Errorf("tool %q not listed; got %v", want, seen)
		}
	}
}

func TestTargetExtractorThroughFacade(t *testing.T) {
	var extractor api.TargetExtractor = func(_ string, _ json.RawMessage) string { return "custom-target" }
	opts := api.Options{TargetExtractor: extractor}
	if opts.TargetExtractor == nil {
		t.Fatal("TargetExtractor did not survive the facade alias")
	}
	_ = api.New(testHost{"extractor-server", "v1"}, "v1", opts)
}

func TestActivityHelpersThroughFacade(t *testing.T) {
	if !strings.HasPrefix(api.ActivityLine("ping", "", true, "", time.Millisecond), "ping ") {
		t.Fatalf("unexpected activity line: %q", api.ActivityLine("ping", "", true, "", time.Millisecond))
	}
	n, err := api.CountTools(context.Background(), api.New(testHost{"count-server", "v1"}, "v1", api.Options{}))
	if err != nil {
		t.Fatalf("CountTools: %v", err)
	}
	if n < 1 {
		t.Errorf("CountTools() = %d, want at least one tool", n)
	}
}
