# Dokumentasi MayaMCP

Indeks panduan. Mulai dari baris pertama yang cocok dengan kebutuhan Anda.

| Dokumen | Cocok untuk | Isi |
|---|---|---|
| [`quickstart.md`](quickstart.md) | "Saya mau server jalan sekarang" | `go get`, tiga jalur konsumsi, contoh stdio/HTTP, troubleshooting |
| [`api-guide.md`](api-guide.md) | "Saya mau pakai sebagai pustaka" | Peta export `api`, pilihan `New`/`NewWithOptions`/`NewCore`, hook, identitas |
| [`creating-tools.md`](creating-tools.md) | "Saya mau tambah satu tool" | Input schema, error domain vs protokol, helper registrasi, golden |
| [`tool-authoring.md`](tool-authoring.md) | "Saya bangun server domain penuh" | Contoh end-to-end dengan hook, resource, middleware, dan test |
| [`observability.md`](observability.md) | "Saya konsumsi/mengubah activity log" | Format `tool: `, suffix token, atribut slog, jaminan anti-secret |
| [`mcp-as-template.md`](mcp-as-template.md) | "Kenapa repositori ini dibuat" | Arsip proses ekstraksi dari `ngodingmcp` (rujukan upstream) |

## Di luar folder ini

- [`README.md`](../README.md) — ringkasan proyek, perintah `make`, dan kontrak versi.
- [`AGENTS.md`](../AGENTS.md) — konvensi untuk agen dan kontributor.
- [`CHANGELOG.md`](../CHANGELOG.md) — catatan rilis CalVer.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — alur kontribusi dan quality gate.
- `TaskDev/` — notebook arsitektur dan audit; **lokal** (ter-`.gitignore`), jadi
  tautan ke sana tidak akan berfungsi di kloning publik. Keputusan penting sudah
  diringkas di dokumen-dokumen di atas.
- `consumer/` — modul nyata yang hanya mengimpor `api` (lihat
  [`consumer/README.md`](../consumer/README.md)).

## Aturan singkat yang sering terlewat

- Kontrak publik adalah paket `api`; `internal/` tidak bisa diimpor dari modul lain.
- `tools/list` dikunci golden (**884 byte** file, **601 byte** payload compact)
  dengan budget 2048 — jangan naikkan tanpa keputusan sadar.
- `ActivityPrefix = "tool: "` adalah kontrak konsumen (TUI).
- Bugfix memakai Prove-It: test merah dulu, lalu perbaikan.
