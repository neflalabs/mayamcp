# Consumer Acceptance Module

This directory is a **separate Go module** (`example.com/mayamcp-consumer`)
that imports only `github.com/neflalabs/mayamcp/api`. It proves that MayaMCP is
genuinely consumable from another module, not just from tests inside the repo.

## Why a separate module

Go's `internal` rule prevents other modules from importing `internal/app` and
`internal/mcpserver`. Tests inside the mayamcp repository cannot prove the
external import path. This module uses `replace ... => ../` so the build stays
local while only `api` is visible.

## Running

```sh
make consumer-test      # from the mayamcp root
# or:
cd consumer && GOWORK=off go test -count=1 ./...
```

`GOWORK=off` prevents the parent `go.work` from taking over module resolution.
CI runs the same target.

## What it verifies

`acceptance_test.go` builds a server with `api.NewWithOptions` and hooks:

- only consumer tools are registered (no `ping`/`describe_server`);
- a custom resource via `RegisterResources`;
- custom middleware via `Middleware`;
- the public activity middleware via `api.ActivityMiddleware`;
- an argument containing `Bearer secret-token` does **not** leak into logs.

If this test fails, the public import path is broken — fix it before adding
features (Stop-the-Line).

## During `make rename`

`make rename MODULE=...` updates `consumer/go.mod` automatically:

- the old `require` is dropped and `require $MODULE v0.0.0` added;
- `replace $MODULE => ../` is set again;
- imports in `acceptance_test.go` are rewritten.

Do not edit `consumer/go.mod` manually after a rename unless you have a
specific reason.
