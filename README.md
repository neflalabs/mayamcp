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
- `docs/` — indeks, quickstart, panduan API, observability, dan authoring tool
  (mulai dari [`docs/README.md`](docs/README.md)).
- `consumer/` — modul terpisah yang hanya mengimpor `api` untuk acceptance test.
- `scripts/parity-check.sh` + `testdata/parity/` — gate golden lintas repo.
- `TaskDev/` — guideline arsitektur dan histori keputusan; **lokal saja**
  (di-`.gitignore`, tidak ikut ter-clone).
- `.github/workflows/ci.yml` — check, consumer test, parity gate, build, cross-compile.
- `.github/workflows/release.yml` — rilis CalVer + artifact.

Domain contoh bernama `Host` di `internal/mcpserver/domain.go`. Saat memakai
template ini, ganti `Host` dan `tools_*.go` dengan domain target; berkas
transport/options tidak perlu diubah.

## Dua cara memakai

Ada dua mode konsumsi:

1. **Impor `github.com/neflalabs/mayamcp/api` (jalur kanonik).** Package `api`
   adalah fasad publik yang membungkus `internal/app` dan `internal/mcpserver`
   (yang tidak bisa diimpor langsung dari modul lain karena aturan `internal`
   Go). Helper `ResolveIdentity`, `NewLogger`, `ActivityMiddleware`,
   `AddTextTool`, dan `AddUntypedTool` bisa dipakai satu per satu, atau lewat
   `api.NewCore`/`api.NewWithOptions` bila lifecycle juga diambil.
2. **Salin template lalu edit `internal/` (fork/prototipe).** `make rename
   MODULE=...` meng-instansiasi template di module path lain. Jalur ini tetap
   tersedia, tetapi perbaikan inti tidak otomatis ikut — pilih impor untuk
   konsumsi jangka panjang.

Repo publiknya `github.com/neflalabs/mayamcp`; rilis terbaru `v2026.11.8`
(artefak dan catatan GitHub Release).

**Catatan Go modules:** tag CalVer `v2026.11.8` dibaca Go sebagai *major version*
`v2026`, sehingga `go get ...@v2026.11.8` ditolak. Untuk mengimpor pakai
branch/commit yang menghasilkan pseudo-version:

```sh
go get github.com/neflalabs/mayamcp@main
go mod tidy
```

Untuk versi terpin, pakai hash commit, mis. `@af55bb9`. Tag `v20*` tetap dipakai
untuk rilis biner, bukan versi modul Go.

Untuk pengembangan lintas repo (mayamcp + ngodingmcp), pakai `go.work` di
direktori induk atau `replace github.com/neflalabs/mayamcp => ../mayamcp`.
Panduan lengkap ada di [`docs/quickstart.md`](docs/quickstart.md) dan
[`docs/api-guide.md`](docs/api-guide.md).

Contoh pemakaian sebagai pustaka:

```go
import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neflalabs/mayamcp/api"
)

type host struct{}

func (host) Name() string    { return "my-server" }
func (host) Version() string { return "v1" }

func serve() error {
	// Versi diambil dari host.Version(); tidak perlu ditulis dua kali.
	srv := api.NewWithOptions(host{}, api.Options{})
	return srv.Run(context.Background(), &mcp.StdioTransport{})
}
```

Untuk mengganti starter tool `ping`/`describe_server`, berikan `Register`:

```go
srv := api.NewWithOptions(host{}, api.Options{
	Register: func(s *mcp.Server) {
		api.AddTextTool(s, &mcp.Tool{Name: "echo", Description: "Echo a label."}, echoHandler)
	},
})
```

Fasad ini hanya berisi alias tipe dan fungsi delegasi; tidak ada logika yang
ditulis ulang. Field baru pada `app.Config` otomatis tersedia lewat
`api.Config`.

## Menjalankan

```sh
make build
./mayamcp                         # stdio, untuk client lokal
./mayamcp -transport=http -addr=127.0.0.1:8080
./mayamcp -log-level=debug -log-json     # log JSON untuk container/systemd
```

`-log-level` menerima `debug`, `info` (default), `warn`/`warning`, atau `error`;
nilai lain menyebabkan exit code `2`.

HTTP memakai Streamable HTTP dengan `Stateless: true`. Autentikasi/TLS
diserahkan ke aplikasi atau reverse proxy yang menanam template ini.

## Kompatibilitas & versi

CalVer `vYYYY.MM.MINOR`. Paket `api` adalah satu-satunya kontrak publik:
penambahan fungsi/field boleh tanpa kenaikan MINOR; penghapusan atau perubahan
tanda tangan hanya pada MINOR dengan catatan migrasi di `CHANGELOG.md`. Kontrak
tool (`tools.golden.json`) dan format `ActivityPrefix` tidak berubah lewat
perubahan `api` biasa.

## Test

```sh
make test            # go test -count=1 ./...
make check           # gofmt, vet, staticcheck, go test -race
make consumer-test   # modul konsumen eksternal (hanya import api)
make parity-check    # golden ngodingmcp + build dua repo (lewati bila tak ada)
make release VERSION=v2026.11.8   # artifact lintas platform
make rename MODULE=github.com/acme/custom-mcp   # fork/prototipe
```

Kontrak tool dikunci oleh `internal/mcpserver/testdata/tools.golden.json`.
Setelah mengubah tool secara sengaja:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

Anggaran `tools/list` dijaga `TestToolsListBudget`.
