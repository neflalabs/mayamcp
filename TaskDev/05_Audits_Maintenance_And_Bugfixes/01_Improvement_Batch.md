# 01 — Improvement Batch

**Metadata Tracking:**
* **Tanggal Dibuat:** `2026-11-25 16:00:56 +0900`
* **Pembaruan Terakhir:** `2026-11-25 16:00:56 +0900`
* **Status:** `Completed`
* **Target Versi / CalVer:** `v2026.11.1`
* **Kategori:** `Audits Maintenance And Bugfixes`
* **Komponen Terkait:** `api/`, `cmd/mayamcp/`, `internal/app/`, `internal/mcpserver/`, `AGENTS.md`, `go.mod`, `.github/workflows/ci.yml`, `TaskDev/01_Foundation_And_Architecture/01_MayaMCP_Architecture_Guideline.md`
* **Tags:** `[#hardening, #coverage, #prove-it, #ci, #api-facade, #toolchain]`

---

## 🎯 1. Konteks & Tujuan

Tindak lanjut dari `01_MayaMCP_Architecture_Guideline.md` (S1–S6) dan daftar
improvement yang disampaikan setelah brief codebase. Sasaran: menutup celah test
jalur runtime, merapikan desain yang masih setengah jadi, menambah paritas CI,
dan menjaga konsistensi toolchain tanpa mengubah kontrak tool.

Prinsip yang dipakai: **Stop-the-Line** (test gagal → perbaiki akar sebelum
lanjut) dan **Prove-It Pattern** (reproduksi merah → perbaikan → hijau).

---

## 🔬 2. Prove-It: Dua Kegagalan Saat Implementasi

### 2.1 Custom tool untuk jalur error缺少 input schema

Saat menambah tool uji `failing_tool` lewat `Server.AddTool` (tanpa generic
helper), SDK panic:

```text
panic: AddTool "failing_tool": missing input schema
```

* **❌ Sebelum**
  ```go
  s.AddTool(&mcp.Tool{Name: "failing_tool"}, func(...) (*mcp.CallToolResult, error) { ... })
  ```
* **✅ Sesudah** — pakai generic `mcp.AddTool` yang meng-infer schema dari tipe `In`
  ```go
  mcp.AddTool(s, &mcp.Tool{Name: "failing_tool"},
      func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) { ... })
  ```

### 2.2 `RegisterOnShutdown` dipanggil asynchronous

Test mengira callback langsung terpasang begitu `runHTTP` return; ternyata
`http.Server.Shutdown` menjalankan `onShutdown` di goroutine terpisah.

* **❌ Sebelum**
  ```go
  select { case <-shutdownCalled: default: t.Fatal("srv.Shutdown was not called") }
  ```
* **✅ Sesudah**
  ```go
  select { case <-shutdownCalled: case <-time.After(2*time.Second): t.Fatal(...) }
  ```

Kedua kegagalan diperbaiki sebelum fitur berikutnya dikerjakan; `go test ./...`
kembali hijau.

---

## 🛠️ 3. Perubahan per Item

### S1 — Test jalur runtime
`internal/mcpserver/activity_test.go` kini punya `TestActivityMiddlewareLogsToolCallsEndToEnd`
(in-memory session + `lockedBuffer`): handshake tidak memunculkan baris activity;
`ping` → `level=INFO`, `tool: ping ✓`, atribut byte/token; `failing_tool` →
`level=WARN`, `tool: failing_tool ✗ error`; dan tool dengan argumen `command`
berisi Bearer token tidak membocorkan secret ke log.

`cmd/mayamcp/main.go` memisahkan `serveHTTP` (signal wiring) dari `runHTTP`
(cycle life server) supaya shutdown bisa dites tanpa mengirim sinyal ke proses
test. `TestRunHTTPShutdownOnContextDone`, `TestRunHTTPReturnsListenError`, dan
`TestLimitBody` ditambahkan.

### S2 — `defaultInstructions` generik
* **❌ Sebelum:** menyebut `ping` dan `describe_server` secara eksplisit.
* **✅ Sesudah:** `Use the available tools to answer requests...` + anjuran
  mengisi `Instructions` eksplisit saat mengganti tool. Tidak ada tool yang
  di-hardcode di inti.

