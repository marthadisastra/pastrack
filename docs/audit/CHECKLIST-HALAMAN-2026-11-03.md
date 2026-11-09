# Audit fungsi dan data PasTrack — 3 November 2026

Lingkup: source aktif `internal/platform`, aplikasi live https://toko.azlaundry.my.id, database `pastrack_v2_20261103_r2`. Pemeriksaan authenticated GET live, inspeksi binding UI dan proses backend, pengujian DOM, serta pengujian integrasi pada database lokal terisolasi. Tidak ada perubahan data bisnis live atau deployment selama audit. Login audit membuat sesi autentikasi normal.

**Batas audit:** tidak dilakukan pemeriksaan visual browser atau klik manual menyeluruh. HTTP 200 membuktikan endpoint merespons, bukan membuktikan setiap tombol dan nilai pada layar benar. CRUD bisnis live tidak dijalankan. Daftar record pada hasil API dibatasi 200; bukan jumlah total tabel jika `has_more=true`. Aksi yang hanya ditemukan pada source ditandai belum diverifikasi ujung ke ujung.

## Bukti pemeriksaan

- [x] 37 endpoint GET live merespons HTTP 200; rincian di `live-api-20261103.json`.
- [x] 17 pengujian DOM lulus; rincian di `dom-tests-20261103.txt`.
- [x] `go test -race ./...` lulus dengan database pengujian terisolasi.
- [x] `go vet ./...` lulus.
- [x] Fuel terakhir: 112.618 L / 93.85%, raw height 4699, calibration_id 4.
- [x] Satu kendaraan, satu gateway, satu sensor teridentifikasi dalam respons live.
- [x] Ignition fisik tetap NOT_INSTALLED.
- [ ] Data real time baru belum terverifikasi: posisi terakhir 2 November 2026 17:52:44 WIB; penerimaan terakhir 3 November 2026 13:23:29 WIB. Status STALE benar, bukan posisi real time.
- [ ] Verifikasi visual, responsif, peta tile, dan seluruh interaksi pengguna pada browser masih diperlukan.

Status **Parsial** berarti fungsi dasar tersedia tetapi ada batas/temuan; **Kosong** berarti endpoint sehat namun belum memiliki data; **Perlu perbaikan** berarti ketidaksesuaian source terhadap fungsi halaman sudah ditemukan. Tidak ada klaim seluruh halaman selesai 100%.

## Checklist 17 halaman menu utama

