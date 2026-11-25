# 02 — Comprehensive Codebase Suggestions

**Metadata Tracking:**
* **Tanggal Dibuat:** `2026-11-25 16:11:00 +0900`
* **Pembaruan Terakhir:** `2026-11-25 16:37:55 +0900`
* **Status:** `Completed`
* **Target Versi / CalVer:** `v2026.11.2`
* **Kategori:** `Audits Maintenance And Bugfixes`
* **Komponen Terkait:** `api/`, `cmd/mayamcp/`, `internal/app/`, `internal/mcpserver/`, `docs/`, `Makefile`, `.github/workflows/ci.yml`, `AGENTS.md`, `TaskDev/`
* **Tags:** `[#audit, #suggestions, #roadmap, #api-design, #observability, #performance, #dx, #hygiene]`

---

## 🎯 1. Ringkasan Eksekutif & Skor Baseline

Audit menyeluruh dilakukan terhadap seluruh 30 berkas codebase (`mayamcp` v2026.11.1, 1.907 baris Go + 866 baris Markdown; 206 baris di `docs/`, `README.md`, dan `AGENTS.md`) pasca selesainya batch S1–S6.

| Dimensi Evaluasi | Skor | Ringkasan Kondisi Saat Ini |
|---|---:|---|
| **Arsitektur & Batas Layer** | **9.0 / 10** | `internal/mcpserver` benar-benar *transport-agnostic*; `api/` fasad tipis tanpa duplikasi logika; SSE `WriteTimeout=0` tepat sasaran. |
| **Kualitas Kode & Pengujian** | **8.5 / 10** | Idiomatik, minim dependency eksternal, coverage 85.9%, uji regresi anti-bocor secret sangat kuat. Masih ada sisa domain `ngodingmcp` di `activity.go`, inkonsistensi `Host.Version()`, dan `identityHost` yang belum meng-embed `Host`. |
| **Usefulness & Developer Experience** | **7.5 / 10** | Sangat solid sebagai fondasi internal, namun masih terlalu minimalis sebagai *starter template* publik karena belum ada pola contoh tool berparameter dan penanganan error domain. |
| **Skor Keseluruhan** | **8.3 / 10** | Fondasi rekayasa yang matang dan jujur pada ruang lingkupnya; siap ditingkatkan ke level *production-grade starter template*. |

---

## 🔍 2. Temuan & Saran Perbaikan Komprehensif

Saran dikelompokkan berdasarkan tingkat prioritas dan dampaknya terhadap stabilitas kontrak (`tools.golden.json` & budget 2048 byte).

---

### 🔴 Prioritas Tinggi (P0 — Kontrak API, Ekstensibilitas Template & Git Hygiene)

#### R1. Selesaikan Inkonsistensi `Host.Version()` & Ubah `identityHost` Menjadi *Struct Embedding*
* **Lokasi:** `internal/mcpserver/domain.go:6-9`, `internal/mcpserver/server.go:45-87`, `api/api.go:41-43`
* **Masalah:**
  1. Kontrak `Host` mendefinisikan `Version() string`, tetapi `mcpserver.New(host Host, version string, opts Options)` **tidak pernah memanggil `host.Version()`**. Jika pemanggil mengirim `version == ""` dengan `host` yang memiliki `Version() == "v2.1.0"`, fungsi diam-diam mengabaikan `host.Version()`, memakai `DefaultVersion` (`"dev"`), lalu menimpa `host` dengan `identityHost{version: "dev"}`. Di `api/api_test.go:76`, pemanggil terpaksa menulis versi dua kali (`api.New(testHost{"memory-server", "v1"}, "v1", ...)`).
  2. `identityHost` saat ini hanya mengimplementasikan `Name()` dan `Version()`, lalu menimpa variabel `host` (`host = identityHost{...}`). Begitu pengguna template menambah method domain baru pada interface `Host`, `mcpserver.New` akan gagal kompilasi karena `identityHost` tidak mengimplementasikan method baru tersebut.
