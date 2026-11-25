# Writing Tools for a Domain Server

This guide extends [`creating-tools.md`](creating-tools.md) with a
full-server example. For a single extra tool, read that document instead.

## Complete skeleton

```go
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neflalabs/mayamcp/api"
)

type host struct{}

func (host) Name() string    { return "files-server" }
func (host) Version() string { return "v1" }

type readArgs struct {
	Path string `json:"path" jsonschema:"File path relative to root"`
}

type readResult struct {
	Bytes int `json:"bytes"`
}

func readFile(ctx context.Context, req *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, readResult, error) {
	if in.Path == "" {
		// Domain error: the model can correct the call.
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required"}},
		}, readResult{}, nil
	}
	body, err := os.ReadFile(in.Path)
	if err != nil {
		// Protocol error: an infrastructure problem the model cannot fix.
		return nil, readResult{}, err
	}

	// Small record the client really uses → mcp.AddTool.
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}},
		readResult{Bytes: len(body)}, nil
}

func main() {
	logger := api.NewLogger(api.LogOptions{Writer: os.Stderr, Level: slog.LevelInfo})

	srv := api.NewWithOptions(host{}, api.Options{
		Logger: logger,
		TargetExtractor: func(_ string, args json.RawMessage) string {
			var in readArgs
			if json.Unmarshal(args, &in) != nil {
				return ""
			}
			return in.Path // never return file contents or tokens
		},
		Register: func(s *mcp.Server) {
			// Large text → AddTextTool (no outputSchema).
			api.AddTextTool(s, &mcp.Tool{Name: "dump", Description: "Dump raw text."},
				func(ctx context.Context, _ *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, string, error) {
					body, err := os.ReadFile(in.Path)
					if err != nil {
						return nil, "", err
					}
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, "", nil
				})

			// Small record → mcp.AddTool (with outputSchema).
			mcp.AddTool(s, &mcp.Tool{Name: "read", Description: "Read a file."}, readFile)
		},
		RegisterResources: func(s *mcp.Server) {
			s.AddResource(&mcp.Resource{Name: "guide", URI: "memo://guide"}, readGuide)
		},
	})

	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		panic(err)
	}
}
```

`readGuide` above is a placeholder — adapt it to your domain. What matters is
the **separation of roles**:

| Part | Owner |
|---|---|
| Tools and `TargetExtractor` | Consumer / domain |
| Identity, logger, activity middleware, hooks | MayaMCP core |
| Transport, auth, UI, config store | Application |

## Choosing a helper

| Situation | Helper |
|---|---|
| Large text output (files, logs, listings, diffs) | `api.AddTextTool` |
| Short confirmation read by programs | `api.AddUntypedTool` |
| Small record whose schema clients use | `mcp.AddTool` with typed `Out` |

An `outputSchema` is paid on **every session** through `tools/list`; do not add
one just because it "might be useful".

## Errors: domain vs protocol

- `IsError: true` with a `nil` error → the model sees the message and can fix
  the call. Use it for input validation and domain states.
- `nil, err` → system/transport failure. The model cannot fix it.
- Never include absolute host paths, tokens, or secrets in an error message.

## Testing tools

Use an in-memory transport so no stdio/HTTP port is touched:

```go
serverT, clientT := mcp.NewInMemoryTransports()
ctx := context.Background()
if _, err := srv.Connect(ctx, serverT, nil); err != nil {
	t.Fatal(err)
}
client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
session, err := client.Connect(ctx, clientT, nil)
if err != nil {
	t.Fatal(err)
}
defer session.Close()

tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
// ...
res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "read", Arguments: map[string]any{"path": "x.txt"}})
// ...
```

Real examples: `api/toolkit_test.go` and `consumer/acceptance_test.go`.

## Built-in contract

- Golden `internal/mcpserver/testdata/tools.golden.json` = pretty **884 bytes**.
- `TestToolsListBudget` measures the compact payload at **601 bytes** against a
  2048-byte budget.
- Example tools must **not** be registered on the built-in server; write them
  as `Example...` functions in `*_test.go` or as documentation.
- If a new tool genuinely belongs in the template, regenerate the golden in the
  same commit:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

## Checklist

- [ ] Input uses a struct with `json`/`jsonschema` tags, not `any`.
- [ ] Domain errors and protocol errors are distinguished.
- [ ] Helper chosen by payload size; `outputSchema` is intentional.
- [ ] `TargetExtractor` returns no secret; tested with hostile input.
- [ ] In-memory tests for `ListTools` and `CallTool`.
- [ ] `make check` passes; golden/budget unchanged unless intended.
- [ ] User-facing changes are reflected in `README.md`/`docs/`.
