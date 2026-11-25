# Glossary

One place for every term that is easy to misread in this repository. If a word
below appears in a guide, this is its meaning here.

## Project terms

| Term | Meaning |
|---|---|
| **C-lite** | "C" is an **architecture level label** (A = copy the template, B = call helper functions, C = framework owns the lifecycle). C-lite is the lightweight internal framework mode: `NewCore` plus a small set of hooks, no plugin system. It has nothing to do with the C programming language. |
| **Prove-It** | Write a failing reproduction test **before** changing production code, then make it pass. |
| **Stop-the-Line** | Fix the root cause of a failing test or build before adding new features. |
| **golden / golden test** | The locked `tools/list` snapshot in `internal/mcpserver/testdata/tools.golden.json` (884-byte pretty file). `TestToolContract` compares against it; regenerate deliberately with `-args -update`. |
| **budget** | The 2048-byte cap on the **compact `tools/list` payload**, measured by `TestToolsListBudget` (601 bytes for the default two tools). It is not the golden file size. |
| **CalVer** | Date-shaped release labels such as `v2026.11.8`, kept for human-readable changelog dates and old binary tags. Not valid Go module versions. |
| **semver** | Module version `v0.MINOR.PATCH` used by `go get` and GitHub Releases. `v0.1.0` is the current baseline. |
| **slice S1–S20 / R1–R10** | Internal planning IDs from the private architecture notebook. Changelog entries describe them in plain language instead. |

## API names in plain words

| Term | Meaning |
|---|---|
| **façade** | A thin public wrapper (`api/`) that delegates to internal code; it contains no logic of its own. |
| **downstream** | Any application that imports MayaMCP. |
| **reference consumer** | The upstream application used to validate the core and prove parity. You do not need it to use this template. |
| **NgodingMCP** | The upstream application this template was extracted from. Its identity, UI, and domain are deliberately not part of MayaMCP. |
| **Host** | The tiny interface a consumer implements: `Name()` and `Version()`. |
| **NewCore** | Builder that assembles identity, logger, hooks, and activity middleware without starting a listener. |
| **TargetExtractor** | Optional function that derives a short, safe label from tool arguments for the activity log. Sanitized by the core, but must never return secrets. |
| **ActivityPrefix** | The `tool: ` prefix that marks an activity log record. |
| **tool contract** | The tool names, schemas, and `tools/list` output locked by the golden file and budget test. |
| **activity record** | One privacy-safe log line per `tools/call`: tool, optional target, outcome, duration, and token estimate. |

## Everyday technical terms

| Term | Meaning |
|---|---|
| **MCP** | Model Context Protocol — the protocol this server speaks. |
| **tools/list** | MCP method that lists available tools. Its size is a per-session token cost. |
| **inputSchema** | JSON schema of a tool's input, inferred from the Go input struct. |
| **outputSchema** | JSON schema of a typed tool output. Useful for small records; costs tokens in `tools/list`. |
| **structuredContent** | Typed JSON payload returned alongside text content. |
| **Streamable HTTP** | MCP transport over HTTP. This template uses `Stateless: true` so restarts do not break sessions. |
| **SSE** | Server-Sent Events: a long-lived response stream, which is why `WriteTimeout` is left at zero. |
| **stdio** | Transport using stdin/stdout. stdout is reserved for the MCP protocol; logs go to stderr. |
| **slog** | Go's standard `log/slog` package. |
| **go.work** | Local Go workspace file that links sibling modules. Machine-local, not committed here. |
| **replace** | A `go.mod` directive that substitutes a module with a local path during development. |
| **-ldflags -X** | Build-time injection of a version variable into the binary. |