| Halaman / route | Checklist data dan fungsi | Status / temuan |
|---|---|---|
| Fleet Dashboard `#fleet-dashboard` | [x] fleet/live dan overview; [x] 1 kendaraan; [x] feed aktivitas; [ ] kesegaran real time; [ ] seluruh KPI | **Parsial.** GPS STALE. Nilai offline tidak disediakan query overview; UI hanya memberi fallback 0 jika fleet kosong, sehingga KPI offline tidak lengkap. |
| Vehicles `#vehicles` | [x] list 1 kendaraan; [x] DOM register kendaraan; [x] detail routes; [ ] seluruh edit/delete live | **Parsial.** CRUD diuji pada DB terisolasi, bukan perubahan bisnis live. |
| Live Tracking `#live-tracking` | [x] koordinat/speed terakhir; [x] source marker/popup; [ ] kiriman baru; [ ] visual peta | **Parsial.** Peta memakai posisi terakhir. Traffic/satellite/heatmap belum terintegrasi; beberapa kontrol hanya menjelaskan keterbatasan. |
| Trip History `#trip-history` | [x] gps/positions; [x] range inklusif WIB; [x] pagination; [ ] ringkasan perjalanan | **Parsial.** Halaman memuat posisi GPS, bukan hasil segmentasi trip. `/trips` kosong. Filter default hari ini tidak mencakup data GPS 1–2 November; kosong pada hari ini bukan kehilangan data. |
| Geofences `#geofences` | [x] endpoint; [x] source create/edit/draw; [ ] geofence aktual; [ ] transisi masuk/keluar | **Kosong / belum tervalidasi ujung ke ujung.** Belum ada geofence atau assignment. |
| Fuel Monitoring `#fuel-monitoring` | [x] riwayat pembacaan; [x] liter/persen; [x] raw + validity; [x] Tank Spec dihapus; [ ] freshness fuel pada layar | **Parsial.** 219 valid dan 25 out_of_range pada rekalkulasi sebelumnya. `reading_at` null; grafik memakai received_at sebagai fallback. Waktu penerimaan tidak boleh disamakan dengan waktu pengukuran. |
| Fuel Analytics `#fuel-analytics` | [x] endpoint; [ ] summary konsumsi; [ ] grafik analitik sesuai metrik | **Perlu perbaikan.** `fuel_consumption_summaries` kosong; tidak ditemukan proses pengisian tabel pada backend aktif. Renderer chart hanya menerima fuel_liters + validity, sedangkan endpoint analytics berisi consumed_l. |
| Fuel Events `#fuel-events` | [x] endpoint; [x] source deteksi kejadian tervalidasi; [ ] kejadian nyata | **Kosong / parsial.** Tidak ada fuel event. Layout dibagi dengan Analytics; beberapa elemen statistik/grafik tidak sesuai bentuk data event. Tidak boleh menganggap kosong sebagai bukti tidak pernah ada theft/refuel. |
| GPS Quality `#gps-quality` | [x] endpoint berisi data; [ ] seluruh kolom/grafik kualitas | **Parsial.** Data tersedia (lebih dari 200 record); renderer chart umum hanya fuel sehingga grafik kualitas belum khusus GPS. |
| Device Connectivity `#device-connectivity` | [x] 2 log jaringan; [ ] status heartbeat pada halaman; [ ] seluruh statistik jaringan | **Parsial.** Sumber utama network_status_logs, bukan devices/health; sebagian field tampilan dapat N/A karena payload belum mengirim field tersebut. |
| Data Activity `#data-activity` | [x] parse log tersedia; [x] pagination; [ ] latency/throughput trend | **Parsial.** Endpoint log parser sehat; tidak ada chart time-series khusus aktivitas pada renderer umum. |
| Raw Protocol Logs `#raw-protocol-logs` | [x] list pesan; [x] payload detail tersedia pada source; [x] admin guard; [ ] clipboard browser; [ ] auto-scroll nyata | **Parsial.** Auto-scroll hanya mengubah dataset/toast pada handler; tidak ditemukan aksi scroll terkait. Detail payload/clipboard belum diuji manual browser dalam audit ini. |
| Alert Center `#alert-center` | [x] endpoint; [x] source acknowledge; [ ] alert nyata; [ ] operasional triage lengkap | **Kosong / parsial.** Tidak ada alert. Telepon/WhatsApp/foto/BAP dan layanan eksternal dinonaktifkan, bukan fitur selesai. |
| Reports `#reports` | [x] API list; [x] generator/download di backend; [x] integrasi pengujian; [ ] ekspor live/download visual; [ ] schedule | **Parsial.** List report live kosong. Schedule dinonaktifkan. Tidak membuat report live selama audit. |
| Device Management `#device-management` | [x] 1 GV30CEU; [x] endpoint relasi kendaraan; [x] tes idempotensi dan konflik; [ ] aksi hardware | **Parsial.** Hubungkan kendaraan tersedia; OTA/reset/poll/immobilizer belum didukung. Riwayat sensor sekarang diturunkan dari gateway oleh receiver. |
| Sensor Management `#sensor-management` | [x] 1 CFS300B via gateway; [x] raw; [x] kalibrasi 120 L aktif; [x] tes Empty/Full/draft/stale; [ ] tampilan liter/persen langsung di tabel | **Parsial.** Endpoint list sensor hanya mengirim raw dan decoding_state; fuel liter/persen tersedia di fuel/readings, belum digabung ke baris sensor. Kalibrasi admin saja. |
| User Management `#user-management` | [x] list admin; [x] role/CRUD pada DB uji; [ ] semua aksi live; [ ] 2FA | **Parsial.** 2FA belum dikonfigurasi; matriks permission pada UI berupa informasi statis, bukan editor permission granular. |

## Checklist halaman detail kendaraan dan playback

| Halaman / route | Checklist | Status / temuan |
|---|---|---|
| Telematics Overview `#vehicles/1/overview` | [x] kendaraan dan fleet terakhir; [ ] KPI lengkap | **Parsial.** API kendaraan bukan snapshot semua telemetri. Nilai yang tidak dibinding tetap N/A. |
| Trip kendaraan `#vehicles/1/trips` | [x] filter kendaraan/range WIB; [x] posisi/speed; [ ] perjalanan tersegmentasi | **Parsial.** Memakai gps/positions; `/vehicles/1/trips` kosong. |
| Fuel & BLE `#vehicles/1/fuel` | [x] API fuel kendaraan valid; [ ] scope seluruh tabel pada layar; [ ] chart sesuai sumber | **Perlu perbaikan.** `tableEndpoint()` mengganti tabel pertama menjadi sensor-telemetry dan tabel lain menjadi vehicles, bukan fuel/readings. Pengambilan tabel tambahan memakai `?limit=50` tanpa filter kendaraan. Berisiko menampilkan sensor kendaraan lain bila fleet bertambah. |
| Sensor Calibration `#vehicles/1/calibration` | [x] 1 kalibrasi approved kendaraan; [x] Register Unit mengarah Device Management; [ ] alur satu pintu konsisten | **Parsial.** Ada dua jalur UI kalibrasi: detail lama dengan points/approve dan dialog Empty/Full di Sensor Management. Layout lama perlu dirapikan agar tidak membingungkan. |
| Queclink Device Info `#vehicles/1/device` | [x] 1 GPS terhubung; [ ] kontrol perangkat | **Parsial.** Informasi protokol tersedia; perintah hardware dinonaktifkan. |
| Route Playback `#playback/1` | [x] source play/pause/scrub/rate; [x] range; [ ] keseluruhan lintasan pada semua halaman; [ ] browser | **Parsial.** Titik diproses dari `records` halaman aktif (default 50), bukan seluruh hasil lintasan; urutan API id DESC kemudian diurutkan waktu untuk playback. |

