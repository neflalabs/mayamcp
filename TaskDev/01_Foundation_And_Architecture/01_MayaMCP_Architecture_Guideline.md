# 01 — MayaMCP Architecture Guideline

**Metadata Tracking:**
* **Tanggal Dibuat:** `2026-11-25 15:31:20 +0900`
* **Pembaruan Terakhir:** `2026-11-25 15:31:20 +0900`
* **Status:** `Completed`
* **Target Versi / CalVer:** `v2026.11.0`
* **Kategori:** `Foundation And Architecture`
* **Komponen Terkait:** `api/`, `cmd/mayamcp/`, `internal/app/`, `internal/mcpserver/`, `docs/mcp-as-template.md`, `README.md`, `AGENTS.md`, `Makefile`
* **Tags:** `[#architecture, #adr, #template, #mcp, #transport-agnostic, #source-of-truth]`

---

## 🎯 1. Konteks & Pernyataan Masalah

### 1.1 Konteks Bisnis & Teknis

MayaMCP adalah repo template yang diekstrak dari `ngodingmcp` dengan prinsip
*keep as is dulu, ekstrak belakangan*. Catatan asli ekstraksi disalin ke
`docs/mcp-as-template.md` pada sesi sebelumnya. Setelah ekstraksi, kode berada
dalam kondisi berikut:

* `internal/mcpserver` hanya bergantung pada Go SDK MCP (`mcp`) dan stdlib;
  tidak tahu soal HTTP, stdio, auth, UI, filesystem, atau workspace.
* `internal/app.NewCore(Config)` merakit server + domain host tanpa listener.
* `cmd/mayamcp` memilih transport: `stdio` (default) atau Streamable HTTP
  dengan `Stateless: true`, timeout terikat, dan graceful shutdown.
* `api/` adalah fasad publik (alias + delegasi) karena aturan `internal` Go
  melarang impor langsung dari modul lain.
* Kontrak `tools/list` dikunci golden (`testdata/tools.golden.json`, 601 byte)
  dan budget 2048 byte.
* Perbaikan yang sudah masuk: identitas tunggal (`serverInfo` ==
  `describe_server`), sanitasi activity log (tidak ada `command`/secret/URL
  userinfo di log), penghapusan port debris `run_command`, dan config passthrough
  untuk `Title`/`Instructions`.

Masalah yang harus dicegah: tanpa guideline yang eksplisit, template ini bisa
berangsur menjadi *appliance* (auth/UI/config store), atau menyimpang dari
`ngodingmcp` tanpa mekanisme sinkronisasi.

### 1.2 Tujuan Awal Repo (yang dijaga dokumen ini)

1. Menyediakan **fondasi MCP server Go yang transport-agnostic** dan bebas domain.
2. Bisa dikonsumsi dua cara: **disalin lalu diedit**, atau **diimpor** lewat
   `mayamcp/api`.
3. Menjaga **kontrak tool dan anggaran token** tetap stabil dan disengaja.
4. Menjaga identitas konsumen, path host, dan secret **tidak pernah** masuk ke
   inti atau ke log.
5. Menjadi *source-of-truth* untuk inti MCP yang nantinya di-*backport* ke
   `ngodingmcp`.

### 1.3 Drivers Keputusan

* **D1 — Core minimal.** Perubahan inti harus bisa dipakai ulang oleh domain
  apa pun; setiap fitur baru di core menaikkan biaya bagi semua konsumen.
* **D2 — Kontrak stabil.** Nama/skema tool dan `tools/list` adalah API publik;
  perubahan harus disengaja, terukur, dan punya bukti test.
* **D3 — Privasi by construction.** Activity log adalah satu-satunya output
  observability bawaan, sehingga jaminan "tanpa konten/secret" harus diuji,
  bukan hanya dikomentari.
* **D4 — Satu sumber kebenaran.** Identitas dan konfigurasi tidak boleh punya
  dua jalur yang bisa berbeda.
* **D5 — Anti-divergensi.** Dua repo (template + aplikasi) harus punya arah
  pemimpin yang jelas.
* **D6 — Dependency rendah.** Go stdlib + SDK MCP; dependency baru harus
  bisa dibenarkan.

---

## 🏗️ 2. Evaluasi Opsi & Matriks Trade-off

