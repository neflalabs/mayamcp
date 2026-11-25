package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func assertSafeTarget(t *testing.T, got string) {
	t.Helper()
	if r := []rune(got); len(r) > maxTargetLen {
		t.Fatalf("target longer than %d runes: %q", maxTargetLen, got)
	}
	for _, r := range got {
		if unicode.IsControl(r) {
			t.Fatalf("control character in target: %q", got)
		}
	}
}

func FuzzSanitizeTarget(f *testing.F) {
	f.Add("simple")
	f.Add("  a\n\tb  ")
	f.Add("user:pass@host")
	f.Add(strings.Repeat("x", 200))
	f.Fuzz(func(t *testing.T, s string) {
		assertSafeTarget(t, SanitizeTarget(s))
	})
}

func FuzzActivityTarget(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"name":"a","path":"b"}`))
	f.Add([]byte(`{"from":"https://user:pass@host/x?token=abc","to":"old.go"}`))
	f.Add([]byte(`{"from":123,"to":null,"args":["x"]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		assertSafeTarget(t, activityTarget(json.RawMessage(raw)))
	})
}

func BenchmarkMeasureActivity(b *testing.B) {
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("x", 64<<10)}},
		StructuredContent: map[string]any{
			"ok": true, "label": strings.Repeat("y", 4<<10),
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := MeasureActivity(res); !ok {
			b.Fatal("result was not measured")
		}
	}
}

func BenchmarkActivityTarget(b *testing.B) {
	raw := json.RawMessage(`{"from":"https://a.example/x","to":"old.go","name":"n","path":"p"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = activityTarget(raw)
	}
}
