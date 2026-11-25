# MayaMCP

Starter template untuk server MCP berbasis Go. Inti (`internal/mcpserver`)
tidak bergantung pada transport, autentikasi, Web UI, atau sistem operasi;
front end memilih transport saat startup.

## Struktur

- `internal/mcpserver` — server, opsi identitas, helper hasil, activity log,
  dan dua tool contoh (`ping`, `describe_server`).
- `internal/app` — `NewCore`: membangun server + domain host tanpa listener.
- `cmd/mayamcp` — pemilih transport: `stdio` (default) atau `http`.

Domain contoh bernama `Host` di `internal/mcpserver/domain.go`. Saat memakai
template ini, ganti `Host` dan `tools_*.go` dengan domain target; berkas
transport/options tidak perlu diubah.

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