* **Saran Perbaikan:**
  1. Terapkan urutan resolusi versi yang simetris dengan `name`:
     `opts.Version` (opsional baru di `Options`) → parameter `version` → `host.Version()` (bila `host != nil`) → `DefaultVersion`. Field baru ini wajib pula punya default sesuai konvensi `AGENTS.md`.
  2. Gunakan *interface embedding* pada `identityHost` dengan fallback `fallbackHost` ketika `host == nil`:
     ```go
     type fallbackHost struct{}
     func (fallbackHost) Name() string    { return DefaultName }
     func (fallbackHost) Version() string { return DefaultVersion }

     type identityHost struct {
         Host
         name    string
         version string
     }
     func (h identityHost) Name() string    { return h.name }
     func (h identityHost) Version() string { return h.version }
     ```
  3. Tambahkan unit test di `internal/mcpserver/server_identity_test.go` untuk memastikan `host.Version()` dihormati saat parameter `version` kosong, dan tetap kalah prioritas jika `version` eksplisit diberikan.
* **Dampak Kontrak:** Tidak mengubah `tools/list` (`tools.golden.json` aman).

#### R2. Cabut Sisa Kunci Domain `ngodingmcp` dari `activityTarget()` ke Opsi Konfigurasi
* **Lokasi:** `internal/mcpserver/activity.go:17`, `internal/mcpserver/activity.go:182-254`
* **Masalah:** Meski `internal/mcpserver` diklaim bebas domain, `activityTarget()` masih meng-hardcode 9 kunci argumen milik tool filesystem/git/fetch `ngodingmcp` (`source_url`, `from`, `to`, `path`, `name`, `ref`, `remote`, `paths`, `pattern`) serta menyebut TUI di komentar paket inti (`activity.go:17`). Jika konsumen memakai template ini untuk domain lain (database, HTTP API, Docker, IoT), `activityTarget()` selalu mengembalikan string kosong `""` tanpa ekstensi yang sah.
* **Saran Perbaikan:**
  1. Tambahkan field opsional pada `mcpserver.Options` (dan teruskan melalui `app.Config`):
     ```go
     // TargetExtractor extracts an un-sanitized target label from tool call
     // arguments. Output is always passed through cleanTarget before logging.
     TargetExtractor func(tool string, args json.RawMessage) string
     ```
  2. Pertahankan ekstraktor bawaan yang ringkas dan generik (`name`, `path`, `id`, `target`), sementara middleware **selalu** membungkus hasil ekstraktor dengan `cleanTarget()` agar batas panjang (`maxTargetLen = 60`) dan pembersihan karakter kontrol tetap terjamin *by construction*.
  3. Saat *backport* ke `ngodingmcp`, cukup suntikkan ekstraktor domain `ngodingmcp` lewat `Options.TargetExtractor`.
* **Dampak Kontrak:** Tidak mengubah `tools/list`; format baris `tool: <target> ✓/✗` tetap identik.

#### R3. Perbaiki Bug Parsing `from` / `to` pada Non-URL di `activityTarget()`
* **Lokasi:** `internal/mcpserver/activity.go:210-219`
* **Masalah:** Blok `case str("from") != "" || str("to") != ""` memanggil `host("from")` (`urlHost`) secara langsung. Apabila argumen berisi path lokal seperti `{"from": "old.go", "to": "new.go"}`, `urlHost` gagal menemukan hostname dan menghasilkan log tidak informatif: `"? → ?"`.
* **Saran Perbaikan:** Gunakan helper `hostOrSafeValue(raw string)`: bila `strings.Contains(raw, "://")` gunakan `urlHost(raw)`, selain itu gunakan `safeTargetValue(raw)`. Dengan demikian `https://a.example/x → https://b.example/y` tetap menjadi `a.example → b.example`, sedangkan `old.go → new.go` tampil bersih tanpa bocor kredensial.

#### R4. Kunci Status Working Tree Git (`v2026.11.1`)
* **Lokasi:** Root repository (`git status`)
* **Masalah:** Terdapat 10 berkas termodifikasi dan 7 berkas/direktori baru (`api/`, `TaskDev/`, `docs/`, `.github/`, test baru) yang belum di-commit sejak batch `v2026.11.1`.
* **Saran Perbaikan:** Jalankan `make check` dan commit seluruh perubahan batch `v2026.11.1` secara atomik sebelum memulai batch perubahan baru agar riwayat Git menjadi *baseline* yang bersih untuk *backport* ke `ngodingmcp`.

---

### 🟡 Prioritas Menengah (P1 — Performa, Transport HTTP & Pembersihan Dead State)

