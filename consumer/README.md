# Consumer Acceptance Module

Direktori ini adalah **modul Go terpisah** (`example.com/mayamcp-consumer`) yang
hanya mengimpor `github.com/neflalabs/mayamcp/api`. Tujuannya membuktikan bahwa
mayamcp benar-benar bisa dikonsumsi dari luar modul, bukan hanya dari test internal.

## Kenapa modul terpisah

Aturan `internal` Go melarang modul lain mengimpor `internal/app` dan
`internal/mcpserver`. Test di dalam repositori mayamcp tidak bisa membuktikan
jalur import eksternal. Modul ini memakai `replace ... => ../` sehingga
kompilasi tetap lokal, tetapi hanya `api` yang terlihat.

## Menjalankan

```sh
make consumer-test      # dari root mayamcp
# atau:
cd consumer && GOWORK=off go test -count=1 ./...
```

`GOWORK=off` dipakai supaya `go.work` di direktori induk tidak mengambil alih
resolusi modul. CI menjalankan target yang sama.

## Yang diverifikasi

`acceptance_test.go` membangun server dengan `api.NewWithOptions` dan hook:

- hanya tool konsumen yang terdaftar (tanpa `ping`/`describe_server`);
- resource kustom via `RegisterResources`;
- middleware kustom via `Middleware`;
- activity middleware publik via `api.ActivityMiddleware`;
- argumen mengandung `Bearer secret-token` **tidak** bocor ke log.

Kalau test ini gagal, jalur impor publik sedang rusak — perbaiki sebelum fitur
baru ditambahkan (Stop-the-Line).

## Saat `make rename` dijalankan

`make rename MODULE=...` memperbarui `consumer/go.mod` otomatis:

- `require` lama di-drop, `require $MODULE v0.0.0` ditambahkan;
- `replace $MODULE => ../` diset ulang;
- import di `acceptance_test.go` ikut terganti.

Jadi jangan mengedit `consumer/go.mod` manual setelah rename kecuali ada alasan
khusus.
