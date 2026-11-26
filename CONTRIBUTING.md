# Contributing to MayaMCP

MayaMCP is a template: keep the core small, the contract stable, and every
change backed by a test.

## Before you start

1. Read [`AGENTS.md`](AGENTS.md) for conventions (golden tests, identity,
   core vs application boundary).
2. Read [`docs/README.md`](docs/README.md) for the documentation map.
3. Install Go 1.26+ and run `make check`. The baseline must be green before you
   add anything — fix failures before adding features.

## Workflow

```sh
make test            # fast iteration
make check           # gofmt, vet, staticcheck, go test -race
make consumer-test   # external import path must keep working
```

If `make check` fails, fix the root cause before adding features.

## Failing-test-first

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

| Belongs in the core | Belongs in the consuming application |
|---|---|
| `Host` interface, identity + defaults, logger, activity middleware | Auth/OAuth, terminal UI, web UI, service manager, config store |
| `api/` façade, `ping`/`describe_server`, golden + budget tests | Domain tools, filesystem, workers, custom resources |

`internal/` cannot be imported from another module — everything public goes
through `api/`. See [`docs/api-guide.md`](docs/api-guide.md).

## Fuzzing and benchmarks

```sh
go test ./internal/mcpserver -run Fuzz          # seed corpus
go test ./internal/mcpserver -fuzz FuzzActivityTarget -fuzztime=30s
go test ./internal/mcpserver -run '^$' -bench . -benchmem
```

The fuzz targets assert that sanitized targets stay bounded and free of
control characters. Benchmarks cover `MeasureActivity` and `activityTarget`;
use them before changing the hot path in the activity middleware.

## User-facing changes

Update `README.md`/`docs/` whenever behaviour changes. When adding an exported
function to `api`, write godoc that states its defaults and when to use it.

## Commits and releases

- Commit messages explain *why*, not just *what*.
- Module versions are semver `v0.MINOR.PATCH`; releases are created by a
  `v0.*` tag or `workflow_dispatch` in `.github/workflows/release.yml`.
  Older `v2026.11.x` CalVer labels were removed from the remote; module versions
  are semver only.
- `CHANGELOG.md` records published releases; add an entry when behaviour or
  compatibility changes.

## Pull request checklist

- [ ] `make check` and `make consumer-test` pass.
- [ ] A failing reproduction was written first for bugfixes/contract changes.
- [ ] Golden and budget unchanged, or updated in the same commit.
- [ ] `README.md`/`docs/` updated for user-facing changes.
- [ ] No secrets, host paths, or downstream identity in the core or logs.
- [ ] No new dependency without justification.
