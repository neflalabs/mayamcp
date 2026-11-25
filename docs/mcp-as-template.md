# MayaMCP — Panduan Refactor Struktur NgodingMCP & Ekstraksi Template MCP

> **Provenansi:** dokumen ini disalin dari `ngodingmcp/mcp-as-template.md` pada
> 2026-11-25. Status, hash commit, dan tautan TaskDev di bawah merujuk ke repo
> `ngodingmcp` (upstream), bukan ke repo template ini.

Nama resmi target untuk fondasi/template MCP server berbasis Go ini adalah **MayaMCP** (dengan pendekatan kerja **keep as is dulu**: rapikan struktur internal `ngodingmcp` secara bertahap tanpa merusak perilaku yang berjalan, baru ekstrak **MayaMCP** setelah struktur beres).

> **Status terkini (2026-11-25, kode `b9bdb22`):** catatan ini masih *scratch* lokal, belum masuk TaskDev. Bagian "perapihan struktur" di §4 sudah **dikerjakan**: Foundation 03/06 selesai sebelumnya, lalu decouple + standarisasi (commit `5427e0c`, `1270761`, `b9bdb22`). **Tidak ada rebranding ke MayaMCP** — ekstraksi tetap akan dilakukan di repo terpisah.

---

## 1. Filosofi Nama: **MayaMCP**

* **Bunyi & Pelafalan (*International-Friendly*):** Terdengar seperti nama perempuan yang akrab, singkat, dan sangat mudah dilafalkan penutur bahasa Inggris (*MAH-yah* + `MCP`), sekaligus punya karakter *brand* yang kuat dan *sellable*.
* **Akar Makna & Potensi Bentuk:** Dalam khazanah bahasa Indonesia/Sanskerta, *maya* lekat dengan nuansa *ciptaan, wujud yang terbentuk, daya ubah, dan keluasan bentuk*. Dalam konteks **MayaMCP**, maknanya pas sebagai **titik awal (*genesis / seed*) yang belum terkunci ke satu domain sempit**, namun siap diwujudkan menjadi MCP server apa pun—mulai dari tool lokal, layanan internal, integrasi API, hingga server domain spesifik.
* **Relasi dengan `NgodingMCP`:** `NgodingMCP` lahir dari kebutuhan spesifik *sandboxed editor & workspace* plus *control panel* operasional. **MayaMCP** mengambil disiplin rekayasa dan pola MCP-nya, lalu menyediakannya sebagai fondasi yang lebih bersih, fleksibel, dan siap dikembangkan ke berbagai arah tanpa membawa beban identitas lama.

---

## 2. Strategi Umum: Rapikan Struktur Dulu, Ekstrak Belakangan

* **Kenapa diperbaiki dulu di `NgodingMCP`:** Melakukan refactor struktur sambil sekaligus mencabut fitur ke **MayaMCP** berisiko memecah alur yang sedang berjalan (terutama *workspace*, OAuth, Web UI, TUI, dan *contract/budget tests* di `internal/mcpserver/`).
* **Urutan aman:**
  1. **Keep as is dulu:** Pertahankan kondisi kode yang sedang jalan sembari mengunci titik-titik perapihan.
  2. **Rapikan batas tanggung jawab di `ngodingmcp`:** Pisahkan batas antara *MCP server orchestration* dan *domain state/UI/service wiring*. **(Sebagian sudah selesai — lihat §4.)**
  3. **Ekstrak ke MayaMCP:** Setelah struktur stabil dan seluruh *test* hijau, angkat inti MCP ke *starter template* **MayaMCP** dan sesuaikan identitasnya.

---

## 3. Hal yang Dipertahankan (*Keep As Is*) Selama Fase Perapihan

* **Kontrak & perilaku tool yang sudah berjalan:** Jangan mengubah nama, skema, maupun anggaran token tool secara sembarangan tanpa memperbarui *golden/budget test* (`tools_contract_test.go`, `tools_budget_test.go`, `testdata/tools.golden.json`).
* **Transport Streamable HTTP stateless (`Stateless: true`):** Tetap pertahankan agar *restart* atau *rebuild* saat pengembangan tidak memutus sesi klien akibat `Mcp-Session-Id`.
* **Keamanan & *activity logging* yang sudah ada:** `safeErrorMiddleware`, `activityMiddleware`, proteksi path/file sensitif, dan autentikasi yang sudah terpasang tetap dijaga selama struktur dipecah/dirapikan.

---

## 4. Area Struktur yang Akan Dirapikan Sebelum Ekstrak ke MayaMCP

> **Koreksi 2026-11-25:** bagian A dan B di bawah sudah ditulis ulang untuk memisahkan **yang sudah selesai** dari **sisa pekerjaan**. Rujukan: task
> `ngodingmcp:TaskDev/01_Foundation_And_Architecture/03_MCP_Server_Tool_Layout_Refactor.md` (`Completed`) dan
> `ngodingmcp:TaskDev/01_Foundation_And_Architecture/06_Token_Efficiency_Plan.md` (`Completed`) — keduanya ada di repo upstream, bukan di template ini. Angka lama di dokumen itu (26 core / 34 tool) sudah usang; kondisi sekarang **46 tool, 38 core**.

