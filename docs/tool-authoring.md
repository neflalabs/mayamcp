# Menulis Tool untuk Server Domain

Panduan ini melengkapi [`creating-tools.md`](creating-tools.md) dengan contoh
rakit-server-penuh. Untuk satu tool tambahan, cukup baca dokumen itu.

## Kerangka lengkap

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
		// Domain error: model can correct the call.
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required"}},
		}, readResult{}, nil
	}
	body, err := os.ReadFile(in.Path)
	if err != nil {
		// Protocol error: infrastructure problem, not a model mistake.
		return nil, readResult{}, err
	}

	// Tipe kecil yang benar-benar dipakai klien → mcp.AddTool.
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
			return in.Path // jangan pernah kembalikan isi file/token
		},
		Register: func(s *mcp.Server) {
			// Teks besar → AddTextTool (tanpa outputSchema).
			api.AddTextTool(s, &mcp.Tool{Name: "dump", Description: "Dump raw text."},
				func(ctx context.Context, _ *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, string, error) {
					body, err := os.ReadFile(in.Path)
					if err != nil {
						return nil, "", err
					}
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, "", nil
				})

			// Record kecil → mcp.AddTool (dengan outputSchema).
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

`readGuide` di atas adalah placeholder — sesuaikan dengan domain
Anda. Yang penting adalah **pemisahan peran**:

| Bagian | Milik |
|---|---|
| Tool & `TargetExtractor` | Konsumen/domain |
| Identitas, logger, activity middleware, hook | MayaMCP core |
| Transport, auth, UI, config store | Aplikasi |

## Memilih helper

| Situasi | Helper |
|---|---|
| Output teks besar (file, log, listing, diff) | `api.AddTextTool` |
| Konfirmasi pendek tapi dibaca program | `api.AddUntypedTool` |
| Record kecil yang skemanya dipakai klien | `mcp.AddTool` dengan `Out` typed |

`outputSchema` dibayar **di setiap session** lewat `tools/list`, jangan
ditambahkan hanya karena "mungkin berguna".

## Error: domain vs protokol

- `IsError: true` + `nil error` → model melihat pesan dan bisa memperbaiki
  panggilan. Pakai untuk validasi input dan keadaan domain.
- `nil, err` → kegagalan sistem/transport. Model tidak bisa memperbaikinya.
- Jangan pernah memasukkan path absolut host, token, atau secret ke pesan error.

## Menguji tool

Pakai transport in-memory agar tidak menyentuh stdio/HTTP:

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

Contoh nyata: `api/toolkit_test.go` dan `consumer/acceptance_test.go`.

## Kontrak bawaan

- Golden `internal/mcpserver/testdata/tools.golden.json` = file pretty **884 byte**.
- `TestToolsListBudget` mengukur payload compact **601 byte** dari budget 2048.
- Tool **contoh** tidak boleh didaftarkan ke server bawaan; tulis sebagai
  `Example...` di `*_test.go` atau dokumentasi.
- Bila tool baru memang masuk template, regenerasi golden di commit yang sama:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

## Checklist sebelum selesai

- [ ] Input memakai struct + tag `json`/`jsonschema`, bukan `any`.
- [ ] Error domain dan error protokol dibedakan.
- [ ] Helper dipilih sesuai ukuran payload; `outputSchema` disengaja.
- [ ] `TargetExtractor` tidak mengembalikan secret; diuji dengan payload jahat.
- [ ] Test in-memory untuk `ListTools` dan `CallTool`.
- [ ] `make check` hijau; golden/budget tidak berubah kecuali disengaja.
- [ ] Perubahan perilaku pengguna diiringi update `README.md`/`docs/`.