| Kriteria Evaluasi | Opsi A: Template minimal + stdio/HTTP stateless | Opsi B: HTTP-first | Opsi C: Full appliance | Opsi D: Server domain spesifik |
|---|---|---|---|---|
| Kompleksitas implementasi | Rendah | Rendah–Sedang | Tinggi | Sedang |
| Biaya bagi konsumen | Rendah | Sedang | Tinggi | Rendah, tapi tidak fleksibel |
| Kemudahan backport ke ngodingmcp | Tinggi | Sedang | Rendah | Rendah |
| Risiko drift identitas/kontrak | Rendah | Sedang | Tinggi | Sedang |
| Kesesuaian dengan tujuan awal | **Tinggi** | Sedang | Rendah | Rendah |
| Skalabilitas domain | Tinggi | Tinggi | Sedang | Rendah |

### 2.1 Analisis Mendalam Tiap Opsi

* **Opsi A (dipilih):** core berisi tool, identitas, dan activity log; transport
  dipilih di `cmd`. Konsumen membawa domain, auth, dan UI. Biaya perubahan
  rendah dan cocok untuk backport.
* **Opsi B:** fokus produksi HTTP, auth/TLS via reverse proxy. stdio tetap ada.
  Risikonya: godaan memasang middleware HTTP-spesifik di core.
* **Opsi C:** mengimpor OAuth, Web UI, TUI, config store, dan systemd dari
  `ngodingmcp`. Ini mengembalikan beban identitas lama yang justru dibuang saat
  ekstraksi.
* **Opsi D:** template langsung dikunci ke satu domain. Bertentangan dengan
  mulanya MayaMCP sebagai *genesis/seed* yang belum terkunci domain.

---

## ⚖️ 3. Keputusan Terpilih & Justifikasi

### 3.1 Solusi Terpilih

**Opsi A — Template minimal dengan dua profil transport (stdio default, HTTP
stateless), fasad publik `api/`, dan `ngodingmcp` sebagai konsumen yang
meng-*port* perbaikan inti dari sini.**

### 3.2 Topologi Layer

```text
                konsumen eksternal (go module lain)
                          │ import
                          ▼
              ┌───────────────────────────┐
              │ api/   (fasad publik)     │  alias + delegasi, tanpa logika
              └─────────────┬─────────────┘
                            │
              ┌─────────────▼─────────────┐
              │ app/   NewCore(Config)    │  identitas + logger, tanpa listener
              └─────────────┬─────────────┘
                            │
              ┌─────────────▼─────────────┐
              │ mcpserver/                │  HOST + OPTIONS + TOOLS + ACTIVITY
              │  Host, Options, identity  │  ← inti transport-agnostic
              │  addPing, describe_server │
              │  activityMiddleware       │
              └─────────────┬─────────────┘
                            │ *mcp.Server
              ┌─────────────▼─────────────┐
              │ go-sdk /mcp               │
              └─────────────┬─────────────┘
                            │
             ┌──────────────┴───────────────┐
             ▼                              ▼
   cmd: stdio (default)          cmd: Streamable HTTP
   stdout = protokol             Stateless: true, timeout, shutdown
```

### 3.3 Batas Tanggung Jawab (Aturan Keras)

| Masuk ke inti (`internal/mcpserver`, `internal/app`) | Tetap di luar / di konsumen |
|---|---|
| `Host` interface (kontrak domain minimal) | Implementasi `workspace`/filesystem nyata |
| Resolusi identitas (`Name`, `Title`, `Instructions`, versi) | Auth, OAuth, cookie, session store |
| Tool contoh (`ping`, `describe_server`) | Web UI, TUI, systemd/service |
| `addTextTool`, preset anotasi, `textResult` | Config store, workspace switch |
| `activityMiddleware`, `ActivityLine`, `ActivityPrefix`, `CountTools` | Resource `AGENTS.md`, prompts, module opt-in |
| Golden contract + token budget test | Health check, metrik, tracing |
| Fasad `api/` + transport selector di `cmd/` | Tool dengan efek samping/domain spesifik |

**Aturan tambahan:** komentar kode singkat berbahasa Inggris, dokumentasi
pengguna berbahasa Indonesia; tidak ada identitas konsumen (`ngodingmcp/...`)
di dalam inti; tidak ada dependency baru tanpa justifikasi.

### 3.4 Identitas & Konfigurasi — Satu Sumber Kebenaran

* `mcpserver.DefaultName` / `DefaultVersion` adalah satu-satunya sumber default.
* `mcpserver.New` me-resolve nama: `opts.Name` → `host.Name()` → `DefaultName`;
  versi: `opts.Version` → argumen `version` → `host.Version()` →
  `DefaultVersion`. Hasil resolusi dikirim langsung ke `Implementation`
  (`initialize.serverInfo`) dan `describe_server`. `Host` sengaja tidak
  dibungkus, supaya method domain tambahan tidak hilang.