#### R5. Hilangkan Alokasi Buffer `json.Marshal` pada `measureResult()`
* **Lokasi:** `internal/mcpserver/activity.go:121-140`
* **Masalah:** Pada setiap `tools/call`, `measureResult` memanggil `json.Marshal(r.Content)` dan `json.Marshal(r.StructuredContent)` hanya untuk mengambil `len(raw)`. Untuk tool yang mengembalikan payload besar (misalnya pembacaan berkas ratusan KB atau output log panjang di `ngodingmcp`), ini menggandakan alokasi memori heap per panggilan.
* **Saran Perbaikan:** Gunakan `byteCounter` (`type byteCounter int64` yang mengimplementasikan `io.Writer`) bersama `json.NewEncoder(&counter)` untuk menghitung jumlah byte JSON secara eksak dengan **nol alokasi buffer** (`O(1)` memory). Perhatikan bahwa `json.Encoder.Encode` menambahkan 1 byte trailing newline (`\n`), sehingga kurangi 1 byte bila `counter > 0` agar hasilnya 100% identik dengan `len(json.Marshal(...))`.

#### R6. Hapus Field Mati `Core.options` & Tambah Opsi Level/Format Log di `app.Config`
* **Lokasi:** `internal/app/core.go:15-60`, `internal/app/core_test.go:167-186`
* **Masalah:**
  1. Field `options mcpserver.Options` disimpan di struct `app.Core`, namun tidak pernah dibaca di mana pun dalam kode produksi—keberadaannya semata-mata untuk diinspeksi oleh satu white-box test (`TestNewCoreOptionsPassthrough`).
  2. `NewCore` mengunci `slog.NewTextHandler` dengan `slog.LevelInfo` ketika membuat logger dari `LogWriter` atau `os.Stderr`, padahal deployment kontainer HTTP sering membutuhkan JSON handler atau level log yang bisa dikonfigurasi.
* **Saran Perbaikan:**
  1. Hapus field `options` dari `app.Core`. Verifikasi *passthrough* opsi melalui perilaku eksternal sesi MCP dan logger.
  2. Tambahkan `LogLevel slog.Leveler` dan `LogJSON bool` opsional pada `app.Config` (tetap 100% Go stdlib `log/slog`).

#### R7. Penguatan Transport CLI & HTTP di `cmd/mayamcp/main.go`
* **Lokasi:** `cmd/mayamcp/main.go:23-112`, `cmd/mayamcp/main_test.go`
* **Masalah:**
  1. Mode `stdio` menjalankan `core.Server.Run(context.Background(), ...)` tanpa `signal.NotifyContext`, berbeda dengan mode `http` yang menangani `SIGINT`/`SIGTERM` secara bersih.
  2. Variabel versi di-hardcode `"dev"` pada `flag.String("version", "dev", ...)` sehingga tidak mendukung injeksi versi saat build rilis (`-ldflags "-X main.defaultVersion=v2026.11.2"`).
  3. Belum ada endpoint `/healthz`, peringatan log ketika `-addr` di-bind ke alamat non-loopback (`0.0.0.0`) tanpa auth/TLS bawaan, serta pengujian *round-trip* klien MCP sungguhan di atas `StreamableHTTPHandler` (coverage `cmd/mayamcp` saat ini `37,8%`).
* **Saran Perbaikan:**
  1. Deklarasikan `var defaultVersion = mcpserver.DefaultVersion` di `package main` agar bisa di-override via `-ldflags`, lalu jadikan nilai default flag `-version`.
  2. Bungkus `stdio` dengan `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` dan perlakukan `context.Canceled` sebagai exit bersih (`exit 0`).
  3. Gunakan `http.NewServeMux()` di layer `cmd/` untuk menyediakan `GET /healthz` (`200 OK`) di samping endpoint MCP, catat `slog.Warn` bila `-addr` bukan loopback, dan tambahkan integration test `httptest.NewServer` + `mcp.StreamableClientTransport` di `cmd/mayamcp/main_test.go`.

---

### 🟢 Prioritas Standar (P2 — Developer Experience, Dokumentasi & Otomasi)

#### R8. Sediakan Pola Contoh Tool Berparameter & Domain Error Tanpa Mengubah Kontrak Golden
* **Lokasi:** `api/example_test.go` (baru) dan `docs/creating-tools.md` (baru)
* **Masalah:** Kedua tool bawaan (`ping` dan `describe_server`) memakai argumen kosong (`struct{}`). Developer yang mengkloning MayaMCP tidak melihat contoh resmi cara:
  - Mendefinisikan struct input dengan tag `json` dan `jsonschema` (`description`, enum, required).
  - Membedakan **domain/validation error** (`&mcp.CallToolResult{IsError: true, Content: ...}, nil`) dari **protocol/transport error** (`return nil, err`).
  - Memilih antara `addTextTool` (hemat token `tools/list`, tanpa `outputSchema`) vs `mcp.AddTool` (dengan `outputSchema`).
