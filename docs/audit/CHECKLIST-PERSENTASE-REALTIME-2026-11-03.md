# Checklist fungsi per halaman setelah pembaruan real time — 3 November 2026

## Cara membaca persentase

Ini adalah **skor fungsi inti yang terverifikasi dalam audit**, bukan estimasi waktu pengerjaan, SLA, atau klaim seluruh fitur mockup selesai. Setiap halaman dinilai dengan 10 kriteria yang sama; setiap kriteria bernilai 10 poin. ✓ = bukti yang cukup pada lingkup API/source/test saat audit; — = belum lengkap atau belum terverifikasi. Tidak ada kredit parsial. Kriteria J belum lulus pada halaman lain. Fleet Dashboard telah menjalani QA browser fungsi utamanya di desktop/mobile serta tes akses semua peran; lihat checklist final dashboard. Fitur hardware/layanan tambahan yang belum tersedia tetap dicantumkan dalam catatan, meskipun tidak seluruhnya masuk 10 kriteria inti. Skor tidak boleh dibaca sebagai persentase seluruh tombol di mockup.

- **A**: Endpoint GET live merespons 200.
- **B**: Data aktual domain tersedia (bukan hanya empty state).
- **C**: Sumber dan scope kendaraan sesuai kebutuhan halaman.
- **D**: Field inti tampilan sesuai data domain.
- **E**: Filter/rentang/pagination yang relevan terikat dan diperiksa.
- **F**: Aksi utama halaman diverifikasi otomatis/integrasi terisolasi.
- **G**: Pembaruan incremental terpasang pada tampilan utama (bukan sekadar koneksi SSE).
- **H**: Penanganan empty/error tersedia dalam binding aktif.
- **I**: Endpoint melewati autentikasi/role guard yang relevan.
- **J**: Visual dan klik ujung-ke-ujung semua fungsi halaman telah diuji browser.

## Pembaruan real time yang sudah deploy

- [x] SSE mengirim delta per field, bukan event kosong yang memicu reload seluruh halaman.
- [x] Snapshot awal dikirim saat koneksi dibuka/dibuka kembali; event berikutnya hanya mengirim field/urutan/penghapusan yang berubah.
- [x] Tidak menggunakan setInterval atau setTimeout(load) untuk mengambil ulang data real time.
- [x] Receiver tetap memicu event sesudah commit melalui Unix socket; bridge receiver tidak diubah pada pembaruan ini.
- [x] Tabel mempertahankan node baris dan sel yang tidak berubah; filter dan cursor masuk ke subscription.
- [x] Marker tracking dipertahankan dan dianimasikan menuju koordinat laporan baru dengan requestAnimationFrame; animasi mengikuti prefers-reduced-motion. Ini interpolasi visual antarlaporan, bukan koordinat GPS baru.
- [x] Grafik fuel dan daftar tracking diperbarui di tempat.
- [x] Lokal memakai stream authoritative live melalui proxy.
- [x] Autentikasi diperiksa kembali sebelum push; role API yang normal tetap berlaku.
- [x] Heartbeat 25 detik hanya menjaga koneksi/validasi sesi, bukan polling data; reconnect SSE memiliki backoff bawaan.
- [ ] Belum ada bukti device fisik mengirim posisi baru selama pengujian ini. Data lama tetap lama; SSE tidak memperbaiki device offline atau timestamp perangkat.
- [ ] Feed non-table Raw/Alert/Notifications dan diagram ringkasan dashboard belum menggunakan patch incremental khusus.
- [ ] Grafik/playback riwayat koordinat tidak menyisipkan titik event otomatis selama playback aktif.
- [ ] Status freshness yang bergantung waktu tanpa event baru belum memiliki transisi deadline khusus; akan dihitung ulang ketika snapshot/event/reconnect berikutnya terjadi.
- [ ] Pengujian beban banyak subscriber serta visual animasi browser masih diperlukan. Server membaca snapshot resource yang disubscribe per event dan membandingkannya; belum ada cache fan-out bersama antarsubscriber.

## Skor 17 menu utama