### S3 — `MetaNamespace` dihapus
Plumbing empat lapis (`Config` → `Options` → `New` → middleware dengan
parameter `_ string`) tidak punya konsumen sejak `_meta` `run_command` tidak ikut
ke template. Dihapus dari `app.Config`, `mcpserver.Options`, konstanta default,
tanda tangan `activityMiddleware`/`activityOutcomeWith`, test, dan konvensi
`AGENTS.md`. `ngodingmcp` tetap memakai namespace sendiri saat di-backport.

### S4 — Fasad publik lengkap
`api/api.go` mengekspor `ActivityPrefix`, `ActivityLine`, dan `CountTools`
(delegasi, bukan reimplementasi). `api/api_test.go` menguji ketiganya dari
package `api_test`.

### S5 — go.mod / toolchain
* **❌ Sebelum:** `go 1.26.10` (patch pin), tanpa direktif toolchain.
* **✅ Sesudah:** `go 1.26.0` + `toolchain go1.27.4`; `go mod tidy` tidak
  menambah dependency baru.

### S6 — Paritas CI & sinkronisasi
`.github/workflows/ci.yml` baru: job `check` menjalankan `make check` + `make
build`; job `notebook` menjalankan `taskdev-cli validate` secara kondisional bila
skill CLI tersedia di runner (tidak ada path absolut yang hardcode). Checklist
backport sudah berada di guideline §3.8.

### Polish — kontrak hasil call, body limit, target, CountTools
* `TestToolCallResults` mengunci hasil `ping` → `pong` dan `describe_server`
  → `test-server v1` + structuredContent.
* `limitBody` membatasi POST MCP ke 1 MiB via `http.MaxBytesReader` (413).
* `safeTargetValue` tidak lagi membuang semua `@`: `user:pass@host` tetap
  ditolak, `@scope/pkg` dipertahankan.
* `CountTools` memberi bound 5 detik bila `ctx` tidak punya deadline.

---

## ✅ 4. Bukti Test & Coverage

```text
make check  → gofmt, go vet, staticcheck, go test -race: LOLOS
go test ./... → ok di api, cmd/mayamcp, internal/app, internal/mcpserver
```

| Package | Sebelum | Sesudah |
|---|---:|---:|
| `api` | 100,0% | 100,0% |
| `cmd/mayamcp` | 2,9% | 37,8% |
| `internal/app` | 88,2% | 88,2% |
| `internal/mcpserver` | 81,4% | 94,8% |
| **Total** | 70,7% | **85,9%** |

Fungsi kritis: `activityMiddleware` 0% → **100%**; `runHTTP` 0% → **90,9%**;
`limitBody` **100%**. `serveHTTP` tetap 0% karena hanya membungkus
`signal.NotifyContext` + delegasi ke `runHTTP`.

Golden `testdata/tools.golden.json` **tidak berubah** — kontrak `tools/list`
tetap 601 byte dari budget 2048.

---

## ⚠️ 5. Dampak & Risiko

* Penghapusan `MetaNamespace` adalah perubahan API publik, tapi aman karena
  template baru dan tidak ada konsumen eksternal yang tercatat.
* CI notebook step bersifat kondisional: runner tanpa skill taskdev melewati
  validasi TaskDev; ini disengaja agar CI tidak bergantung pada path home.
* `serveHTTP` belum ter-cover langsung; risikonya rendah karena logikanya sudah
  dipindah ke `runHTTP`.

---

## ⏱️ 6. Riwayat Revisi & Audit Trail

| Tanggal & Waktu | Versi / Commit | Penulis | Ringkasan Perubahan |
|---|---|---|---|
| `2026-11-25 16:00:56 +0900` | `v2026.11.1` | AI / Developer | Eksekusi S1–S6 + polish, Prove-It pada dua test gagal, coverage total 70,7% → 85,9%, CI workflow, update status guideline |

---

## 📚 7. Referensi

* `TaskDev/01_Foundation_And_Architecture/01_MayaMCP_Architecture_Guideline.md` — keputusan arsitektur & roadmap.
* `TaskDev/README.md` — master index.
* `AGENTS.md` — konvensi (termasuk pengaman nama `MayaMCP` di index TaskDev).