* `app.Config` meneruskan `Title` dan `Instructions` ke `mcpserver.Options`.
  `LogWriter` menggantikan `Logger` (terdokumentasi).
* `api.Config` adalah alias `app.Config`, sehingga field baru otomatis tersedia
  tanpa duplikasi definisi.
* `MetaNamespace` dihapus dari `Config`, `Options`, dan konvensi (batch
  `v2026.11.1`): plumbing-nya mati sejak `_meta` `run_command` tidak ikut ke
  template. `ngodingmcp` tetap memakai namespace sendiri saat di-backport.

### 3.5 Kontrak Tool & Anggaran Token

* Nama, deskripsi, anotasi, `inputSchema`, dan `outputSchema` dikunci oleh
  `TestToolContract` + `testdata/tools.golden.json`.
* `TestToolsListBudget` menjaga `tools/list` (saat ini 601 byte dari 2048 byte).
  **Dilarang** menaikkan budget tanpa keputusan sadar + update golden di commit
  yang sama.
* Regenerasi: `go test ./internal/mcpserver -run TestToolContract -args -update`,
  lalu review diff.
* Kebijakan payload: teks besar/listing → `addTextTool`; record kecil yang
  benar-benar dibutuhkan client → `mcp.AddTool` dengan output typed.
  `addUntypedTool` (dari `ngodingmcp`) tidak di-port karena belum ada pemakainya;
  polanya tercatat di `AGENTS.md`.
* `defaultInstructions` masih menyebut `ping`/`describe_server`; kalau tool
  diganti, `Instructions` **wajib** diisi oleh konsumen (lihat S2 di roadmap).

### 3.6 Transport & Keamanan

* **stdio (default):** stdout eksklusif untuk protokol MCP; logger ke stderr.
* **HTTP:** `mcp.NewStreamableHTTPHandler` dengan `Stateless: true`;
  `ReadHeaderTimeout=10s`, `ReadTimeout=30s`, `IdleTimeout=120s`;
  **`WriteTimeout=0`** karena SSE stream bukan request biasa. Graceful shutdown
  via `signal.NotifyContext` + `Shutdown` 10s.
* Auth/TLS **diserahkan** ke aplikasi/reverse proxy; ini keputusan sadar, bukan
  utang teknis. Bind default `127.0.0.1`.

### 3.7 Kontrak Activity Log

* Satu baris per `tools/call`: `tool: <target> ✓/✗ <durasi> ~<tok> tok`.
* Target berasal dari **allowlist eksplisit**; `command`/`args` tidak dicatat;
  URL direduksi ke hostname (tanpa userinfo/port/query/fragment).
* File contents, edit text, env, token, signature, dan pesan error tidak pernah
  masuk log. `source_url` memakai host saja.
* Token adalah **estimasi bytes/4**, bukan hitungan eksak; suffix memakai
  payload terbesar (`content` atau `structuredContent`) agar tidak double count.
  Atribut slog tetap menyimpan byte/estimasi token per payload.
* Perubahan pada format baris/prefix berisiko memutus konsumen seperti TUI
  `ngodingmcp`; perlakukan sebagai perubahan kontrak.

### 3.8 Source-of-Truth & Prosedur Backport

Keputusan: **MayaMCP memimpin perbaikan inti MCP; `ngodingmcp` yang
meng-port.** Alasan: template lebih kecil, lebih cepat dites, dan tidak membawa
domain/appliance yang menyamarkan bug inti.

Checklist saat ada perbaikan inti:

1. Terapkan perubahan di `mayamcp` lebih dulu, disertai test (Prove-It:
   test gagal lebih dulu untuk bugfix).
2. Jalankan `make check` di `mayamcp` dan pastikan golden/budget masih valid.
3. Port ke `ngodingmcp`, sesuaikan namespace identitas (`ngodingmcp`, 46 tool)
   dan awasi Konvensi AGENTS.md upstream.
4. Yang **tidak** di-backport otomatis: identitas/default, resource URI,
   `_meta` `run_command`, cookie/auth header, dan tool domain.
5. Catat tanggal + CalVer di audit trail dokumen ini bila perubahan bersifat
   arsitektural.

### 3.9 Testing & Quality Gate

* `make test` — `go test -count=1 ./...`.
* `make check` — gofmt, `go vet`, staticcheck versi terkunci, `go test -race`.
  Ini gate sebelum selesai (Stop-the-Line: gagal = perbaiki, jangan tambah fitur).
