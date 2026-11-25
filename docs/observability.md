# Observability: Activity Log

The only built-in observability output in MayaMCP is **one log line per
`tools/call`**. Its format is a contract: downstream log readers and terminal
UIs recognise lines by the `tool: ` prefix.

## Format

```text
tool: <tool> [<target>] ✓|✗ [detail] <duration> [~N tok]
```

Examples:

```text
tool: ping ✓ 850µs
tool: describe_server ✓ 1.2ms ~10 tok
tool: failing_tool ✗ error 3.1ms
tool: edit_file main.go ✓ 12ms ~8.3k tok
```

| Part | Rule |
|---|---|
| Prefix | `api.ActivityPrefix` = `"tool: "`, always at the start of `msg` |
| Tool | Tool name, already sanitized |
| Target | Optional, from the generic key allowlist or a `TargetExtractor` |
| Outcome | `✓` success, `✗` failure |
| Detail | Only `error`, `timed out`, or `exit N` — never raw error text |
| Duration | `µs`, `ms`, `s`, or `m…s` |
| Suffix | ` ~N tok` only when the token estimate is greater than zero |

Successes are logged at `INFO`, failures at `WARN`.

## slog attributes

Besides `msg`, each record carries exact attributes:

| Attribute | Meaning |
|---|---|
| `content_bytes` | Byte size of `Content` after JSON encoding |
| `content_tokens` | `content_bytes / 4` |
| `structured_bytes` | Byte size of `StructuredContent` |
| `structured_tokens` | `structured_bytes / 4` |

Token estimates are `bytes/4`, not tokenizer output. The compact suffix uses
the **larger** payload because a client forwards one or both to the model —
summing both would double count. Consumers that want the breakdown can call
`api.MeasureActivity(res).Suffix(true)`, which produces
` ~N tok (ct X + st Y)` (the verbose format used by some downstream clients).

## Privacy guarantees

The following never reach the log:

- `command`, `args`, file contents, edit text, environment values, tokens,
  signatures;
- URL userinfo, port, query, and fragment — only the hostname is retained;
- raw error messages (only `error`/`timed out`/`exit N`);
- credential-style `user:pass@host` values.

Targets come from an explicit allowlist: `name`, `path`, `id`, `target`, and
`from`/`to`. Non-URL `from`/`to` values are sanitized as local paths
(`old.go → new.go`) instead of becoming `? → ?`.

A consumer `TargetExtractor` runs first, and its result is always passed
through `api.SanitizeTarget` (control characters normalized, whitespace
collapsed, capped at 60 runes before an ellipsis). The sanitizer **is not a
redactor**: extractors must still never return secrets.

## Using it on your own server

```go
srv := mcp.NewServer(&impl, nil)
api.AddTextTool(srv, tool, handler)
srv.AddReceivingMiddleware(api.ActivityMiddleware(logger, extractor))
```

With `NewCore`, if you already have domain-aware middleware, set
`Config.DisableActivityMiddleware = true` and install your own — for example,
to surface domain `_meta` exit codes as `✗ exit N`.

## Consuming the lines

```go
if strings.HasPrefix(record.Message, api.ActivityPrefix) {
	line := strings.TrimPrefix(record.Message, api.ActivityPrefix)
	// tokenize lightly: tool [target] ✓/✗ [detail] duration [suffix]
}
```

Do not parse by splitting on the Nth space: `target` and `detail` can be empty.
Use the `✓`/`✗` token as the status marker.

## Regression tests

- `internal/mcpserver/activity_test.go` — secret leakage, allowlist, suffixes,
  outcomes, and end-to-end middleware.
- `api/toolkit_test.go` — public `ActivityMiddleware` wrapper, `MeasureActivity`,
  `SanitizeTarget`, and `New` hooks.
- Downstream consumers keep their own `_meta` outcome tests; the core's tool
  contract is guarded by the golden test and `TestToolsListBudget`.

A change to the prefix, line format, or attributes is a contract change —
update tests and golden files in the same commit.
