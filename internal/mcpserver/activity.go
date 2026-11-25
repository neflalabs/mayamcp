package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ActivityPrefix starts the message of every tool-activity log record, so UIs
// such as the TUI can recognise and highlight them.
const ActivityPrefix = "tool: "

// maxTargetLen caps the target shown in an activity line.
const maxTargetLen = 60

// activityMiddleware logs one concise line per tools/call, for example
// "ping ✓ 850µs" or "describe_server ✓ 1.2ms". The line carries a compact
// token estimate for the result, for example " ~10 tok"; exact byte counts
// and estimated token counts (bytes/4) for the `content` and
// `structuredContent` payloads are also attached as slog attributes.
//
// Only the tool name, a sanitized target derived from an explicit allowlist,
// the outcome, the duration and the size annotation are logged. Commands,
// file contents, edit text, environment values, URL userinfo and query
// strings (including signatures and Bearer tokens), error messages and
// tokens are never logged. A consumer can supply a TargetExtractor for
// domain-specific argument keys; its output is always sanitized with
// cleanTarget before it reaches the log.
func activityMiddleware(logger *slog.Logger, extractor TargetExtractor) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			started := time.Now()
			res, err := next(ctx, method, req)

			name, target := "unknown", ""
			if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
				name = cleanTarget(r.Params.Name)
				target = resolveTarget(name, r.Params.Arguments, extractor)
			}
			ok, detail := activityOutcomeWith(res, err)
			level := slog.LevelInfo
			if !ok {
				level = slog.LevelWarn
			}
			line := ActivityLine(name, target, ok, detail, time.Since(started))
			attrs := []any{}
			// JSON-encoding the result is O(payload). Skip it entirely when the
			// configured level would discard the record anyway.
			if logger.Enabled(ctx, level) {
				if size, measured := measureResult(res); measured {
					line += resultSizeSuffix(size)
					attrs = append(attrs,
						"content_bytes", size.contentBytes,
						"content_tokens", size.contentTokens,
						"structured_bytes", size.structuredBytes,
						"structured_tokens", size.structuredTokens,
					)
				}
			}
			logger.Log(ctx, level, ActivityPrefix+line, attrs...)
			return res, err
		}
	}
}

// ActivityMiddleware is the public form of activityMiddleware. Consumers that
// call mcp.NewServer themselves can add it to their own server without going
// through New or NewCore.
func ActivityMiddleware(logger *slog.Logger, extractor TargetExtractor) mcp.Middleware {
	return activityMiddleware(logger, extractor)
}

// ActivityLine formats one activity entry: tool, target, ✓/✗, optional detail
// and duration. Empty parts are left out.
func ActivityLine(tool, target string, ok bool, detail string, d time.Duration) string {
	parts := []string{tool}
	if target != "" {
		parts = append(parts, target)
	}
	if ok {
		parts = append(parts, "✓")
	} else {
		parts = append(parts, "✗")
	}
	if detail != "" {
		parts = append(parts, detail)
	}
	parts = append(parts, shortDur(d))
	return strings.Join(parts, " ")
}

// activityOutcomeWith decides success and a short, content-free detail.
// Tool-level failures are reported through CallToolResult.IsError; transport
// and protocol failures arrive as err. Result metadata is not inspected.
func activityOutcomeWith(res mcp.Result, err error) (bool, string) {
	if err != nil {
		return false, "error"
	}
	r, isTool := res.(*mcp.CallToolResult)
	if !isTool || r == nil {
		return true, ""
	}
	if r.IsError {
		return false, "error"
	}
	return true, ""
}

// resultSize summarises the exact byte size of one tool result and the
// corresponding estimated context-token counts (bytes/4). The two payloads a
// client may forward to the model are kept separate: the `content` block (what
// the model reads) and `structuredContent` (the same data as JSON, when the
// client passes both).
type resultSize struct {
	contentBytes     int64
	contentTokens    int64
	structuredBytes  int64
	structuredTokens int64
}

// byteCounter implements io.Writer without allocating a buffer, so JSON byte
// counting does not duplicate a large tool payload on every call.
type byteCounter int64

func (c *byteCounter) Write(p []byte) (int, error) {
	*c += byteCounter(len(p))
	return len(p), nil
}

// jsonByteLen matches len(json.Marshal(v)) without building the JSON buffer.
// json.Encoder.Encode appends a trailing newline, which is subtracted here.
func jsonByteLen(v any) int64 {
	var c byteCounter
	if err := json.NewEncoder(&c).Encode(v); err != nil {
		return 0
	}
	if c > 0 {
		return int64(c) - 1
	}
	return 0
}

// measureResult computes exact byte counts and bytes/4 token estimates. It
// reports false when res is not a tool result (protocol-level error or a
// non-tool method).
func measureResult(res mcp.Result) (resultSize, bool) {
	var s resultSize
	r, ok := res.(*mcp.CallToolResult)
	if !ok || r == nil {
		return s, false
	}
	if len(r.Content) > 0 {
		s.contentBytes = jsonByteLen(r.Content)
		s.contentTokens = s.contentBytes / 4
	}
	if r.StructuredContent != nil {
		s.structuredBytes = jsonByteLen(r.StructuredContent)
		s.structuredTokens = s.structuredBytes / 4
	}
	return s, true
}

