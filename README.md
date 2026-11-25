# MayaMCP

A transport-agnostic Go starter template **and** importable library for building
[Model Context Protocol](https://modelcontextprotocol.io/) servers. The core
(`internal/mcpserver`) knows nothing about HTTP, stdio, authentication, UI, or
the operating system — the front end chooses a transport at startup.

```sh
go get github.com/neflalabs/mayamcp@main
```

> **Version note:** release tags follow CalVer (`v2026.11.8`) and are used for
> binaries and GitHub Releases. Go modules read `v2026` as a major version, so
> import with `@main` or a commit hash instead of a CalVer tag. See
> [Versioning](#versioning).

## Why "MayaMCP"?

- **Pronunciation.** *MAH-yah* + `MCP`: short, familiar, and easy for
  non-Indonesian speakers to say, while still distinct as a brand.
- **Meaning.** In Indonesian/Sanskrit, *maya* evokes creation, form,
  transformation, and openness. It fits a **genesis / seed** that is not yet
  locked to one domain — ready to become a local tool server, an internal
  service, an API integration, or a domain-specific MCP server.
- **Relationship to NgodingMCP.** `NgodingMCP` was born from a concrete need:
  a sandboxed editor/workspace plus an operations control panel. MayaMCP takes
  the same engineering discipline and MCP patterns and offers them as a
  cleaner, more flexible foundation **without the old appliance identity**.
  NgodingMCP is now the reference consumer; MayaMCP is the source of truth for
  core fixes.

## Design principles

1. **Small core.** Anything reusable by any domain belongs in the core;
   anything domain-specific belongs in the consumer.
2. **Stable contract.** Tool names/schemas and `tools/list` are public API,
   locked by a golden test and a byte budget.
3. **Privacy by construction.** The only built-in observability output is a
   one-line activity record that never carries commands, file contents,
   tokens, or secrets.
4. **One source of truth.** Identity and configuration resolve exactly once
   and feed both `initialize.serverInfo` and `describe_server`.
5. **Low dependency surface.** Go standard library plus the official MCP Go
   SDK.

## Quick start

### 1. Use it as a library (recommended)

```sh
mkdir my-mcp && cd my-mcp
go mod init example.com/my-mcp
go get github.com/neflalabs/mayamcp@main
go mod tidy
```

```go
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neflalabs/mayamcp/api"
)

type host struct{}

func (host) Name() string    { return "my-server" }
func (host) Version() string { return "v1" }

type echoArgs struct {
	Label string `json:"label" jsonschema:"Label to echo"`
}

func echo(_ context.Context, _ *mcp.CallToolRequest, in echoArgs) (*mcp.CallToolResult, string, error) {
	if in.Label == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "label is required"}},
		}, "", nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Label}}}, in.Label, nil
}

func main() {
	srv := api.NewWithOptions(host{}, api.Options{
		Register: func(s *mcp.Server) {
			api.AddTextTool(s, &mcp.Tool{Name: "echo", Description: "Echo a label."}, echo)
		},
	})
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
```

A non-nil `Register` replaces the starter tools (`ping`, `describe_server`).
Leave it out to keep them. More examples live in `api/example_test.go`.

### 2. Use the C-lite core

```go
core := api.NewCore(api.Config{
	Name:    "my-server",
	Version: "v1",
	Register:          registerDomainTools,
	RegisterResources: registerResources,
	Middleware:        []mcp.Middleware{safeErrorMiddleware},
})

// core.Server is ready for stdio, Streamable HTTP, or an in-memory transport.
```

`NewCore` assembles identity, logging, hooks, and the activity middleware in a
defined order. Set `DisableActivityMiddleware: true` when you provide your own
domain-aware middleware.

### 3. Fork the template

```sh
make rename MODULE=github.com/acme/custom-mcp
```

This rewrites `go.mod`, imports, the binary name, `cmd/`, CI, `.gitignore`, and
the `consumer/` module. Forks do not receive core fixes automatically — use the
library path for long-lived consumers.

## Running the example binary

```sh
make build
./mayamcp                                    # stdio (default)
./mayamcp -transport=http -addr=127.0.0.1:8080
./mayamcp -log-level=debug -log-json         # JSON logs for containers/systemd
```

`-log-level` accepts `debug`, `info` (default), `warn`/`warning`, and `error`;
anything else exits with code `2`. HTTP uses Streamable HTTP with
`Stateless: true`. Authentication and TLS are deliberately left to the
embedding application or a reverse proxy.

## Public API at a glance

| Need | Use |
|---|---|
| Build a server with your own tools | `api.NewWithOptions` or `api.NewCore` |
| Reuse identity/logging on your own `mcp.Server` | `api.ResolveIdentity`, `api.NewLogger` |
| Register a text-heavy tool without `outputSchema` | `api.AddTextTool` |
| Register a short JSON confirmation without `outputSchema` | `api.AddUntypedTool` |
| Log tool activity yourself | `api.ActivityMiddleware`, `api.MeasureActivity`, `api.SanitizeTarget` |
| Count tools from a consumer | `api.CountTools` |

`api/` is the only supported import surface; `internal/` cannot be imported
from another module. See [`docs/api-guide.md`](docs/api-guide.md).

## Project layout

```text
api/                    public façade (aliases + delegation only)
internal/mcpserver/     transport-agnostic core: identity, tools, activity log
internal/app/           NewCore builder without a listener
cmd/mayamcp/            transport selector (stdio / Streamable HTTP)
consumer/               separate module importing only api (acceptance test)
docs/                   developer guides (start at docs/README.md)
scripts/parity-check.sh cross-repo tool-contract gate
testdata/parity/        recorded golden hash for ngodingmcp
.github/workflows/      CI and CalVer release
```

`TaskDev/` holds the local architecture notebook and audit trail. It is
intentionally gitignored and not part of this repository.

## Testing & quality

```sh
make test            # go test -count=1 ./...
make check           # gofmt, go vet, staticcheck, go test -race
make consumer-test   # external module, imports only api
make parity-check    # ngodingmcp golden unchanged + both modules build
```

The `tools/list` contract is locked by
`internal/mcpserver/testdata/tools.golden.json` (884-byte pretty file,
601-byte compact payload) with a 2048-byte budget. Regenerate intentionally:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

Bugfixes follow the Prove-It pattern: write the failing reproduction first.
Coverage stays at or above the 87.4% baseline (currently ~96%; the CLI entry
point and transport tests are fully covered except the trivial `os.Exit` wiring
in `main`).

## Versioning

CalVer is `vYYYY.MM.MINOR`. Additions to `api` are allowed without a MINOR bump;
removals or signature changes require one plus a migration note in
[`CHANGELOG.md`](CHANGELOG.md). The tool contract and the `tool: ` activity
prefix never change through a routine `api` change.

Release tags (`v20*`) drive `.github/workflows/release.yml` and binary
artifacts. Module versions use independent semver tags: `v0.1.0` currently
points at the same commit as the `v2026.11.8` release baseline. Because Go
interprets `v2026` as a major-version suffix, module consumers should pin
`@v0.1.0` (once pushed) or track `@main` / a commit hash — never the CalVer tag.

## Documentation

Start with [`docs/README.md`](docs/README.md). The most useful next reads are
[`docs/quickstart.md`](docs/quickstart.md),
[`docs/api-guide.md`](docs/api-guide.md), and
[`docs/observability.md`](docs/observability.md).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`AGENTS.md`](AGENTS.md) for
conventions, quality gates, and the golden-test workflow. To report a
vulnerability privately, see [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE). Use it as a library, fork it, or ship it in a product — keep
the copyright notice.
