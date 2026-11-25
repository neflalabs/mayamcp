# Quickstart

From zero to a running MCP server in a few minutes. Requires Go 1.26+.

## Path 1 — Import the library (recommended)

Canonical module: `github.com/neflalabs/mayamcp`. The latest binary release is
`v2026.11.8`; for Go modules use `@main` or a commit hash, not the CalVer tag
(because `v2026` is read as a major version).

```sh
mkdir my-mcp && cd my-mcp
go mod init example.com/my-mcp
go get github.com/neflalabs/mayamcp@main
go mod tidy
```

A minimal server with one tool of your own (starter tools are not included):

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

A non-nil `Register` replaces `ping`/`describe_server`. Pass plain
`api.Options{}` to keep the starter tools. More examples live in
`api/example_test.go`.

## Path 2 — C-lite lifecycle with `NewCore`

Use this when identity, resources, and middleware should be assembled once:

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

`core.Logger` uses `slog` (text or JSON depending on `LogJSON`). The activity
middleware is installed automatically; set `DisableActivityMiddleware: true`
when you provide your own domain-aware middleware.

## Path 3 — Fork a prototype

```sh
make rename MODULE=github.com/acme/custom-mcp
```

This updates `go.mod`, imports in `*.go`/`*.md`, the binary name, `cmd/`, CI,
`.gitignore`, **and** `consumer/go.mod`. Core fixes do not reach a fork
automatically — choose Path 1 for long-lived consumers.

## Running this template directly

```sh
make build
./mayamcp                                    # stdio (default)
./mayamcp -transport=http -addr=127.0.0.1:8080
./mayamcp -log-level=debug -log-json         # JSON logs for containers/systemd
./mayamcp -transport=http -addr=:8080        # ⚠️ warns: non-loopback bind without auth/TLS
```

`-log-level` accepts `debug`, `info` (default), `warn`/`warning`, and `error`.
Anything else exits with code `2`.

## Cross-repo development

`ngodingmcp` uses mayamcp as its source. Set up a local workspace:

```go
// /home/nefla/Devel/mcpserver/go.work (not committed)
go 1.26.10

use (
	./mayamcp
	./ngodingmcp
)
```

Or add `replace github.com/neflalabs/mayamcp => ../mayamcp` to the consumer's
`go.mod`. `go.work` is gitignored because it is machine-local.

## Verification

```sh
make check          # gofmt + vet + staticcheck + go test -race
make consumer-test  # external module, imports only api
make parity-check   # ngodingmcp golden unchanged + both modules build
```

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `invalid version: module path must match major version (".../v2026")` | CalVer tags cannot be used with `go get`; use `@main` or a commit hash |
| `package ... is not in std` / module not found | Missing `go get` or `replace`; run `go mod tidy` |
| `AddTool "x": missing input schema` | Use generic `mcp.AddTool`, not manual `s.AddTool`; or `api.AddTextTool` |
| `tools/list` suddenly grows | The new tool carries an `outputSchema`; prefer `api.AddTextTool`/`api.AddUntypedTool` |
| `TestToolContract` fails | The golden file needs an intentional update: `go test ./internal/mcpserver -run TestToolContract -args -update`, then review the diff |
| No activity lines | Default level is `info`; set `-log-level=debug` or `LogLevel` in Config, or you enabled `DisableActivityMiddleware` |
| Activity log shows `? → ?` | `from`/`to` arguments are not URLs; `hostOrSafeValue` should keep local values — report a regression |
| `make rename` says "module path unchanged" | `MODULE` equals the current path; choose another |
| Consumer test breaks after rename | Re-run the latest `make rename`; it now updates `consumer/go.mod` |
