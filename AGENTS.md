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
  `testdata/tools.golden.json` (file pretty **884 byte**) dengan
  `go test ./internal/mcpserver -run TestToolContract -args -update`, lalu review diff.
  Anggaran `TestToolsListBudget` mengukur `tools/list` compact (**601 byte** / 2048),
  bukan ukuran file golden.
- Jangan menaikkan budget `tools/list` tanpa keputusan sadar.
- Tool dengan hasil teks besar memakai `addTextTool`; `outputSchema` lewat
  `mcp.AddTool` hanya untuk record kecil yang memang dibutuhkan client.
  Pola ini diwariskan dari `ngodingmcp` (arsip lokal:
  `TaskDev/Note/MCP_AS_TEMPLATE.md`).
- Perubahan perilaku pengguna harus disertai update `README.md` / `docs/`.
- `TaskDev/README.md` berisi nama proyek `MayaMCP`. CLI taskdev punya fallback
  hardcoded `Nefla POS`, jadi setiap selesai `taskdev_cli.py sync` (terutama
  setelah regenerate dari nol) wajib cek baris `proyek **MayaMCP**` dan koreksi
  manual bila berubah. Jangan anggap CLI mendeteksi nama dari repo.
- `TaskDev/` bersifat privat (di-gitignore), jadi validasi notebook dijalankan
  lokal lewat `make taskdev-validate`, bukan di CI.

## Perintah

```sh
make test             # go test -count=1 ./...
make check            # gofmt, vet, staticcheck, go test -race
make build            # binary ./mayamcp
make consumer-test    # modul eksternal yang hanya import api
make parity-check     # golden ngodingmcp + build dua repo
make release          # artifact rilis lintas platform
make rename           # instansiasi template di module path lain
make taskdev-validate # taskdev_cli.py validate --strict (lokal)
make taskdev-sync     # regenerate indeks TaskDev + jaga nama MayaMCP
```
