# AGENTS.md

Panduan singkat untuk AI yang bekerja di repo template ini.

## Struktur

- `internal/mcpserver` — inti transport-agnostic; satu file per domain tool.
- `internal/app` — `NewCore`, konstruksi server tanpa transport.
- `cmd/mayamcp` — transport stdio (default) atau HTTP stateless.

## Konvensi

- Utamakan Go stdlib; dependency baru harus bisa dibenarkan.
- Opsi identitas (`Name`, `Title`, `Instructions`, `Version`) harus punya default;
  jangan hardcode identitas konsumen di dalam inti.
- Kontrak tool dikunci golden test. Perubahan kontrak: perbarui
  `testdata/tools.golden.json` dengan
  `go test ./internal/mcpserver -run TestToolContract -args -update`, lalu review diff.
- Jangan menaikkan budget `tools/list` tanpa keputusan sadar.
- Tool dengan hasil teks besar memakai `addTextTool`; `outputSchema` lewat
  `mcp.AddTool` hanya untuk record kecil yang memang dibutuhkan client.
  Pola ini diwariskan dari `ngodingmcp` (lihat `docs/mcp-as-template.md`).
- Perubahan perilaku pengguna harus disertai update `README.md` / `docs/`.
- `TaskDev/README.md` berisi nama proyek `MayaMCP`. CLI taskdev punya fallback
  hardcoded `Nefla POS`, jadi setiap selesai `taskdev-cli sync` (terutama setelah
  regenerate dari nol) wajib cek baris `proyek **MayaMCP**` dan koreksi manual
  bila berubah. Jangan anggap CLI mendeteksi nama dari repo.

## Perintah

```sh
make test        # go test -count=1 ./...
make check       # gofmt, vet, staticcheck, go test -race
make build       # binary ./mayamcp
```
