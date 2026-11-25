# Package `api` Guide

`github.com/neflalabs/mayamcp/api` is the **only public contract**. It contains
type aliases and delegations; all logic lives in `internal/app` and
`internal/mcpserver`, which other modules cannot import because of Go's
`internal` rule.

## Choosing an entry point

| Need | Use | Starter tools |
|---|---|---|
| You already have an `mcp.Server` and just need helpers | `ResolveIdentity`, `NewLogger`, `ActivityMiddleware`, `AddTextTool`, `AddUntypedTool` | None (your own server) |
| You want a ready server with your own tools | `NewWithOptions(host, opts)` | Present when `Register == nil`, absent when non-nil |
| You want the full lifecycle: hooks, resources, middleware | `NewCore(cfg)` | Same as `NewWithOptions` |
| Legacy compatibility, explicit version argument | `New(host, version, opts)` | Same as `NewWithOptions` |

Prefer `NewWithOptions` over `New`: the version comes from `Options.Version`,
then `host.Version()`, then the default — no need to write it twice.

## Export map

**Identity and logging**
- `type Identity { Name, Title, Version string }` + `ResolveIdentity(Identity) mcp.Implementation`
- `type LogOptions { Writer, Logger, Level, JSON }` + `NewLogger(LogOptions) *slog.Logger`
- `type Host` — minimal domain contract: `Name() string`, `Version() string`

**Tool registration**
- `type Handler[In, Out any]` — generic handler shape
- `AddTextTool[In, Out]` — `content` only, no `outputSchema`/`structuredContent`
- `AddUntypedTool[In, Out]` — `structuredContent`, no `outputSchema`
- `mcp.AddTool` (SDK) — `content` + `outputSchema` for small typed records

**Runtime**
- `NewCore(Config) *Core` — `Core{ Server, Host, Logger }`
- `NewWithOptions`, `New`
- Neither starts a listener; you choose the transport.

**Observability**
- `ActivityMiddleware(logger, extractor) mcp.Middleware`
- `MeasureActivity(res) (ActivitySize, bool)` and `ActivitySize.Suffix(verbose)`
- `SanitizeTarget(s) string`
- `const ActivityPrefix`, `ActivityLine(tool, target, ok, detail, d)`
- `CountTools(ctx, s)`

Every helper has defaults: `mayamcp`, `dev`, `MayaMCP`, generic instructions,
prefix `tool: `. See [`observability.md`](observability.md) for the log format.

## Identity resolution

Field | Priority (highest → lowest)
---|---
Name | `Options.Name` → `Host.Name()` → `DefaultName` (`mayamcp`)
Version | `Options.Version` → `version` argument (`New`) → `Host.Version()` → `DefaultVersion` (`dev`)
Title | `Options.Title` → `defaultTitle` (`MayaMCP`)
Instructions | `Options.Instructions` → generic template

The resolved values are used **identically** for `initialize.serverInfo` and
the `describe_server` tool. `Host` is never wrapped, so extra domain methods
remain available.

## Lifecycle hooks

Field | Behaviour
---|---
`Register == nil` | `ping` + `describe_server` are registered (see `cmd/mayamcp`)
`Register != nil` | Only consumer tools; starter tools are **not** registered
`RegisterResources != nil` | Called after server creation to add resources
`Middleware` | Added **before** the activity middleware (custom → activity)
`TargetExtractor == nil` | Generic extractor: `name`, `path`, `id`, `target`, `from`/`to`
`DisableActivityMiddleware` | Skips the built-in activity middleware when you install your own

A complete example is in [`tool-authoring.md`](tool-authoring.md).

## A safe `TargetExtractor`

```go
extractor := func(tool string, args json.RawMessage) string {
	var in struct {
		Repo string `json:"repo"`
	}
	if json.Unmarshal(args, &in) != nil {
		return ""
	}
	// Never return a token, host path, or field contents.
	return in.Repo
}

opts := api.Options{TargetExtractor: extractor}
```

Extractor output is always passed through `api.SanitizeTarget` (control
characters normalized, whitespace collapsed, capped at 60 runes). That is
**formatting, not redaction**: the extractor itself must still never return a
secret.

## Measuring result size

```go
if size, ok := api.MeasureActivity(res); ok {
	_ = size.Suffix(false) // " ~10 tok"
	_ = size.Suffix(true)  // " ~10 tok (ct 6 tok + st 4 tok)"
}
```

Token estimates are `bytes/4`, not exact tokenizer counts. The compact suffix
uses the **larger** payload (`content` or `structuredContent`) to avoid double
counting.

## Compatibility

Module versions are semver `v0.MINOR.PATCH`. Additions to `api` are allowed
without a MINOR bump; removals or signature changes require one plus a migration
note in `CHANGELOG.md`. The tool contract and `ActivityPrefix` never change
through a routine `api` change. CalVer labels in the changelog are human-readable
release dates, not module versions.

See the package godoc for details: `go doc github.com/neflalabs/mayamcp/api`.