## Checklist route tambahan di luar 17 sidebar

| Route | Data saat audit | Status / temuan |
|---|---|---|
| `#notifications` | 1 notifikasi | **Perlu perbaikan.** Memakai layout Alert Center; renderer feed mencari plate/event/raw code dan mengabaikan title/body notifikasi. Mark-read dan push delivery belum terhubung pada platform aktif. |
| `#audit` | 42 audit record | **Perlu perbaikan.** Memakai layout User Management dan resource audit; header tabel tidak khusus audit, sehingga sebagian field actor/action/entity/waktu tidak dipetakan dengan benar. |
| `#models` | 1 model | **Parsial.** CRUD resource tersedia, layout inventaris device dipakai ulang dan tidak semua header sesuai model. |
| `#tanks` | 2 profil internal | **Perlu perapihan.** Route internal masih ada walau Tank Spec di monitoring dihapus; profil tetap diperlukan sebagai penyimpanan kapasitas kalibrasi, bukan bukti spesifikasi pabrikan kendaraan. |
| `#calibrations` | 2 versi: legacy unverified + approved | **Parsial.** API menyimpan versi; layout kendaraan digunakan ulang. Versi unverified tidak boleh tampil sebagai kalibrasi aktif. |
| `#event-types` | 0 | **Kosong / layout generik.** Tidak termasuk sidebar; belum ada katalog. |
| `#event-severities` | 0 | **Kosong / layout generik.** Tidak termasuk sidebar; belum ada katalog severity. |
| `#geofence-assignments` | 0 | **Kosong / layout generik.** Belum ada hubungan geofence-kendaraan. |
| `#sensor-telemetry` | lebih dari 200 record | **Parsial.** Endpoint sehat; layout sensor dipakai ulang, tidak seluruh detail telemetry ditampilkan dalam kolom khusus. |

## Temuan lintas halaman dan prioritas perbaikan

### P1 — ketepatan data

- [ ] Perbaiki scope tabel tambahan Fuel & BLE per kendaraan; jangan mengambil semua sensor tanpa filter.
- [ ] Bedakan timestamp GPS/pengukuran vs received_at dan kesegaran setiap sumber data; jangan menamai pembacaan lama sebagai live.
- [ ] Pisahkan riwayat titik GPS dari ringkasan trip; implementasikan tombol/proses recalculation yang jelas. Backend recalculation hanya menerima validation_state='validated', sehingga 428 posisi legacy_imported tidak masuk kalkulasi walaupun beberapa valid.
- [ ] Implementasikan pengisian fuel_consumption_summaries dan chart analytics sesuai consumed_l/distance/coverage; jangan menyatakan konsumsi valid dari data kosong.

### P2 — kelengkapan alur

- [ ] Tampilkan liter/persen dan kalibrasi aktif di daftar sensor, bersama raw, gateway, dan kendaraan.
- [ ] Ganti renderer generik untuk Notifications dan Audit dengan kolom yang sesuai.
- [ ] Buat grafik GPS Quality, aktivitas, dan fuel event sesuai domain; tombol % fill saat ini memanggil chart yang sama dan masih memakai fuel_liters.
- [ ] Lengkapi playback lintasan melebihi 50 titik dengan pagination/rentang yang jelas.
- [ ] Sediakan daftar kendaraan di dialog Hubungkan GPS secara terpaginasi bila fleet melebihi 200; saat ini dropdown dibatasi 200.
- [ ] Satukan akses kalibrasi ke dialog Empty/Full; pensiunkan alur manual yang tidak dibutuhkan dari UI pengguna.
- [ ] Uji ekspor laporan live, clipboard, semua filter, peta, dialog, dan responsive di browser.

### P3 — integrasi yang belum tersedia

- [ ] Push notification delivery, jadwal laporan, traffic/satellite, telepon/WhatsApp/BAP, OTA/reset/poll perangkat.
- [ ] Physical ignition tetap NOT_INSTALLED; tidak boleh mengaktifkan status mesin dari asumsi wiring.

**Kesimpulan audit:** data GPS dan fuel utama sudah tersedia, tetapi project belum dapat dinyatakan seluruh halamannya berfungsi lengkap. HTTP endpoint sehat dan tes lulus harus dipisahkan dari kesesuaian seluruh field, chart, filter, dan tombol terhadap kebutuhan pengguna. Audit ini menghasilkan checklist, bukan implementasi perbaikan.