* **Saran Perbaikan:** Karena menambah tool ke server default akan mengubah `testdata/tools.golden.json`, tulis panduan ringkas di `docs/creating-tools.md` dan tambahkan testable Go example (`ExampleNew_customTool` di `api/example_test.go`) yang diverifikasi otomatis oleh `go test ./...` tanpa menambah beban kontrak bawaan.

#### R9. Perbaiki Tautan & Provenansi Dokumentasi di `docs/mcp-as-template.md`
* **Lokasi:** `docs/mcp-as-template.md:41-44`, `README.md:12-14`
* **Masalah:** `docs/mcp-as-template.md` memiliki tautan relatif ke `TaskDev/01_Foundation_And_Architecture/03_MCP_Server_Tool_Layout_Refactor.md` dan `06_Token_Efficiency_Plan.md` yang hanya ada di repo upstream `ngodingmcp`, sehingga menjadi *broken link* di repo `mayamcp`.
* **Saran Perbaikan:** Ubah tautan relatif tersebut menjadi referensi teks eksplisit ke repo upstream (`ngodingmcp:TaskDev/...`) agar tidak ada tautan mati di dalam repo template.

#### R10. Tingkatkan `Makefile`, CI & Otomasi Guardrail `TaskDev/README.md`
* **Lokasi:** `Makefile`, `.github/workflows/ci.yml`, `AGENTS.md:24-27`
* **Masalah:**
  1. `make build` belum menyuntikkan versi CalVer ke binary, dan belum ada target `make rename MODULE=...` untuk menginstansiasi template menjadi modul baru.
  2. `make check` selalu memanggil `go run` dengan `STATICCHECK_VERSION` sehingga gagal jika dijalankan di lingkungan *offline* tanpa cache modul.
  3. `AGENTS.md` mencatat bahwa `taskdev-cli sync` memiliki fallback hardcoded `Nefla POS` sehingga `TaskDev/README.md` harus dikoreksi manual setiap kali sync.
* **Saran Perbaikan:**
  1. Tambahkan target `cover` di `Makefile`, injeksi `-ldflags "-X main.defaultVersion=$(VERSION)"` pada `build`, dan sediakan target `rename` untuk mengganti nama modul `mayamcp` saat kloning template.
  2. Pada target `check`, cek terlebih dahulu apakah binary `staticcheck` sudah tersedia di `PATH` (`command -v staticcheck`) sebelum fallback ke `go run` dengan `STATICCHECK_VERSION`, serta tambahkan `go mod verify` dan cross-compile check di `.github/workflows/ci.yml`.
  3. Tambahkan target `taskdev-sync` di `Makefile` yang menjalankan `taskdev-cli sync` lalu otomatis memastikan `proyek **MayaMCP**` tetap terjaga di `TaskDev/README.md`.

---

## 🗺️ 3. Roadmap Vertical Slices Lanjutan (`S7`–`S12`)

Melanjutkan penomoran dari `S1`–`S6` pada `01_MayaMCP_Architecture_Guideline.md`:

| Slice | Judul & Sasaran | Rujukan Saran | Target CalVer | Status |
|---|---|---|---|---|
| **S7** | **Git Baseline & Hygiene:** Commit batch `v2026.11.1` secara atomik dan perbaiki tautan upstream di `docs/mcp-as-template.md` | R4, R9 | `v2026.11.1` | `Completed` (`v2026.11.2`) |
| **S8** | **Identity Contract Symmetry & Host Embedding:** Hormati `Host.Version()` di `mcpserver.New`, ubah `identityHost` menjadi *interface embedding*, dan tambah unit test identitas | R1 | `v2026.11.2` | `Completed` (`v2026.11.2`) |
| **S9** | **Domain-Free Activity Extractor:** Ekstrak `Options.TargetExtractor`, perbaiki fallback `from`/`to` non-URL, dan ganti `json.Marshal` dengan `byteCounter` | R2, R3, R5 | `v2026.11.2` | `Completed` (`v2026.11.2`) |
| **S10** | **Core & CLI/HTTP Polish:** Hapus `Core.options`, tambah `LogLevel`/`LogJSON`, signal handling pada `stdio`, `/healthz`, peringatan non-loopback, dan HTTP round-trip test | R6, R7 | `v2026.11.2` | `Completed` (`v2026.11.2`) |
| **S11** | **DX & Tool Authoring Guide:** Tambah `api/example_test.go` (`ExampleNew_customTool`) dan `docs/creating-tools.md` tanpa menyentuh `tools.golden.json` | R8 | `v2026.11.2` | `Completed` (`v2026.11.2`) |
| **S12** | **Makefile, CI & Tooling Automation:** Tambah target `cover`, `rename`, `ldflags` build, offline-friendly `staticcheck`, cross-compile CI, dan guard `taskdev-sync` | R10 | `v2026.11.2` | `Completed` (`v2026.11.2`) |

