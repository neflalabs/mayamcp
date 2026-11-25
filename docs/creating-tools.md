# Creating Tools in MayaMCP

A short guide to adding a tool to a server built from this template. An
automatically verified example lives in `api/example_test.go`
(`ExampleNew_customTool`).

## 1. Input structure

Define a struct with `json` tags; the `jsonschema` tag becomes the property
description in `inputSchema`:

```go
type echoArgs struct {
	Label string `json:"label" jsonschema:"Text to echo"`
}
```

`mcp.AddTool` infers the schema from the `In` type, including `required`.
Avoid `any` as an input type: its schema is an empty object and clients will
not know which arguments are available.

## 2. Domain errors vs protocol errors

- **Domain/validation error** → return a `*mcp.CallToolResult` with
  `IsError: true`. The model sees the message and can correct its call.
- **Protocol/transport error** → return `nil, err`. Use this for system
  failures the model cannot fix.

```go
return &mcp.CallToolResult{
	IsError: true,
	Content: []mcp.Content{&mcp.TextContent{Text: "label is required"}},
}, nil, nil
```

Never put host paths, tokens, or secret data in an error message.

## 3. Choosing a registration helper

| Need | Use |
|---|---|
| Large text results (files, listings, logs) | `api.AddTextTool` — `content` only, no `outputSchema` |
| Short JSON confirmations for programmatic clients | `api.AddUntypedTool` — `structuredContent`, no `outputSchema` |
| Small records the client really needs | `mcp.AddTool` with a typed `Out` → `outputSchema` + `structuredContent` |

From outside the module, import `github.com/neflalabs/mayamcp/api`:
`api.NewWithOptions(...)` returns an `*mcp.Server`. Supply `Options.Register`
to replace the starter tools, then register yours with `api.AddTextTool`,
`api.AddUntypedTool`, or `mcp.AddTool` when an `outputSchema` is genuinely
needed.

## 4. Preserving the built-in contract

The contract is locked by
`internal/mcpserver/testdata/tools.golden.json` (pretty file **884 bytes**).
`TestToolsListBudget` measures the compact `tools/list` payload at **601 bytes**
against a 2048-byte budget. These are two different numbers, so:

- do **not** register examples and experiments on the built-in server;
- write them as `Example...` functions in `*_test.go` or as documentation;
- if a tool genuinely belongs in the template, update the golden file in the
  same commit:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

## 5. Activity log

Targets come from the generic keys `name`, `path`, `id`, `target`, and
`from`/`to`. For domain-specific keys (for example git `ref`/`remote`), supply
a `TargetExtractor` through `api.Options` / `app.Config`. Its output is always
truncated and sanitized by the middleware, but the extractor itself must never
return a secret. See [`observability.md`](observability.md).