| Halaman | Route | A | B | C | D | E | F | G | H | I | J | Skor | Sisa fungsi/batas |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Fleet Dashboard | `#fleet-dashboard` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | **100%** | Checklist penerimaan terbaru 20/20 lulus; QA desktop/mobile dan akses peran diuji. Gateway offline tetap offline; tidak ada klaim pengujian fisik baru. |
| Vehicles | `#vehicles` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | **100% registry** | Daftar dipisahkan dari inspector. Checklist registry15/15 lulus; halaman detail kendaraan tetap diaudit terpisah. |
| Live Tracking | `#live-tracking` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | **90%** | Marker dan daftar dipatch; traffic/satellite belum tersedia; belum ada bukti kiriman fisik terbaru. |
| Trip History | `#trip-history` | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | — | **80%** | Range WIB dan koordinat/speed tersedia; belum berupa ringkasan trip tersegmentasi. |
| Geofences | `#geofences` | ✓ | — | ✓ | — | ✓ | — | — | ✓ | ✓ | — | **50%** | Belum ada geofence/assignment aktual; proses transisi dan drawing browser belum diuji. |
| Fuel Monitoring | `#fuel-monitoring` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | **90%** | Liter/persen valid, tabel/grafik mendapat delta; waktu pengukuran historis sebagian tidak tersedia. |
| Fuel Analytics | `#fuel-analytics` | ✓ | — | — | — | ✓ | — | — | ✓ | ✓ | — | **40%** | Summary konsumsi kosong, proses agregasi belum ada, chart domain belum sesuai. |
| Fuel Events | `#fuel-events` | ✓ | — | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **60%** | Endpoint/tabel menerima delta; belum ada event nyata, layout analytics/event perlu dipisahkan. |
| GPS Quality | `#gps-quality` | ✓ | ✓ | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **70%** | Data ada; chart kualitas GPS khusus belum tersedia. |
| Device Connectivity | `#device-connectivity` | ✓ | ✓ | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **70%** | Log jaringan ada; status heartbeat dan metrik jaringan belum lengkap. |
| Data Activity | `#data-activity` | ✓ | ✓ | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **70%** | Parse log tersedia; grafik throughput/latency domain belum lengkap. |
| Raw Protocol Logs | `#raw-protocol-logs` | ✓ | ✓ | ✓ | — | ✓ | — | — | ✓ | ✓ | — | **60%** | List raw non-table belum dipatch incremental; clipboard/detail browser/auto-scroll perlu uji/perbaikan. |
| Alert Center | `#alert-center` | ✓ | — | ✓ | — | ✓ | — | — | ✓ | ✓ | — | **50%** | Tidak ada alert nyata; feed non-table belum dipatch, layanan triage eksternal belum tersedia. |
| Reports | `#reports` | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | ✓ | ✓ | — | **70%** | Generator diuji pada DB uji; list live kosong, download browser dan schedule belum lengkap. |
| Device Management | `#device-management` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | **90%** | Mapping GPS idempotent dan konflik diuji; hardware command tetap belum tersedia. |
| Sensor Management | `#sensor-management` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | **90%** | Raw/relasi/kalibrasi berfungsi; liter/persen belum digabung langsung ke baris sensor. |
| User Management | `#user-management` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | **90%** | CRUD/roles diuji terisolasi; 2FA dan permission granular belum tersedia. |

## Skor halaman detail dan route tambahan

