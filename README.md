# MayaMCP

Starter template untuk server MCP berbasis Go. Inti (`internal/mcpserver`)
tidak bergantung pada transport, autentikasi, Web UI, atau sistem operasi;
front end memilih transport saat startup.

## Latar & rujukan upstream

MayaMCP diekstrak dari `ngodingmcp`. Catatan asli proses ekstraksi — filosofi
nama, batas inti vs *appliance*, inventaris identitas yang perlu diganti, dan
pertanyaan terbuka soal *source-of-truth* antar dua repo — disimpan di
[`docs/mcp-as-template.md`](docs/mcp-as-template.md). Dokumen itu disalin apa
adanya dari `ngodingmcp/mcp-as-template.md`, jadi rujukan commit/TaskDev di
dalamnya merujuk ke upstream.

## Struktur

- `internal/mcpserver` — server, opsi identitas, helper hasil, activity log,
  dan dua tool contoh (`ping`, `describe_server`).
- `internal/app` — `NewCore`: membangun server + domain host tanpa listener.
- `cmd/mayamcp` — pemilih transport: `stdio` (default) atau `http`.
- `docs/mcp-as-template.md` — catatan upstream tentang ekstraksi template.
- `docs/creating-tools.md` — panduan membuat tool berparameter.
- `TaskDev/` — guideline arsitektur dan histori keputusan; **lokal saja**
  (di-`.gitignore`, tidak ikut ter-clone).
- `.github/workflows/ci.yml` — `make check`, `go mod verify`, build, cross-compile.

Domain contoh bernama `Host` di `internal/mcpserver/domain.go`. Saat memakai
template ini, ganti `Host` dan `tools_*.go` dengan domain target; berkas
transport/options tidak perlu diubah.

## Dua cara memakai

Ada dua mode konsumsi:

1. **Salin template lalu edit `internal/`.** Kloning repo ini, ganti domain
   `Host` dan tool sesuai kebutuhan, lalu jalankan lewat `cmd/mayamcp`. Cocok
   bila Anda ingin mengubah perilaku inti server.
2. **Impor `mayamcp/api` sebagai pustaka.** Package `api` adalah fasad publik
   yang membungkus `internal/app` dan `internal/mcpserver` (yang tidak bisa
   diimpor langsung dari modul lain karena aturan `internal` Go). Cocok bila
   Anda ingin memakai server ini apa adanya dari modul Anda sendiri.

Contoh pemakaian sebagai pustaka:

```go
import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mayamcp/api"
)

type host struct{}

func (host) Name() string    { return "my-server" }
func (host) Version() string { return "v1" }

func serve() error {
	srv := api.New(host{}, "v1", api.Options{})
	return srv.Run(context.Background(), &mcp.StdioTransport{})
}
```

Fasad ini hanya berisi alias tipe dan fungsi delegasi; tidak ada logika yang
ditulis ulang. Field baru pada `app.Config` otomatis tersedia lewat
`api.Config`.

## Menjalankan

```sh
make build
./mayamcp                         # stdio, untuk client lokal
./mayamcp -transport=http -addr=127.0.0.1:8080
```

HTTP memakai Streamable HTTP dengan `Stateless: true`. Autentikasi/TLS
diserahkan ke aplikasi atau reverse proxy yang menanam template ini.

## Test

```sh
make test
make check
```

Kontrak tool dikunci oleh `internal/mcpserver/testdata/tools.golden.json`.
Setelah mengubah tool secara sengaja:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

Anggaran `tools/list` dijaga `TestToolsListBudget`.
