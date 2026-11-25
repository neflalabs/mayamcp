# Membuat Tool di MayaMCP

Panduan singkat untuk menambah tool ke server yang dibangun dari template ini.
Contoh yang bisa dijalankan otomatis ada di `api/example_test.go`
(`ExampleNew_customTool`).

## 1. Struktur input

Definisikan struct dengan tag `json`; tag `jsonschema` berisi deskripsi
property di `inputSchema`:

```go
type echoArgs struct {
	Label string `json:"label" jsonschema:"Text to echo"`
}
```

`mcp.AddTool` meng-infer schema dari tipe `In` (termasuk daftar `required`).
Hindari `any` untuk input: skemanya menjadi object kosong dan klien tidak tahu
argumen yang tersedia.

## 2. Error domain vs error protokol

* **Error domain/validasi** → kembalikan `*mcp.CallToolResult` dengan
  `IsError: true`. Model melihat pesan error dan bisa memperbaiki panggilannya.
* **Error protokol/transport** → kembalikan `nil, err`. Ini untuk kegagalan
  sistem yang memang tidak bisa dikoreksi model.

```go
return &mcp.CallToolResult{
	IsError: true,
	Content: []mcp.Content{&mcp.TextContent{Text: "label is required"}},
}, nil, nil
```

Jangan memasukkan path host, token, atau data rahasia ke pesan error.

## 3. Memilih helper registrasi

| Kebutuhan | Cara |
|---|---|
| Hasil berupa teks besar (file, listing, log) | `addTextTool` — hanya `content`, tanpa `outputSchema` (di dalam `internal/mcpserver`) |
| Konfirmasi singkat dengan JSON untuk klien | `mcp.AddTool` dengan `Out any`; konsumen bisa memakai `Options.TargetExtractor` untuk label aktivitas |
| Record kecil yang benar-benar dibutuhkan klien | `mcp.AddTool` dengan tipe `Out` → ada `outputSchema` + `structuredContent` |

Dari luar modul, pakai `mayamcp/api`: `api.New(...)` mengembalikan
`*mcp.Server`, lalu daftarkan tool dengan `mcp.AddTool`.

## 4. Menjaga kontrak bawaan

Golden `internal/mcpserver/testdata/tools.golden.json` mengunci `tools/list`
(601 byte dari budget 2048). Karena itu:

* contoh dan percobaan **jangan** didaftarkan ke server bawaan;
* tulis sebagai `Example...` di `*_test.go` atau dokumentasi saja;
* jika tool baru memang masuk ke template, perbarui golden di commit yang sama:

```sh
go test ./internal/mcpserver -run TestToolContract -args -update
```

## 5. Activity log

Target log aktivitas diambil dari kunci generik `name`, `path`, `id`, `target`,
serta `from`/`to`. Untuk kunci spesifik domain (misal git `ref`/`remote`),
berikan `TargetExtractor` lewat `api.Options` / `app.Config`. Keluaran
ekstraktor selalu dipangkas dan dibersihkan oleh middleware, tetapi
implementasinya tetap tidak boleh mengembalikan secret.
