# Panduan Paket `api`

Paket `github.com/neflalabs/mayamcp/api` adalah **satu-satunya kontrak publik**.
Isinya alias tipe dan delegasi; semua logika ada di `internal/app` dan
`internal/mcpserver` (yang tidak bisa diimpor lintas modul karena aturan `internal`
Go).

## Memilih entry point

| Kebutuhan | Pakai | Starter tool |
|---|---|---|
| Sudah punya `mcp.Server`, tinggal pakai helper | `ResolveIdentity`, `NewLogger`, `ActivityMiddleware`, `AddTextTool`, `AddUntypedTool` | Tidak ada (server Anda sendiri) |
| Ingin server siap pakai dengan tool sendiri | `NewWithOptions(host, opts)` | Ada bila `Register == nil`, hilang bila non-nil |
| Ingin lifecycle lengkap: hook, resource, middleware | `NewCore(cfg)` | Sama seperti `NewWithOptions` |
| Kompatibilitas lama, versi diberikan eksplisit | `New(host, version, opts)` | Sama seperti `NewWithOptions` |

`NewWithOptions` lebih disarankan daripada `New`: versi diambil dari
`Options.Version`, lalu `host.Version()`, lalu default — tidak perlu ditulis dua
kali.

## Peta export

**Identitas & logger**
- `type Identity { Name, Title, Version string }` + `ResolveIdentity(Identity) mcp.Implementation`
- `type LogOptions { Writer, Logger, Level, JSON }` + `NewLogger(LogOptions) *slog.Logger`
- `type Host` — kontrak domain minimal: `Name() string`, `Version() string`

**Registrasi tool**
- `type Handler[In, Out any]` — bentuk handler generik
- `AddTextTool[In, Out]` — `content` saja, tanpa `outputSchema`/`structuredContent`
- `AddUntypedTool[In, Out]` — ada `structuredContent`, tanpa `outputSchema`
- `mcp.AddTool` (dari SDK) — `content` + `outputSchema` untuk record kecil

**Runtime**
- `NewCore(Config) *Core` — `Core{ Server, Host, Logger }`
- `NewWithOptions`, `New`
- `NewWithOptions` tidak memulai listener; Anda yang memilih transport.

**Observability**
- `ActivityMiddleware(logger, extractor) mcp.Middleware`
- `MeasureActivity(res) (ActivitySize, bool)` dan `ActivitySize.Suffix(verbose)`
- `SanitizeTarget(s) string`
- `const ActivityPrefix`, `ActivityLine(tool, target, ok, detail, d)`
- `CountTools(ctx, s)`

Semua helper punya default: `mayamcp`, `dev`, `MayaMCP`, instruksi generik,
prefix `tool: `. Lihat [`observability.md`](observability.md) untuk format log.

## Resolusi identitas

Komponen | Prioritas (tertinggi → terendah)
---|---
Nama | `Options.Name` → `Host.Name()` → `DefaultName` (`mayamcp`)
Versi | `Options.Version` → argumen `version` (`New`) → `Host.Version()` → `DefaultVersion` (`dev`)
Title | `Options.Title` → `defaultTitle` (`MayaMCP`)
Instructions | `Options.Instructions` → template generik

Nilai yang sudah di-resolve dipakai **sama** untuk `initialize.serverInfo` dan
tool `describe_server`. `Host` tidak pernah dibungkus, jadi method domain
tambahan tetap tersedia.

## Hook C-lite

Field | Perilaku
---|---
`Register == nil` | `ping` + `describe_server` didaftarkan (contoh `cmd/mayamcp`)
`Register != nil` | Hanya tool dari konsumen; starter tool **tidak** didaftarkan
`RegisterResources != nil` | Dipanggil setelah server dibuat untuk `AddResource`
`Middleware` | Ditambahkan **sebelum** activity middleware (urutan: custom → activity)
`TargetExtractor == nil` | Ekstraktor generik: `name`, `path`, `id`, `target`, `from`/`to`
`DisableActivityMiddleware` | Melewati activity middleware bawaan bila Anda memasang sendiri

Contoh hook lengkap ada di [`tool-authoring.md`](tool-authoring.md).

## `TargetExtractor` yang aman

```go
extractor := func(tool string, args json.RawMessage) string {
	var in struct {
		Repo string `json:"repo"`
	}
	if json.Unmarshal(args, &in) != nil {
		return ""
	}
	// Jangan pernah mengembalikan token, path host, atau isi field.
	return in.Repo
}

opts := api.Options{TargetExtractor: extractor}
```

Keluaran ekstraktor **selalu** dilewatkan `SanitizeTarget` (karakter kontrol
dibersihkan, panjang dibatasi 60 rune). Itu tidak membuat extractor boleh
mengembalikan secret — sanitizer hanya memformat, bukan menyensor.

## Mengukur ukuran hasil

```go
if size, ok := api.MeasureActivity(res); ok {
	// Bandwidth/estimasi konteks, tanpa menggandakan payload besar.
	_ = size.Suffix(false) // " ~10 tok"
	_ = size.Suffix(true)  // " ~10 tok (ct 6 tok + st 4 tok)"
}
```

Estimasi token = `bytes/4`, bukan hitungan eksak. Suffix compact memakai
payload terbesar (`content` atau `structuredContent`) agar tidak double count.

## Kompatibilitas

CalVer `vYYYY.MM.MINOR`. Penambahan fungsi/field di `api` boleh tanpa kenaikan
MINOR; penghapusan atau perubahan tanda tangan hanya pada MINOR dengan catatan
migrasi di `CHANGELOG.md`. Kontrak `tools/list` dan `ActivityPrefix` tidak
berubah karena perubahan `api` biasa.

Rincian lengkap ada di godoc paket (`go doc github.com/neflalabs/mayamcp/api`).