// resultSizeSuffix renders the compact token annotation appended to the
// activity line, for example " ~10 tok". A client forwards either `content`
// or `structuredContent` (or both), so adding the two estimates would double
// count the same data; the annotation reports the larger of the two as the
// primary payload. Exact per-payload byte counts and token estimates remain
// available as slog attributes (content_bytes, content_tokens,
// structured_bytes, structured_tokens). Zero-cost results get no suffix.
func resultSizeSuffix(s resultSize) string {
	tokens := s.contentTokens
	if s.structuredTokens > tokens {
		tokens = s.structuredTokens
	}
	if tokens <= 0 {
		return ""
	}
	return fmt.Sprintf(" ~%s tok", humanCount(tokens))
}

// humanCount compacts a count with one decimal and a k/M suffix; values
// below 1000 are printed exactly. Values that would round to "1000.0k" are
// promoted to M so the annotation stays clean.
func humanCount(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 999_950: // one decimal would otherwise round up to "1000.0k"
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
}

// TargetExtractor derives a short target label from a tool call. The
// middleware always applies cleanTarget to its result; implementations must
// still never return secrets. A nil extractor uses the generic default below.
type TargetExtractor func(tool string, args json.RawMessage) string

// resolveTarget applies a custom extractor when configured, otherwise the
// default. The final cleanTarget keeps every line short and printable.
func resolveTarget(tool string, args json.RawMessage, extractor TargetExtractor) string {
	if extractor != nil {
		return cleanTarget(extractor(tool, args))
	}
	return activityTarget(args)
}

// activityTarget reads only generic argument keys: name, path, id, target and
// from/to. Domain-specific keys (git ref/remote, source_url, paths, pattern)
// belong in a consumer-supplied TargetExtractor. Commands and content-bearing
// keys are never read.
func activityTarget(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// A fixed struct avoids building a map[string]any on every tools/call.
	// Content-bearing keys are deliberately absent from this allowlist.
	var args struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Name   string `json:"name"`
		Path   string `json:"path"`
		ID     string `json:"id"`
		Target string `json:"target"`
	}
	if json.Unmarshal(raw, &args) != nil {
		return ""
	}
	value := func(s string) string { return safeTargetValue(s) }

	var target string
	switch {
	case args.From != "" || args.To != "":
		from, to := hostOrSafeValue(args.From), hostOrSafeValue(args.To)
		switch {
		case from != "" && to != "":
			target = from + " → " + to
		case from != "":
			target = from
		default:
			target = to
		}
	case value(args.Name) != "":
		target = value(args.Name)
	case value(args.Path) != "":
		target = value(args.Path)
	case value(args.ID) != "":
		target = value(args.ID)
	case value(args.Target) != "":
		target = value(args.Target)
	}
	return cleanTarget(target)
}

// hostOrSafeValue reduces a URL to its hostname, or sanitizes a plain value
// such as a local path. It never returns userinfo or query strings.
func hostOrSafeValue(raw string) string {
	if strings.Contains(raw, "://") {
		return urlHost(raw)
	}
	return safeTargetValue(raw)
}

// safeTargetValue sanitizes a non-URL string argument for logging. It keeps a
// single, query-free token and drops full URLs, credential-style userinfo
// (user:pass@host) and multi-word shell text. A plain scoped name such as
// "@scope/pkg" is kept: the "@" alone is not a secret. It returns "" when
// nothing safe remains.
func safeTargetValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
		s = s[:i]
	}
	if s == "" || strings.Contains(s, "://") {
		return ""
	}
	if i := strings.Index(s, "@"); i >= 0 && strings.Contains(s[:i], ":") {
		// user:pass@host style userinfo; never log it.
		return ""
	}
	return cleanTarget(s)
}

// urlHost returns only the hostname of raw (no userinfo, port, path, query
// or fragment), or "?" when it does not parse as a URL with a host.
func urlHost(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return "?"
	}
	return u.Hostname()
}

// cleanTarget keeps a target on one short printable line.
func cleanTarget(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxTargetLen {
		s = string(r[:maxTargetLen-1]) + "…"
	}
	return s
}

// shortDur renders a duration compactly: 850µs, 12ms, 1.2s, 2m3s.
func shortDur(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return d.Round(time.Second).String()
	}
}

// CountTools reports how many tools s exposes, by listing them over an
// in-memory session. The TUI shows this number while the server runs. When
// ctx has no deadline, a five-second bound prevents a stuck session from
// blocking the caller indefinitely.
func CountTools(ctx context.Context, s *mcp.Server) (int, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, serverT, nil)
	if err != nil {
		return 0, err
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: defaultName, Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		return 0, err
	}
	defer cs.Close()
	n := 0
	params := &mcp.ListToolsParams{}
	for {
		res, err := cs.ListTools(ctx, params)
		if err != nil {
			return 0, err
		}
		n += len(res.Tools)
		if res.NextCursor == "" {
			return n, nil
		}
		params.Cursor = res.NextCursor
	}
}
