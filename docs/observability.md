# Observability: Activity Log

Satu-satunya output observability bawaan MayaMCP adalah **satu baris log per
`tools/call`**. Formatnya adalah kontrak: TUI `ngodingmcp` mengenali baris
memakai prefix `tool: `.

## Format

```text
tool: <tool> [<target>] ✓|✗ [detail] <durasi> [~N tok]
```

Contoh:

```text
tool: ping ✓ 850µs
tool: describe_server ✓ 1.2ms ~10 tok
tool: failing_tool ✗ error 3.1ms
tool: edit_file main.go ✓ 12ms ~8.3k tok
```

| Bagian | Aturan |
|---|---|
| Prefix | `api.ActivityPrefix` = `"tool: "`, selalu diawali di `msg` |
| Tool | Nama tool, sudah disanitasi |
| Target | Opsional, dari allowlist kunci generik atau `TargetExtractor` |
| Hasil | `✓` sukses, `✗` gagal |
| Detail | Hanya `error`, `timed out`, atau `exit N` — tidak pernah isi error |
| Durasi | `µs`, `ms`, `s`, atau `m…s` |
| Suffix | ` ~N tok` hanya bila estimasi token > 0 |

Sukses dicatat pada level `INFO`, gagal pada `WARN`.

## Atribut slog

Selain `msg`, setiap baris membawa atribut eksak:

| Atribut | Arti |
|---|---|
| `content_bytes` | Ukuran `Content` setelah JSON encoding |
| `content_tokens` | `content_bytes / 4` |
| `structured_bytes` | Ukuran `StructuredContent` |
| `structured_tokens` | `structured_bytes / 4` |

Estimasi token adalah `bytes/4`, bukan penghitungan tokenizer. Suffix compact
memakai payload **terbesar**, karena klien meneruskan salah satu (atau keduanya)
ke model — menjumlahkan keduanya akan double count. Konsumen yang ingin rincian
memakai `api.MeasureActivity(res).Suffix(true)` yang menghasilkan
` ~N tok (ct X + st Y)` (format ini dipakai `ngodingmcp`).

## Jaminan privasi

Yang **tidak pernah** masuk log:

- nilai `command`, `args`, isi file, teks edit, environment, token, signature;
- userinfo, port, query, dan fragment URL — hanya hostname yang dipertahankan;
- pesan error mentah (hanya `error`/`timed out`/`exit N`);
- kredensial gaya `user:pass@host`.

Target diambil dari allowlist eksplisit: `name`, `path`, `id`, `target`, dan
`from`/`to`. Nilai `from`/`to` yang bukan URL dibersihkan sebagai path lokal
(`old.go → new.go`), bukan diubah menjadi `? → ?`.

`TargetExtractor` konsumen dipanggil lebih dulu, lalu hasilnya selalu lewat
`api.SanitizeTarget` (karakter kontrol → spasi, whitespace dirapikan, maksimal
60 rune sebelum ellipsis). Sanitizer **bukan** sensor: extractor tetap tidak
boleh mengembalikan secret.

## Memakai di server milik sendiri

```go
srv := mcp.NewServer(&impl, nil)
api.AddTextTool(srv, tool, handler)
srv.AddReceivingMiddleware(api.ActivityMiddleware(logger, extractor))
```

Bila memakai `NewCore` dan sudah memiliki middleware domain-aware, set
`Config.DisableActivityMiddleware = true` lalu pasang sendiri — ini yang
dilakukan `ngodingmcp` agar exit code `_meta` tetap muncul sebagai `✗ exit N`.

## Mengonsumsi baris

```go
if strings.HasPrefix(record.Message, api.ActivityPrefix) {
	line := strings.TrimPrefix(record.Message, api.ActivityPrefix)
	// tokenisasi ringan: tool [target] ✓/✗ [detail] durasi [suffix]
}
```

Jangan parsing dengan memotong spasi ke-N: `target` dan `detail` bisa kosong.
Pakai token `✓`/`✗` sebagai penanda status.

## Test regresi

- `internal/mcpserver/activity_test.go` — secret leakage, allowlist, suffix,
  outcome, dan end-to-end middleware.
- `api/toolkit_test.go` — wrapper publik `ActivityMiddleware`, `MeasureActivity`,
  `SanitizeTarget`, dan hook `New`.
- `ngodingmcp` mempertahankan `_meta` outcome test; golden tool-nya dijaga
  `make parity-check`.

Perubahan pada prefix, format baris, atau atribut dianggap perubahan kontrak —
perbarui golden/test di commit yang sama.
