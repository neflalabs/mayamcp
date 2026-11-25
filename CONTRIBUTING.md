# Berkontribusi ke MayaMCP

MayaMCP adalah template transport-agnostic. Aturan utamanya sederhana: **inti
tetap kecil, kontrak tetap stabil, perubahan dibuktikan dengan test.**

## Sebelum mulai

1. Baca [`AGENTS.md`](AGENTS.md) untuk konvensi (golden test, identitas, batas
   core vs appliance, TaskDev lokal).
2. Baca [`docs/README.md`](docs/README.md) untuk peta dokumentasi.
3. Pasang Go 1.26+ dan jalankan `make check` — baseline harus hijau sebelum
   menambah apa pun (Stop-the-Line).

## Alur kerja

```sh
make test            # iterasi cepat
make check           # gofmt, vet, staticcheck, go test -race
make consumer-test   # jalur import eksternal tidak boleh rusak
make parity-check    # golden ngodingmcp tidak berubah (butuh repo saudara)
```

Bila `make check` gagal, perbaiki akar masalah dulu. Jangan menambah fitur di
atas kegagalan.

## Prove-It Pattern

Untuk bugfix dan perubahan kontrak:

1. Tulis test yang **gagal lebih dulu** (reproduksi).
2. Perbaiki kode produksi sampai test hijau.
3. Jangan menonaktifkan test atau menaikkan budget sebagai jalan pintas.

Contoh nyata ada di `internal/mcpserver/activity_test.go` dan
`cmd/mayamcp/main_test.go`.

## Kontrak tool

`internal/mcpserver/testdata/tools.golden.json` mengunci `tools/list`
(file pretty 884 byte, payload compact 601 byte dari budget 2048).

- Perubahan nama/deskripsi/schema tool harus disengaja.
- Regenerasi:
  ```sh
  go test ./internal/mcpserver -run TestToolContract -args -update
  ```
- Review diff golden di commit yang sama. Budget tidak naik tanpa keputusan
  sadar.

## Batas arsitektur

| Masuk ke inti | Tetap di konsumen |
|---|---|
| Host interface, identitas + default, logger, activity middleware | Auth/OAuth, Web UI/TUI, systemd, config store |
| Helper `api`, `ping`/`describe_server`, golden + budget | Tool domain, filesystem, worker, resource kustom |

`internal/` tidak bisa diimpor dari modul lain — semua yang publik lewat
`api/`. Rincian ada di `docs/api-guide.md`.

## Perubahan perilaku pengguna

Wajib disertai update `README.md`/`docs/`. Kalau menambah exported function di
`api`, tambahkan godoc yang menjelaskan default dan kapan dipakai.

## Commit & rilis

- Pesan commit menjelaskan *why*, bukan hanya *what*.
- CalVer `vYYYY.MM.MINOR`; rilis dibuat lewat tag `v20*` atau
  `workflow_dispatch` pada `.github/workflows/release.yml`.
- `CHANGELOG.md` dihasilkan dari notebook TaskDev:
  ```sh
  taskdev_cli.py changelog --calver=v2026.11
  ```
- Notebook TaskDev (`TaskDev/`) sengaja privat/ter-`.gitignore`. Jangan
  mengharapkannya ada di kloning publik; ringkasan keputusan ada di `docs/`.

## Checklist PR

- [ ] `make check`, `make consumer-test`, `make parity-check` hijau.
- [ ] Prove-It dipakai untuk bugfix/perubahan kontrak.
- [ ] Golden/budget tidak berubah, atau diperbarui di commit yang sama.
- [ ] `docs/` dan `README.md` diperbarui bila perilaku berubah.
- [ ] Tidak ada secret, path host, atau identitas konsumen di inti/log.
- [ ] Tidak ada dependency baru tanpa justifikasi.
