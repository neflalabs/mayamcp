package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestActivityTargetNeverLogsSecrets(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		want      string
		forbidden []string
	}{
		{
			name:      "command with bearer token and args",
			args:      `{"command":"curl -H 'Authorization: Bearer SECRET123' https://example.com","args":["--data","SECRET123"]}`,
			want:      "",
			forbidden: []string{"SECRET123", "Bearer", "curl", "example.com"},
		},
		{
			name:      "from and to with userinfo and signed queries",
			args:      `{"from":"https://user:pass@host.example/a?sig=SECRET_FROM","to":"https://other.example/b?token=SECRET_TO"}`,
			want:      "host.example → other.example",
			forbidden: []string{"SECRET_FROM", "SECRET_TO", "user:pass", "pass@", "sig=", "token="},
		},
		{
			name:      "source_url ignored, generic path kept",
			args:      `{"source_url":"https://user:pass@host.example/a?sig=SECRET_FROM","path":"/docs?token=SECRET_PATH"}`,
			want:      "/docs",
			forbidden: []string{"SECRET_FROM", "SECRET_PATH", "user:pass", "sig=", "token="},
		},
		{
			name:      "path holding a credentialed URL",
			args:      `{"path":"https://user:pass@host.example/x?sig=SECRET_PATH"}`,
			want:      "",
			forbidden: []string{"SECRET_PATH", "user:pass", "host.example", "://"},
		},
		{
			name:      "name with userinfo",
			args:      `{"name":"user:pass@SECRET_HOST"}`,
			want:      "",
			forbidden: []string{"SECRET_HOST", "user:pass"},
		},
		{
			name:      "scoped name keeps @",
			args:      `{"name":"@scope/pkg"}`,
			want:      "@scope/pkg",
			forbidden: nil,
		},
		{
			name:      "git ref ignored by default extractor",
			args:      `{"ref":"refs/heads/main?token=SECRET_REF"}`,
			want:      "",
			forbidden: []string{"SECRET_REF", "token="},
		},
		{
			name:      "remote holding a credentialed URL",
			args:      `{"remote":"https://user:pass@git.example/repo.git?token=SECRET_REMOTE"}`,
			want:      "",
			forbidden: []string{"SECRET_REMOTE", "user:pass", "git.example"},
		},
		{
			name:      "search pattern ignored by default extractor",
			args:      `{"pattern":"foo Bearer SECRET_PATTERN"}`,
			want:      "",
			forbidden: []string{"SECRET_PATTERN", "Bearer"},
		},
		{
			name:      "paths ignored by default extractor",
			args:      `{"paths":["/a?token=SECRET_PATHS","/b","/c"]}`,
			want:      "",
			forbidden: []string{"SECRET_PATHS", "token="},
		},
		{
			name:      "unknown keys are ignored",
			args:      `{"token":"SECRET_TOKEN","env":{"TOKEN":"SECRET_ENV"},"url":"https://user:pass@host.example/?sig=SECRET_URL"}`,
			want:      "",
			forbidden: []string{"SECRET_TOKEN", "SECRET_ENV", "SECRET_URL", "user:pass"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := activityTarget(json.RawMessage(tc.args))
			if got != tc.want {
				t.Errorf("activityTarget() = %q, want %q", got, tc.want)
			}
			for _, secret := range tc.forbidden {
				if strings.Contains(got, secret) {
					t.Errorf("activityTarget() = %q, leaked %q", got, secret)
				}
			}
		})
	}
}