### A. Batas Tanggung Jawab `internal/mcpserver/` vs. `*workspace.Workspace`
* **Sudah selesai (Foundation 03):** registrasi tool dipecah satu file per domain (`tools_*.go`), `server.go` kini ~102 baris berisi `Options`, konstruktor, daftar registrasi, dan wiring middleware. Helper bersama pindah ke `helpers.go`/`register.go`; modul opt-in ada di `modules.go`; resource ada di `resources.go`. Kontrak tool dikunci `TestToolContract` + `testdata/tools.golden.json`, budget dikunci `TestToolsListBudget`.
* **Sudah selesai (2026-11-25, `5427e0c`):** `internal/mcpserver` tidak lagi menerima `*workspace.Workspace` konkret — memakai interface `mcpserver.Workspace`. `fetch`, `gitread`, dan `gitwrite` juga punya interface sempit sendiri, jadi tidak bergantung pada tipe konkret workspace. Identitas server, `instructions`, dan URI resource kini opsi `mcpserver.Options` dengan default yang sama seperti sebelumnya (`1270761`).
* **Masih melekat pada domain workspace:** tipe-tipe data tool (`workspace.ReadResult`, `Edit`, `Entry`, `Match`, dll) tetap dipakai sebagai payload tool. Ini bukan coupling ke implementasi, tapi domain tool memang harus diganti saat ekstraksi ke template.

### B. Pemisahan *Core Server* dari *App / UI / Service Wiring*
* **Sudah selesai:** `internal/app/core.go` (`b9bdb22`) mengekspos `NewCore(Config) (*Core, error)` yang membangun workspace + MCP server + runner/procman/downloader tanpa HTTP, auth, Web UI, TUI, atau systemd. `app.New` sekarang tinggal menggabungkan `Core` dengan lapisan appliance (`Stateless: true` tetap di `app.go`).
* **Masih di lapisan appliance:** `app.go` merangkai token/auth/OAuth, Web UI, mux HTTP, dan `Server`; itu memang tidak perlu ikut ke template generik. `internal/service/` dan `service_cmd.go` tetap di luar inti MCP.
* **Sisa sebelum ekstraksi:** putuskan profil transport (stdio / HTTP stateless / full appliance) dan *source-of-truth* antar-repo — bukan lagi soal coupling kode.

### C. Inventaris Identitas, Prefix, & Metadata Spesifik `ngodingmcp`
Inventaris konkret yang harus ikut daftar saat *rebranding* (bukan hanya daftar umum):

| Jenis | Contoh lokasi / nilai |
|---|---|
| Module path Go | `ngodingmcp/internal/...` di hampir seluruh file Go |
| Nama binary & service | `ngodingmcp`, perintah `serve`/`tui`/`restart`, unit systemd |
| Resource MCP | `ngodingmcp://agents` (default `mcpserver.Options.AgentsURI`, `server.go:23`) |
| Metadata `_meta` | `ngodingmcp/exitCode`, `ngodingmcp/timedOut` (`tools_run.go:62-63`) |
| Cookie sesi | `ngodingmcp_session` (`app.go:29`, `auth.go`) |
| Auth header | `WWW-Authenticate: Bearer realm="ngodingmcp"` (`auth.go:104`) |
| Config & workspace default | `~/.config/ngodingmcp/config.json`, `~/ngodingmcp-workspace` (`config/`) |
| Kontrak & docs | `testdata/tools.golden.json`, `AGENTS.md`, `README.md`, `docs/`, `TaskDev/` |

*Catatan: mengganti prefix `_meta` dan URI resource **akan mengubah kontrak** — golden test dan `tools/list` budget harus diperbarui dalam commit yang sama, sesuai `AGENTS.md`.

---

## 5. Rencana Ekstraksi **MayaMCP** (Setelah Struktur Beres)

> **Keputusan menyusul (2026-11-25):** daftar di bawah adalah rencana asli. Dua
> keputusan terbuka sudah ditutup di
> `TaskDev/01_Foundation_And_Architecture/01_MayaMCP_Architecture_Guideline.md`:
> §3.6 memilih **stdio default + Streamable HTTP stateless**, §3.8 memilih
> **mayamcp sebagai source-of-truth** yang di-port ke `ngodingmcp`.

1. **Tentukan profil target MayaMCP:** *stdio* minimal, HTTP *stateless* saja, atau HTTP lengkap dengan auth/UI.
2. **Ekstrak lapisan MCP + helper payload (`register.go`, pola `tools_<domain>.go`, *middleware* generik)** ke fondasi **MayaMCP**.
3. **Ganti *domain layer*** dengan domain MCP tujuan baru, sesuaikan *instructions* & *resource*. *Contract test* dan *token budget test* sudah ada di `ngodingmcp` (`tools_contract_test.go`, `tools_budget_test.go`) — tinggal diadopsi di template, bukan dibuat dari nol.
4. **Putuskan dulu source-of-truth:** apakah perbaikan inti MCP selanjutnya masuk `ngodingmcp` lalu di-*port* ke MayaMCP, atau sebaliknya. Tanpa keputusan ini, dua repo berisiko divergen.
