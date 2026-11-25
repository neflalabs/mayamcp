# AGENTS.md

Short guidance for AI agents and contributors working in this template repo.

## Structure

- `internal/mcpserver` — transport-agnostic core; one file per tool domain.
- `internal/app` — `NewCore`, server construction without a transport.
- `cmd/mayamcp` — transport selector: stdio (default) or stateless HTTP.
- `api/` — the only public import surface; aliases and delegation only.
- `consumer/` — separate module that imports only `api`.

## Conventions

- Prefer the Go standard library; justify any new dependency.
- Identity options (`Name`, `Title`, `Instructions`, `Version`) must have
  defaults; never hardcode a downstream application's identity in the core.
- The tool contract is locked by a golden test. For a contract change, update
  `testdata/tools.golden.json` (pretty file **884 bytes**) with
  `go test ./internal/mcpserver -run TestToolContract -args -update`, then
  review the diff. `TestToolsListBudget` measures the compact `tools/list`
  payload (**601 bytes** of 2048), not the golden file size.
- Never raise the `tools/list` budget without an explicit decision.
- Use `AddTextTool` for large text results; `outputSchema` via `mcp.AddTool`
  only for small records clients truly need. This pattern is inherited from
  the upstream application.
- User-facing behaviour changes must update `README.md` / `docs/`.
- Core code stays free of transport, auth, UI, and domain specifics. Keep the
  boundary between the reusable core and the consuming application clear.

## Commands

```sh
make test            # go test -count=1 ./...
make check           # gofmt, vet, staticcheck, go test -race
make build           # binary ./mayamcp
make consumer-test   # external module importing only api
make release         # cross-platform release artifacts
make rename          # instantiate the template under another module path
```