---

## ✅ 4. Kriteria Verifikasi & Checklist Backport

Sebelum setiap slice dinyatakan `Completed`:
1. **Prove-It Pattern:** Untuk R1 (bug `Host.Version()` & embedding `Host`) dan R3 (bug `from`/`to` path menghasilkan `? → ?`), tulis unit test yang **gagal terlebih dahulu** sebelum mengubah kode produksi.
2. **Quality Gate:** Jalankan `make check` (`gofmt`, `go vet`, `staticcheck`, `go test -race`) dengan nol warning dan coverage total tetap **≥ 85.9%**.
3. **Contract Guard:** Pastikan `testdata/tools.golden.json` tetap 601 byte (tidak berubah).
4. **Backport ke `ngodingmcp`:** Saat mem-port S8–S9 ke `ngodingmcp`, hubungkan ekstraktor target spesifik `ngodingmcp` (`source_url`, `from`/`to`, `path`, `ref`, `remote`, `paths`, `pattern`) melalui `mcpserver.Options.TargetExtractor`.

---

### Catatan eksekusi

Seluruh S7–S12 selesai dan terverifikasi. **Deviasi dari R1**: saran
`identityHost` dengan *interface embedding* tidak dipakai — embed interface
`Host` tidak mewarisi method konkret tambahan. Solusi final: `describe_server`
menerima nama/versi hasil resolusi secara langsung, sehingga `Host` tidak pernah
dibungkus dan tidak ada risiko method domain hilang atau gagal kompilasi.
`opts.Version` tetap ditambahkan dengan default `DefaultVersion`.

Hasil akhir: `make check` hijau; golden 601 byte tidak berubah; coverage
`api` 100%, `app` 96,0%, `mcpserver` 95,6%, `cmd` 48,1%, total **86,0%**
(melebihi guard 85,9%). Prove-It dipakai untuk R1 dan R3 (dua test merah lebih
dulu). Commit baseline `v2026.11.2` sudah dibuat sebagai `6fb6637`.

---

## ⏱️ 5. Riwayat Revisi & Audit Trail

| Tanggal & Waktu | Versi / Commit | Penulis | Ringkasan Perubahan |
|---|---|---|---|
| `2026-11-25 16:11:00 +0900` | `v2026.11.2` (`Proposed / Planning`) | AI / Developer | Penyusunan audit menyeluruh, 10 rekomendasi terstruktur (P0–P2), dan roadmap lanjutan S7–S12 |
| `2026-11-25 16:18:01 +0900` | `v2026.11.2` | AI / Developer | Perbaikan format hasil verifikasi: status lifecycle standar, H1 sinkron dengan indeks, koreksi jumlah berkas/baris Markdown, catatan default `opts.Version`, rujukan baris `activityTarget` |
| `2026-11-25 16:37:55 +0900` | `v2026.11.2` | AI / Developer | Eksekusi S7–S12: identitas simetris tanpa `identityHost`, `TargetExtractor` + `byteCounter`, log JSON/level, `/healthz` + signal stdio, example tool, Makefile/CI; coverage 86,0% |

---

## 📚 6. Referensi

* `TaskDev/01_Foundation_And_Architecture/01_MayaMCP_Architecture_Guideline.md` — keputusan arsitektur inti dan roadmap S1–S6.
* `TaskDev/05_Audits_Maintenance_And_Bugfixes/01_Improvement_Batch.md` — catatan audit dan coverage batch `v2026.11.1`.
* `AGENTS.md` — konvensi pengembangan, golden test, dan aturan indeks `TaskDev/README.md`.
