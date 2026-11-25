# Documentation Index

Start with the row that matches what you are trying to do.

| Document | For | Contents |
|---|---|---|
| [`GLOSSARY.md`](GLOSSARY.md) | "What does this term mean?" | Every project-specific and protocol term, defined once |
| [`quickstart.md`](quickstart.md) | "I want a server running now" | `go get`, three consumption paths, stdio/HTTP examples, troubleshooting |
| [`api-guide.md`](api-guide.md) | "I want to use this as a library" | Export map, `New` vs `NewWithOptions` vs `NewCore`, hooks, identity, `TargetExtractor` |
| [`creating-tools.md`](creating-tools.md) | "I want to add one tool" | Input schema, domain vs protocol errors, registration helpers, golden contract |
| [`tool-authoring.md`](tool-authoring.md) | "I am building a full domain server" | End-to-end example with hooks, resources, middleware, and tests |
| [`observability.md`](observability.md) | "I consume or change activity logs" | `tool: ` format, token suffix, slog attributes, privacy guarantees |

All guides are in English. A short explanation of the project name lives in
[`README.md`](../README.md#why-mayamcp).

## Elsewhere in the repository

- [`README.md`](../README.md) — project overview, `make` targets, versioning.
- [`AGENTS.md`](../AGENTS.md) — conventions for agents and contributors.
- [`CHANGELOG.md`](../CHANGELOG.md) — release notes (semver releases, CalVer
  date labels).
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contribution workflow and gates.
- [`SECURITY.md`](../SECURITY.md) — private vulnerability reporting.
- `consumer/` — a real module that imports only `api`
  ([`consumer/README.md`](../consumer/README.md)).

## Terms used in these guides

| Term | Meaning here |
|---|---|
| core | Reusable MCP server code in `internal/`; no transport, auth, UI, or domain logic |
| consuming application | The project that imports MayaMCP and supplies its own domain |
| tool contract | The locked `tools/list` output (golden file + byte budget) |
| activity record | One privacy-safe log line per tool call, prefixed `tool: ` |

## Rules that are easy to miss

- The public contract is the `api` package; `internal/` cannot be imported from
  another module.
- `tools/list` is locked by a golden file (**884 bytes** pretty, **601 bytes**
  compact payload) with a 2048-byte budget. Never raise the budget casually.
- `ActivityPrefix = "tool: "` is a consumer contract used by log readers and
  terminal UIs.
- Bugfixes start with a failing reproduction test, then the fix.
