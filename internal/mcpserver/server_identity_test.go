package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type identityTestHost struct {
	name    string
	version string
}

func (h identityTestHost) Name() string    { return h.name }
func (h identityTestHost) Version() string { return h.version }

func identitySession(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	serverSession, err := s.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "identity-test-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// assertResolvedIdentity checks that the MCP handshake and describe_server
// report exactly the same name and version.
func assertResolvedIdentity(t *testing.T, s *mcp.Server, wantName, wantVersion string) {
	t.Helper()
	session := identitySession(t, s)

	info := session.InitializeResult().ServerInfo
	if info == nil {
		t.Fatal("initialize.serverInfo is nil")
	}
	if info.Name != wantName || info.Version != wantVersion {
		t.Errorf("serverInfo = %+v, want name=%q version=%q", info, wantName, wantVersion)
	}
	if instructions := session.InitializeResult().Instructions; !strings.Contains(instructions, wantName) {
		t.Errorf("instructions = %q, want it to mention %q", instructions, wantName)
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "describe_server"})
	if err != nil {
		t.Fatalf("CallTool(describe_server): %v", err)
	}
	if res.IsError {
		t.Fatalf("describe_server returned an error result: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structuredContent: %v", err)
	}
	var described struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &described); err != nil {
		t.Fatalf("unmarshal structuredContent: %v", err)
	}
	if described.Name != info.Name || described.Version != info.Version {
		t.Errorf("describe_server = %+v, serverInfo = %+v: identity diverged", described, info)
	}
	if described.Name != wantName || described.Version != wantVersion {
		t.Errorf("describe_server = %+v, want name=%q version=%q", described, wantName, wantVersion)
	}
}

func TestNewEmptyVersionDefaults(t *testing.T) {
	host := identityTestHost{name: "host-name", version: "host-version"}
	assertResolvedIdentity(t, New(host, "", Options{}), "host-name", "host-version")
}

func TestNewOptNameOverridesHost(t *testing.T) {
	host := identityTestHost{name: "host-name", version: "host-version"}
	assertResolvedIdentity(t, New(host, "v9", Options{Name: "opts-name"}), "opts-name", "v9")
}

func TestNewOptVersionHasHighestPriority(t *testing.T) {
	host := identityTestHost{name: "host-name", version: "host-version"}
	assertResolvedIdentity(t, New(host, "param-version", Options{Version: "opts-version"}), "host-name", "opts-version")
}

type extendedHost interface {
	Host
	Domain() string
}

type domainHost struct {
	identityTestHost
}

func (domainHost) Domain() string { return "d" }

func TestExtendedHostIsAcceptedUnwrapped(t *testing.T) {
	var h extendedHost = domainHost{identityTestHost{name: "n", version: "v"}}
	if got := h.Domain(); got != "d" {
		t.Fatalf("Domain() = %q, want d", got)
	}
	assertResolvedIdentity(t, New(h, "", Options{}), "n", "v")
}

func TestNewNilHostUsesSafeFallback(t *testing.T) {
	t.Run("all defaults", func(t *testing.T) {
		assertResolvedIdentity(t, New(nil, "", Options{}), DefaultName, DefaultVersion)
	})
	t.Run("configured identity", func(t *testing.T) {
		assertResolvedIdentity(t, New(nil, "v2", Options{Name: "configured"}), "configured", "v2")
	})
	t.Run("host with empty name", func(t *testing.T) {
		assertResolvedIdentity(t, New(identityTestHost{version: "x"}, "", Options{}), DefaultName, "x")
	})
}