| Halaman | Route | A | B | C | D | E | F | G | H | I | J | Skor | Sisa fungsi/batas |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Vehicle Overview | `#vehicles/1/overview` | ✓ | ✓ | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **70%** | KPI kendaraan belum lengkap; pembaruan field tabel yang tersedia mengikuti delta. |
| Vehicle Trips | `#vehicles/1/trips` | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | — | **80%** | Koordinat/speed/range ada; trip tersegmentasi kosong. |
| Vehicle Fuel & BLE | `#vehicles/1/fuel` | ✓ | ✓ | — | — | ✓ | — | ✓ | ✓ | ✓ | — | **60%** | Scope tabel sensor tambahan masih perlu dibenahi; endpoint fuel kendaraan benar. |
| Vehicle Calibration | `#vehicles/1/calibration` | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | ✓ | — | **80%** | Kalibrasi aktif ada; jalur UI lama masih perlu disatukan dengan Empty/Full. |
| Vehicle Device Info | `#vehicles/1/device` | ✓ | ✓ | ✓ | ✓ | — | — | ✓ | ✓ | ✓ | — | **70%** | Info GPS terbaca; command hardware tidak tersedia. |
| Route Playback | `#playback/1` | ✓ | ✓ | ✓ | ✓ | ✓ | — | — | ✓ | ✓ | — | **70%** | Play/scrub tersedia di source; hanya halaman titik aktif, tidak menyisipkan delta ke playback historis. |
| Notifications | `#notifications` | ✓ | ✓ | ✓ | — | — | — | — | ✓ | ✓ | — | **50%** | Title/body belum sesuai renderer; mark-read/push dan delta feed belum lengkap. |
| Audit Log | `#audit` | ✓ | ✓ | ✓ | — | — | — | ✓ | ✓ | ✓ | — | **60%** | Endpoint berisi audit; layout/header masih memakai User Management. |
| Device Models | `#models` | ✓ | ✓ | ✓ | — | — | — | ✓ | ✓ | ✓ | — | **60%** | Data model ada; layout inventaris dipakai ulang; CRUD belum diverifikasi khusus halaman. |
| Internal Calibration Capacity | `#tanks` | ✓ | ✓ | ✓ | — | — | — | ✓ | ✓ | ✓ | — | **60%** | Profil internal kapasitas kalibrasi ada; bukan spesifikasi pabrikan, UI generik perlu perapihan. |
| Calibration Versions | `#calibrations` | ✓ | ✓ | ✓ | — | — | ✓ | ✓ | ✓ | ✓ | — | **70%** | Versi approved dan unverified tersedia; layout lama dan pemilihan versi perlu perapihan. |
| Event Types | `#event-types` | ✓ | — | ✓ | — | — | — | — | ✓ | ✓ | — | **40%** | Katalog kosong; memakai layout alert non-table yang belum dipatch. |
| Event Severities | `#event-severities` | ✓ | — | ✓ | — | — | — | — | ✓ | ✓ | — | **40%** | Katalog kosong; memakai layout alert non-table yang belum dipatch. |
| Geofence Assignments | `#geofence-assignments` | ✓ | — | ✓ | — | — | — | — | ✓ | ✓ | — | **40%** | Tidak ada assignment aktual; layout generik, belum diuji ujung ke ujung. |
| Sensor Telemetry | `#sensor-telemetry` | ✓ | ✓ | ✓ | — | ✓ | — | ✓ | ✓ | ✓ | — | **70%** | Data raw ada; tabel generic menerima delta, kolom telemetri khusus perlu dilengkapi. |

## Bukti pengujian

- `go test -race ./...` dengan PASTRACK_TEST_DSN menunjuk pastrack_test_local pada port 33307: lulus.
- `go vet ./...`: lulus.
- 18 pengujian DOM lulus, termasuk bukti event tidak menambah fetch, node baris dipertahankan, dan tidak ada timer reload real time.
- Tes unit diff membuktikan hanya field yang berubah dikirim, field null dan record removal ditangani, serta snapshot identik tidak menimbulkan delta.
- SSE publik live dan lokal HTTP 200; initial delta memuat fuel_percent 93.85 untuk pembacaan 244. Bukti JSON: `realtime-verification-20261103.json`.
- Belum dilakukan uji mutasi data bisnis live atau replay data palsu ke receiver produksi.

## Urutan perbaikan berikutnya

1. Scope kendaraan pada tabel Fuel & BLE, lalu field liter/persen di Sensor Management.
2. Segmentasi trip dan agregasi Fuel Analytics; pastikan data migrasi tervalidasi dapat dihitung sesuai aturan.
3. Patch feed Raw/Alert/Notifications, renderer Audit/Notifications, dan diagram dashboard/freshness deadline.
4. Playback seluruh lintasan, visual QA browser, dan uji beban/reconnect serta kehilangan event.

Lihat audit awal `CHECKLIST-HALAMAN-2026-11-03.md` untuk daftar temuan rinci sebelumnya. Persentase pada dokumen ini menggambarkan kondisi setelah pembaruan stream dan tidak menyatakan temuan lainnya telah diperbaiki.

Pembaruan Fleet Dashboard: lihat `FLEET-DASHBOARD-2026-11-03.md` untuk bukti penyelesaian dan batas yang tersisa.

## Penyelesaian lanjutan Fleet Dashboard

Checklist fungsi dashboard kini lulus 20/20 (100%) pada `FLEET-DASHBOARD-FINAL-2026-11-03.md`. Angka ini adalah hasil checklist penerimaan rinci terbaru, bukan jaminan bebas bug atau bukti gateway sedang online. Skor dashboard diperbarui pada tabel; halaman lain tetap memakai status audit masing-masing.

Pembaruan Vehicles: lihat `VEHICLES-FINAL-2026-11-03.md`. Checklist registry15/15 lulus; angka ini tidak menyelesaikan subhalaman overview/trips/fuel/device/calibration secara otomatis.
