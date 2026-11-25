# Documentation Index

Start with the row that matches what you are trying to do.

| Document | For | Contents |
|---|---|---|
| [`quickstart.md`](quickstart.md) | "I want a server running now" | `go get`, three consumption paths, stdio/HTTP examples, troubleshooting |
| [`api-guide.md`](api-guide.md) | "I want to use this as a library" | Export map, `New` vs `NewWithOptions` vs `NewCore`, hooks, identity, `TargetExtractor` |
| [`creating-tools.md`](creating-tools.md) | "I want to add one tool" | Input schema, domain vs protocol errors, registration helpers, golden contract |
| [`tool-authoring.md`](tool-authoring.md) | "I am building a full domain server" | End-to-end example with hooks, resources, middleware, and tests |
| [`observability.md`](observability.md) | "I consume or change activity logs" | `tool: ` format, token suffix, slog attributes, privacy guarantees |

The detailed guides above are currently written in Indonesian; the root
[`README.md`](../README.md), this index, and
[`CONTRIBUTING.md`](../CONTRIBUTING.md) are in English.

The original extraction notes from `ngodingmcp` are **not** part of the remote
repository. They are archived locally at `TaskDev/Note/MCP_AS_TEMPLATE.md`.
A short explanation of the name lives in
[`README.md`](../README.md#why-mayamcp).

## Elsewhere in the repository

- [`README.md`](../README.md) — project overview, `make` targets, versioning.
- [`AGENTS.md`](../AGENTS.md) — conventions for agents and contributors.
- [`CHANGELOG.md`](../CHANGELOG.md) — CalVer release notes.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contribution workflow and gates.
- `consumer/` — a real module that imports only `api`
  ([`consumer/README.md`](../consumer/README.md)).
- `TaskDev/` — local architecture notebook; gitignored, so links to it do not
  work in a public clone. Key decisions are summarised in the documents above.

## Rules that are easy to miss

- The public contract is the `api` package; `internal/` cannot be imported from
  another module.
- `tools/list` is locked by a golden file (**884 bytes** pretty, **601 bytes**
  compact payload) with a 2048-byte budget. Never raise the budget casually.
- `ActivityPrefix = "tool: "` is a consumer contract (TUI).
- Bugfixes use the Prove-It pattern: failing reproduction first, fix second.