func TestActivityTargetSafeValues(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"path", `{"path":"/tmp/report.txt"}`, "/tmp/report.txt"},
		{"name", `{"name":"describe_server"}`, "describe_server"},
		{"id", `{"id":"job-42"}`, "job-42"},
		{"target", `{"target":"service-a"}`, "service-a"},
		{"ref ignored by default", `{"ref":"refs/heads/main"}`, ""},
		{"remote ignored by default", `{"remote":"origin"}`, ""},
		{"pattern ignored by default", `{"pattern":"^foo.*bar$"}`, ""},
		{"paths ignored by default", `{"paths":["a.go","b.go","c.go"]}`, ""},
		{"from and to hosts", `{"from":"https://host.example/a","to":"https://other.example/b"}`, "host.example → other.example"},
		{"source_url ignored, path kept", `{"source_url":"https://host.example/a","path":"/docs"}`, "/docs"},
		{"command ignored, path kept", `{"command":"curl https://host.example","path":"/tmp/f.txt"}`, "/tmp/f.txt"},
		{"invalid json", `{`, ""},
		{"empty", `{}`, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := activityTarget(json.RawMessage(tc.args)); got != tc.want {
				t.Errorf("activityTarget() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestActivityTargetTruncatesLongTargets(t *testing.T) {
	args, err := json.Marshal(map[string]any{"path": strings.Repeat("a", 100)})
	if err != nil {
		t.Fatal(err)
	}
	got := activityTarget(args)
	want := strings.Repeat("a", maxTargetLen-1) + "…"
	if got != want {
		t.Errorf("activityTarget() = %q, want %q", got, want)
	}
}

func TestActivityLine(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		target string
		ok     bool
		detail string
		d      time.Duration
		want   string
	}{
		{"success", "ping", "", true, "", 850 * time.Microsecond, "ping ✓ 850µs"},
		{"with target", "write_file", "main.go", true, "", 3 * time.Millisecond, "write_file main.go ✓ 3ms"},
		{"tool error", "read_file", "a.txt", false, "error", 1200 * time.Millisecond, "read_file a.txt ✗ error 1.2s"},
		{"long duration", "fetch", "", false, "", 125 * time.Second, "fetch ✗ 2m5s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ActivityLine(tc.tool, tc.target, tc.ok, tc.detail, tc.d); got != tc.want {
				t.Errorf("ActivityLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShortDur(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0µs"},
		{999 * time.Microsecond, "999µs"},
		{time.Millisecond, "1ms"},
		{12600 * time.Microsecond, "12ms"},
		{1200 * time.Millisecond, "1.2s"},
		{62 * time.Second, "1m2s"},
	}

	for _, tc := range tests {
		if got := shortDur(tc.d); got != tc.want {
			t.Errorf("shortDur(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestHumanCount(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{9999, "10.0k"},
		{999_949, "999.9k"},
		{999_999, "1.0M"},
		{1_000_000, "1.0M"},
		{12_345_678, "12.3M"},
	}

	for _, tc := range tests {
		if got := humanCount(tc.n); got != tc.want {
			t.Errorf("humanCount(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestMeasureResultAndSuffixDoNotDoubleCount(t *testing.T) {
	tests := []struct {
		name string
		res  mcp.Result
		ok   bool
	}{
		{
			name: "both payloads",
			res: &mcp.CallToolResult{
				Content:           []mcp.Content{&mcp.TextContent{Text: strings.Repeat("x", 40)}},
				StructuredContent: map[string]any{"name": "described"},
			},
			ok: true,
		},
		{
			name: "content only",
			res: &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("y", 40)}},
			},
			ok: true,
		},
		{
			name: "structured only",
			res:  &mcp.CallToolResult{StructuredContent: map[string]any{"name": "described"}},
			ok:   true,
		},
		{
			name: "empty result",
			res:  &mcp.CallToolResult{},
			ok:   true,
		},
		{"non-tool result", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			size, ok := measureResult(tc.res)
			if ok != tc.ok {
				t.Fatalf("measureResult() ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				if got := resultSizeSuffix(size); got != "" {
					t.Errorf("resultSizeSuffix() = %q, want empty", got)
				}
				return
			}
			tokens := size.contentTokens
			if size.structuredTokens > tokens {
				tokens = size.structuredTokens
			}
			want := ""
			if tokens > 0 {
				want = " ~" + humanCount(tokens) + " tok"
			}
			got := resultSizeSuffix(size)
			if got != want {
				t.Errorf("resultSizeSuffix() = %q, want %q (content=%d, structured=%d tokens)", got, want, size.contentTokens, size.structuredTokens)
			}
			if strings.Contains(got, "ct ") || strings.Contains(got, "st ") || strings.Contains(got, "+") {
				t.Errorf("resultSizeSuffix() = %q, should not sum both payloads", got)
			}
		})
	}
}

func TestActivityOutcomeWith(t *testing.T) {
	tests := []struct {
		name       string
		res        mcp.Result
		err        error
		wantOK     bool
		wantDetail string
	}{
		{"protocol error", nil, errors.New("boom"), false, "error"},
		{"nil result", nil, nil, true, ""},
		{"success", &mcp.CallToolResult{}, nil, true, ""},
		{"tool error", &mcp.CallToolResult{IsError: true}, nil, false, "error"},
		{
			"result metadata is not inspected",
			&mcp.CallToolResult{Meta: mcp.Meta{"mayamcp/exitCode": 2, "mayamcp/timedOut": true}},
			nil, true, "",
		},
		{
			"tool error wins over metadata",
			&mcp.CallToolResult{IsError: true, Meta: mcp.Meta{"mayamcp/exitCode": 1}},
			nil, false, "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, detail := activityOutcomeWith(tc.res, tc.err)
			if ok != tc.wantOK || detail != tc.wantDetail {
				t.Errorf("activityOutcomeWith() = (%v, %q), want (%v, %q)", ok, detail, tc.wantOK, tc.wantDetail)
			}
		})
	}
}

type activityTestHost struct{}

func (activityTestHost) Name() string    { return "activity-test" }
func (activityTestHost) Version() string { return "v1" }

func TestActivityTargetHandlesLocalFromTo(t *testing.T) {
	got := activityTarget(json.RawMessage(`{"from":"old.go","to":"new.go"}`))
	if got != "old.go → new.go" {
		t.Fatalf("activityTarget = %q, want %q", got, "old.go → new.go")
	}
}

func TestCustomTargetExtractorIsSanitized(t *testing.T) {
	custom := func(tool string, args json.RawMessage) string {
		if tool != "fetch" {
			return ""
		}
		var in struct {
			SourceURL string `json:"source_url"`
		}
		if json.Unmarshal(args, &in) != nil {
			return ""
		}
		return hostOrSafeValue(in.SourceURL) + "\n" + strings.Repeat("x", 100)
	}
	got := resolveTarget("fetch", json.RawMessage(`{"source_url":"https://user:pass@host.example/a?sig=SECRET"}`), custom)
	if strings.Contains(got, "SECRET") || strings.Contains(got, "user:pass") || strings.Contains(got, "\n") {
		t.Fatalf("custom extractor output was not sanitized: %q", got)
	}
	if r := []rune(got); len(r) != maxTargetLen {
		t.Errorf("target runes = %d, want %d: %q", len(r), maxTargetLen, got)
	}
}

func TestJSONByteLenMatchesMarshal(t *testing.T) {
	values := []any{
		nil,
		"",
		"hello",
		map[string]any{"a": 1, "b": "two"},
		[]string{"x", "y"},
		[]mcp.Content{&mcp.TextContent{Text: "pong"}},
	}
	for _, v := range values {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", v, err)
		}
		if got, want := jsonByteLen(v), int64(len(raw)); got != want {
			t.Errorf("jsonByteLen(%v) = %d, want %d", v, got, want)
		}
	}
	if got := jsonByteLen(make(chan int)); got != 0 {
		t.Errorf("jsonByteLen(chan) = %d, want 0 on encode error", got)
	}
	if got := urlHost("://bad"); got != "?" {
		t.Errorf("urlHost(invalid) = %q, want ?", got)
	}
}

func TestCountToolsUsesNeutralClientName(t *testing.T) {
	n, err := CountTools(context.Background(), New(activityTestHost{}, "v1", Options{}))
	if err != nil {
		t.Fatalf("CountTools: %v", err)
	}
	if n < 1 {
		t.Errorf("CountTools() = %d, want at least one tool", n)
	}
}

func TestCountToolsHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CountTools(ctx, New(activityTestHost{}, "v1", Options{})); err == nil {
		t.Fatal("CountTools returned nil with an already-canceled context")
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestActivityMiddlewareLogsToolCallsEndToEnd(t *testing.T) {
	var buf lockedBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := New(activityTestHost{}, "v1", Options{Logger: logger})
	mcp.AddTool(s, &mcp.Tool{Name: "failing_tool", Description: "Always reports a tool error."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "boom"}}}, nil, nil
		})
	AddTextTool(s, &mcp.Tool{Name: "secret_probe", Description: "Probe for secret leakage."},
		func(context.Context, *mcp.CallToolRequest, struct {
			Command string `json:"command"`
		}) (*mcp.CallToolResult, any, error) {
			return textResult("ok"), nil, nil
		})

	serverT, clientT := mcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := s.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "middleware-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer cs.Close()

	// The initialize handshake is not a tool call: it must not emit activity.
	if strings.Contains(buf.String(), ActivityPrefix) {
		t.Fatalf("non-tool method logged activity: %s", buf.String())
	}

	if _, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "ping"}); err != nil {
		t.Fatalf("CallTool ping: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"level=INFO", "tool: ping ✓", "content_bytes=", "content_tokens=", "structured_bytes=", "structured_tokens="} {
		if !strings.Contains(got, want) {
			t.Errorf("ping log missing %q:\n%s", want, got)
		}
	}

	if _, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "failing_tool"}); err != nil {
		t.Fatalf("CallTool failing_tool: %v", err)
	}
	got = buf.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "tool: failing_tool ✗ error") {
		t.Errorf("failing tool log missing WARN/failure:\n%s", got)
	}

	const secret = "SECRET_ACTIVITY"
	_, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "secret_probe",
		Arguments: map[string]any{"command": "curl -H 'Authorization: Bearer " + secret + "' https://example.com"},
	})
	if err != nil {
		t.Fatalf("CallTool secret_probe: %v", err)
	}
	if strings.Contains(buf.String(), secret) || strings.Contains(buf.String(), "Bearer") || strings.Contains(buf.String(), "example.com") {
		t.Fatalf("secret leaked into activity log:\n%s", buf.String())
	}
}
