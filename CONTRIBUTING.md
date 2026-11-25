# Contributing to MayaMCP

MayaMCP is a template: keep the core small, the contract stable, and every
change backed by a test.

## Before you start

1. Read [`AGENTS.md`](AGENTS.md) for conventions (golden tests, identity,
   core vs appliance boundary, local TaskDev notebook).
2. Read [`docs/README.md`](docs/README.md) for the documentation map.
3. Install Go 1.26+ and run `make check`. The baseline must be green before you
   add anything (Stop-the-Line).

## Workflow

```sh
make test            # fast iteration
make check           # gofmt, vet, staticcheck, go test -race
make consumer-test   # external import path must keep working
make parity-check    # ngodingmcp golden unchanged (needs the sibling repo)
```

If `make check` fails, fix the root cause before adding features.

### Updating the parity baseline

`make parity-check` compares the `ngodingmcp` tool golden against the hash in
`testdata/parity/ngodingmcp.golden.sha256`. CI checks out the public
`ngodingmcp` repository, so an intentional tool change there will fail
mayamcp's CI until the baseline is updated. Procedure:

1. Regenerate the golden in `ngodingmcp` with its `-args -update` contract
   test and review the diff there.
2. Compute `sha256sum ngodingmcp/internal/mcpserver/testdata/tools.golden.json`.
3. Replace the single line in `testdata/parity/ngodingmcp.golden.sha256` in
   **the same commit** as the related mayamcp change.
4. Run `make parity-check` locally before pushing.

Do not update the hash merely to silence a failure you did not investigate.

### Fuzzing and benchmarks

```sh
go test ./internal/mcpserver -run Fuzz          # seed corpus
go test ./internal/mcpserver -fuzz FuzzActivityTarget -fuzztime=30s
go test ./internal/mcpserver -run '^$' -bench . -benchmem
```

The fuzz targets assert that sanitized targets stay bounded and free of
control characters. Benchmarks cover `MeasureActivity` and `activityTarget`;
use them before changing the hot path in the activity middleware.

## Prove-It pattern

For bugfixes and contract changes:

1. Write a **failing** reproduction test first.
2. Fix the production code until it passes.
3. Never disable a test or raise a budget as a shortcut.

See `internal/mcpserver/activity_test.go` and `cmd/mayamcp/main_test.go`.

## Tool contract

`internal/mcpserver/testdata/tools.golden.json` locks `tools/list` (884-byte
pretty file, 601-byte compact payload, 2048-byte budget).

- Renames, description changes, and schema changes must be deliberate.
- Regenerate with:
  ```sh
  go test ./internal/mcpserver -run TestToolContract -args -update
  ```
- Review the golden diff in the same commit. The budget never rises without an
  explicit decision.

## Architecture boundary

| Belongs in the core | Belongs in the consumer |
|---|---|
| `Host` interface, identity + defaults, logger, activity middleware | Auth/OAuth, Web UI/TUI, systemd, config store |
| `api/` façade, `ping`/`describe_server`, golden + budget tests | Domain tools, filesystem, workers, custom resources |

`internal/` cannot be imported from another module — everything public goes
through `api/`. See [`docs/api-guide.md`](docs/api-guide.md).

## User-facing changes

Update `README.md`/`docs/` whenever behaviour changes. When adding an exported
function to `api`, write godoc that states its defaults and when to use it.

## Commits and releases

- Commit messages explain *why*, not just *what*.
- CalVer is `vYYYY.MM.MINOR`; releases are created by a `v20*` tag or the
  `workflow_dispatch` trigger in `.github/workflows/release.yml`.
- `CHANGELOG.md` is generated from the local TaskDev notebook:
  ```sh
  taskdev_cli.py changelog --calver=v2026.11
  ```
- `TaskDev/` is intentionally private/gitignored. Do not expect it in a public
  clone; key decisions are summarised in `docs/`.

## Pull request checklist

- [ ] `make check`, `make consumer-test`, `make parity-check` pass.
- [ ] Prove-It was used for bugfixes/contract changes.
- [ ] Golden and budget unchanged, or updated in the same commit.
- [ ] `README.md`/`docs/` updated for user-facing changes.
- [ ] No secrets, host paths, or consumer identity in the core or logs.
- [ ] No new dependency without justification.