* Unit test helper (`activityTarget`, `ActivityLine`, `measureResult`,
  `humanCount`), test identitas via in-memory session, golden contract, budget.
* Bugfix wajib memakai **Prove-It Pattern**: reproduksi merah → perbaikan → hijau.
* Coverage setelah batch `v2026.11.2`: `api` 100%, `app` 96,0%,
  `mcpserver` 95,6%, `cmd` 48,1%, total 86,0%. `activityMiddleware` 100% dan
  `runHTTP` 90,9%; `serveHTTP` masih 0% karena hanya membungkus signal wiring.

### 3.10 Versioning

CalVer `vYYYY.MM.MINOR`. Rilis saat ini `v2026.11.1`. Perubahan kontrak tool
atau format activity log menaikkan `MINOR`; perubahan perilaku non-kontrak
mencatat audit trail tanpa mengganti angka rilis mayor.

### 3.11 Konsekuensi Negatif & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Duplikasi perbaikan dua repo | Aturan source-of-truth (§3.8) + checklist backport |
| Core tumbuh jadi appliance | Tabel batas (§3.3) jadi checklist review tiap PR |
| Regresi privasi log | Test regresi secret + Prove-It Pattern |
| Kontrak/tools/list membengkak | Golden + budget 2048; kenaikan butuh keputusan sadar |
| `defaultInstructions` stale setelah ganti tool | Selesai (S2): default generik, konsumen mengisi `Instructions` saat ganti tool |
| `MetaNamespace` plumbing mati | Selesai (S3): dihapus dari API + konvensi |
| Fasad `api` incomplete | Selesai (S4): `ActivityPrefix`, `ActivityLine`, `CountTools` diekspor |

---

## 🗺️ 4. Roadmap Vertical Slices

| Slice | Tujuan | Status |
|---|---|---|
| S1 | Test end-to-end `activityMiddleware` (Info/Warn, prefix, atribut slog) dan shutdown HTTP | `Completed` (`v2026.11.1`) |
| S2 | Hilangkan `defaultInstructions` stale: instruksi eksplisit atau derivasi dari daftar tool aktif | `Completed` (`v2026.11.1`) |
| S3 | Putuskan `MetaNamespace`: hapus plumbing mati atau beri fungsi nyata (selaras dengan AGENTS.md) | `Completed` (`v2026.11.1`) |
| S4 | Lengkapi fasad `api/` dengan `CountTools`, `ActivityLine`, `ActivityPrefix` bila dibutuhkan TUI/konsumen | `Completed` (`v2026.11.1`) |
| S5 | Ratakan `go.mod` (`go 1.26.10` vs toolchain 1.27.x) + kebijakan `toolchain` | `Completed` (`v2026.11.1`) |
| S6 | Formalisasi sinkronisasi mayamcp → ngodingmcp (checklist backport + paritas CI) | `Completed` (`v2026.11.1`) |

---

## 🚫 5. Non-Goal (Tidak Akan Dimasukkan ke Core)

Auth/OAuth/session, Web UI/TUI, systemd, config store, workspace/filesystem,
resource `ngodingmcp://agents`, prompts, module opt-in, health check, metrik
kustom, dan tool domain apa pun. Semua itu milik aplikasi yang menanam template.

---

## ⏱️ 6. Riwayat Revisi & Audit Trail

| Tanggal & Waktu | Versi / Commit | Penulis | Ringkasan Perubahan |
|---|---|---|---|
| `2026-11-25 15:31:20 +0900` | `v2026.11.0` | AI / Developer | Penyusunan guideline arsitektur pasca audit + perbaikan; keputusan profil minimal, source-of-truth mayamcp, dan roadmap S1–S6 |
| `2026-11-25 16:00:56 +0900` | `v2026.11.1` | AI / Developer | Eksekusi S1–S6: test runtime, instruksi generik, penghapusan `MetaNamespace`, fasad `api`, go.mod/toolchain, CI; update status roadmap |
| `2026-11-25 16:37:55 +0900` | `v2026.11.2` | AI / Developer | Eksekusi S7–S12: simetri identitas, `TargetExtractor` + `byteCounter`, log JSON/level, `/healthz`, tool example, Makefile/CI; identitas tidak lagi dibungkus `identityHost` |

---

## 📚 7. Referensi

* `docs/mcp-as-template.md` — catatan asli ekstraksi dari `ngodingmcp`.
* `README.md`, `AGENTS.md` — konvensi dan cara pakai template.
* `internal/mcpserver/testdata/tools.golden.json` — kontrak tool terkunci.
* `ngodingmcp/docs/development.md` (upstream) — batas core vs appliance.
