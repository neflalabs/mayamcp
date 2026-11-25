# AGENTS.md

Short guidance for AI agents and contributors working in this template repo.

## Structure

- `internal/mcpserver` — transport-agnostic core; one file per tool domain.
- `internal/app` — `NewCore`, server construction without a transport.
- `cmd/mayamcp` — transport selector: stdio (default) or stateless HTTP.
- `api/` — the only public import surface; aliases and delegation only.

## Conventions

- Prefer the Go standard library; justify any new dependency.
- Identity options (`Name`, `Title`, `Instructions`, `Version`) must have
  defaults; never hardcode a consumer identity in the core.
- The tool contract is locked by a golden test. For a contract change, update
  `testdata/tools.golden.json` (pretty file **884 bytes**) with
  `go test ./internal/mcpserver -run TestToolContract -args -update`, then
  review the diff. `TestToolsListBudget` measures the compact `tools/list`
  payload (**601 bytes** of 2048), not the golden file size.
- Never raise the `tools/list` budget without an explicit decision.
- Use `AddTextTool` for large text results; `outputSchema` via `mcp.AddTool`
  only for small records clients truly need. This pattern is inherited from
  `ngodingmcp` (local archive: `TaskDev/Note/MCP_AS_TEMPLATE.md`).
- User-facing behaviour changes must update `README.md` / `docs/`.
- `TaskDev/README.md` must contain the project name `MayaMCP`. The taskdev CLI
  has a hardcoded `Nefla POS` fallback, so after every `taskdev_cli.py sync`
  (especially after a from-scratch regeneration) check the
  `proyek **MayaMCP**` line and correct it manually if needed. Do not assume
  the CLI detects the name from the repository.
- `TaskDev/` is private (gitignored), so notebook validation runs locally via
  `make taskdev-validate`, not in CI.

## Commands

```sh
make test             # go test -count=1 ./...
make check            # gofmt, vet, staticcheck, go test -race
make build            # binary ./mayamcp
make consumer-test    # external module importing only api
make parity-check     # ngodingmcp golden + build both repos
make release          # cross-platform release artifacts
make rename           # instantiate the template under another module path
make taskdev-validate # taskdev_cli.py validate --strict (local)
make taskdev-sync     # regenerate TaskDev index and keep the MayaMCP name
```
