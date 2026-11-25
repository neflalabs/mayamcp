# AGENTS.md

Panduan singkat untuk AI yang bekerja di repo template ini.

## Struktur

- `internal/mcpserver` — inti transport-agnostic; satu file per domain tool.
- `internal/app` — `NewCore`, konstruksi server tanpa transport.
- `cmd/mayamcp` — transport stdio (default) atau HTTP stateless.

## Konvensi

- Utamakan Go stdlib; dependency baru harus bisa dibenarkan.
- Opsi identitas (`Name`, `Title`, `Instructions`, `MetaNamespace`) harus punya
  default; jangan hardcode identitas konsumen di dalam inti.
- Kontrak tool dikunci golden test. Perubahan kontrak: perbarui
  `testdata/tools.golden.json` dengan
  `go test ./internal/mcpserver -run TestToolContract -args -update`, lalu review diff.
- Jangan menaikkan budget `tools/list` tanpa keputusan sadar.

## Perintah

```sh
make test        # go test -count=1 ./...
make check       # gofmt, vet, staticcheck, go test -race
make build       # binary ./mayamcp
```
