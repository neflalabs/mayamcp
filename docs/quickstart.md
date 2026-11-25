# Quickstart

Dari kosong sampai server MCP berjalan dalam beberapa menit. Prasyarat: Go 1.26+.

## Jalur 1 — Impor pustaka (disarankan)

Modul kanonik `github.com/neflalabs/mayamcp`. Rilis biner terbaru
`v2026.11.8`; untuk Go modules pakai `@main` atau hash commit, bukan tag CalVer
(karena `v2026` terbaca sebagai major version).

```sh
mkdir my-mcp && cd my-mcp
go mod init example.com/my-mcp
go get github.com/neflalabs/mayamcp@main
go mod tidy
```

Server minimal dengan satu tool sendiri (starter tool tidak ikut):

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

`Register` bukan `nil` menggantikan `ping`/`describe_server`. Bila ingin tetap
memakai starter tool, serahkan `api.Options{}` tanpa `Register`. Detail di
[`api-guide.md`](api-guide.md).

Contoh lain yang diverifikasi `go test ./...` ada di `api/example_test.go`.

## Jalur 2 — Lifecycle C-lite lewat `NewCore`

Gunakan ini bila resource, middleware, dan identitas ingin dirakit sekali:

```go
core := api.NewCore(api.Config{
	Name:    "my-server",
	Version: "v1",
	Register: func(s *mcp.Server) { /* tool domain */ },
	RegisterResources: func(s *mcp.Server) {
		// readNotes: func(context.Context, *mcp.ReadResourceRequest)
		//   (*mcp.ReadResourceResult, error)
		s.AddResource(&mcp.Resource{Name: "notes", URI: "memo://notes"}, readNotes)
	},
	Middleware: []mcp.Middleware{safeErrorMiddleware, holdWorkspaceMiddleware},
})

// core.Server siap diserve lewat stdio atau Streamable HTTP.
```

`core.Logger` memakai `slog` (text/JSON sesuai `LogJSON`). Activity middleware
dipasang otomatis; set `DisableActivityMiddleware: true` bila Anda memasang
middleware sendiri yang sudah memahami data domain.

## Jalur 3 — Fork/prototipe

```sh
make rename MODULE=github.com/acme/custom-mcp
```

Perintah ini memperbarui `go.mod`, import di `*.go`/`*.md`, binary, direktori
`cmd/`, CI, `.gitignore`, **dan** `consumer/go.mod`. Perbaikan inti tidak ikut
otomatis ke fork — untuk konsumsi jangka panjang pilih Jalur 1.

## Menjalankan template ini langsung

```sh
make build
./mayamcp                                   # stdio (default)
./mayamcp -transport=http -addr=127.0.0.1:8080
./mayamcp -log-level=debug -log-json        # JSON log untuk container/systemd
./mayamcp -transport=http -addr=:8080       # ⚠️ warning: bind non-loopback tanpa auth/TLS
```

`-log-level` menerima `debug`, `info` (default), `warn`/`warning`, `error`.
Nilai lain keluar dengan kode `2`.

## Pengembangan lintas repo

`ngodingmcp` memakai mayamcp sebagai source. Siapkan workspace lokal:

```go
// /home/nefla/Devel/mcpserver/go.work (tidak di-commit)
go 1.26.10

use (
	./mayamcp
	./ngodingmcp
)
```

Atau tambahkan `replace github.com/neflalabs/mayamcp => ../mayamcp` ke `go.mod`
konsumen. `go.work` sengaja di-`.gitignore` karena bersifat lokal.

## Verifikasi

```sh
make check          # gofmt + vet + staticcheck + go test -race
make consumer-test  # modul eksternal, hanya import api
make parity-check   # golden ngodingmcp tidak berubah + dua repo build
```

## Troubleshooting

| Gejala | Penyebab / solusi |
|---|---|
| `invalid version: module path must match major version (".../v2026")` | Tag CalVer tidak bisa dipakai `go get`; pakai `@main` atau hash commit |
| `package ... is not in std` / modul tidak ketemu | Lupa `go get` atau `replace`; jalankan `go mod tidy` |
| `AddTool "x": missing input schema` | Pakai `mcp.AddTool` generik, bukan `s.AddTool` manual; atau `api.AddTextTool` |
| `tools/list` tiba-tiba membengkak | Tool baru membawa `outputSchema`; pilih `api.AddTextTool`/`api.AddUntypedTool` |
| `TestToolContract` gagal | Golden perlu diperbarui sengaja: `go test ./internal/mcpserver -run TestToolContract -args -update`, lalu review diff |
| Tidak ada baris activity | Default level `info`; set `-log-level=debug` atau `LogLevel` di Config, atau Anda menyetel `DisableActivityMiddleware` |
| Activity log menampilkan `? → ?` | Argumen `from`/`to` bukan URL; helper `hostOrSafeValue` seharusnya memakai nilai lokal — laporkan regresi |
| `make rename` gagal "module path unchanged" | `MODULE` sama dengan module saat ini; gunakan path lain |
| Consumer test rusak setelah rename | Jalankan ulang `make rename` versi terbaru; kini `consumer/go.mod` ikut diperbarui |
