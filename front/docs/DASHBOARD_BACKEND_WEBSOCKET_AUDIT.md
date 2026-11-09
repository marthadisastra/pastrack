# Audit dashboard dan checklist integrasi Go, database, WebSocket

Tanggal: 7 November 2026, Asia/Jakarta. Status: audit dan rencana implementasi; WebSocket belum diimplementasikan.

Pembaruan tahap 1: kontrak snapshot dashboard versi 1 telah diimplementasikan dan dideploy. Endpoint `GET /api/v1/dashboard/snapshot` membaca kendaraan, ringkasan, GPS, fuel, ignition, sensor, serta alert dari satu transaksi read-only repeatable-read. Dashboard live memakai endpoint ini saat halaman dibuka atau ketika pengguna menekan refresh; belum ada pembaruan otomatis sampai tahap WebSocket selesai.

## Ruang lingkup dan bukti

Audit mencakup DashboardView, tabel kendaraan, peta, grafik, modal inspeksi, useFleet, service HTTP/realtime, notifikasi, API fleet, delta SSE, receiver, schema migrasi dan metadata database live. Tidak ada data produksi yang diubah. Audit ini tidak membuat endpoint atau mengganti deployment aplikasi.

Metadata live diperiksa read-only pada `pastrack_v2_20261103_r2`: seluruh 17 tabel sumber yang diperiksa tersedia. Index histori GPS `(vehicle_id,gps_time,id)`, fuel `(vehicle_id,reading_at,id)`, sensor `(sensor_id,received_at,id)` dan alert `(vehicle_id,state,raised_at,id)` tersedia. Ini membuktikan keberadaan struktur, bukan kelengkapan/akurasi setiap pembacaan sensor. Belum dilakukan rekonsiliasi nilai dashboard terhadap baris produksi, EXPLAIN query, pengujian beban, atau pengujian WebSocket live.

Konfigurasi vhost live mengandung `proxy_http_version` dan `proxy_read_timeout`, tetapi tidak mengandung `proxy_set_header Upgrade` atau `proxy_set_header Connection`. Audit tidak menelusuri seluruh file include Nginx; dukungan upgrade end-to-end harus dibuktikan dengan handshake 101 saat implementasi.

## Verifikasi yang dijalankan

- [x] Baca kontrak Vue/Go, schema dan jalur notifikasi receiver.
- [x] Periksa keberadaan 17 tabel sumber dan index terkait melalui metadata SQL live, read-only.
- [x] Empat tes Playwright demo lulus: dashboard/inspect/filter/export, fokus saat update/offline, fallback gambar peta dan modal inspeksi yang terbuka saat update.
- [x] Tujuh tes utama Go lulus, termasuk subtest role dan state: DashboardStreamCommitAndSessionRevocation, DashboardReadAccessForAllRoles, FleetExclusiveStatesAndFreshness, FleetBufferedFuelIsNotFresh, LiveDeltaOnlyChangedFields, LivePaginationMetadataChangesWithoutChangingRows dan LiveUpstreamRejectsUnknownDestination. Tes integrasi memakai `pastrack_test_local` terisolasi, bukan database produksi.
- [ ] Verifikasi WebSocket 101, replay dan tanpa polling: belum bisa dijalankan karena implementasi WebSocket belum tersedia.
- [x] Verifikasi browser tahap 1 pada live: login session, snapshot SQL, tidak ada request bisnis otomatis selama pengamatan, refresh eksplisit, modal inspeksi, mobile, cookie secure dan logout. Akun fixture sementara dihapus setelah pengujian.

Perintah: `npm test -- --grep 'dashboard inspection|live updates retain|map image failure|realtime changes update'` dari `front`; `go test ./internal/platform -run 'Test.*(Fleet|Dashboard|Live)' -count=1 -v` dengan DSN test terisolasi dan timezone UTC. Hasil UI di atas menguji mode demo; tes stream Go menguji SSE yang ada, bukan bukti bahwa dashboard live sudah terintegrasi.

## Hasil audit

| Prioritas | Temuan terverifikasi dari kode                                                                                                                               | Dampak dan pekerjaan                                                                                                                                     |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P0        | Fleet memakai `VITE_USE_MOCK=true`; `sampleTelemetry()` dipanggil setiap 4 detik                                                                             | Login Go tidak berarti dashboard sudah menggunakan telemetri database. Pisahkan mode dashboard live dari modul lain yang masih demo.                     |
| P0        | `fleetService.listVehicles()` meminta `/vehicles` dan mengharapkan array langsung; Go mengembalikan `{data,...}` dan proyeksi telemetri ada di `/fleet/live` | Jangan hanya mengganti flag mock; buat adapter DTO dan response envelope.                                                                                |
| P0        | Frontend menganggap `id` kode seperti HAULER-04, `imei` penentu monitored; Go memakai ID database dan `device_id`/identifier tersamarkan                     | Gunakan ID database sebagai key, kode kendaraan sebagai label; monitored berdasarkan assignment gateway aktif.                                           |
| P0        | Adapter delta memetakan MOVING/IDLE/PARKED/OFFLINE; Go menghasilkan MOVING/STOPPED/OFFLINE/UNASSIGNED/STALE/GPS_NO_FIX/UNKNOWN_SPEED                         | Kendaraan berhenti bisa tetap tampil moving. Pisahkan movement, connectivity, GPS validity dan alert. Jangan menyimpulkan idle hanya dari kecepatan nol. |
| P0        | Alias delta melewati nilai `null`; entitas baru yang tidak ditemukan di store diabaikan                                                                      | Nilai lama dapat tampak sebagai data terkini; penambahan aset tidak muncul. Terapkan null secara eksplisit dan dukung insert/delete.                     |
| P0        | SQL DECIMAL bisa dikirim sebagai string; tabel/modal memakai `fuel.toFixed()`, recorder memakai `Number.isFinite()`                                          | Normalisasi angka nullable; jangan mengubah data tidak tersedia menjadi nol.                                                                             |
| P0        | Transport produksi `EventSource`/SSE, belum WebSocket                                                                                                        | Bangun endpoint WSS, adapter browser dan proxy upgrade; SSE tidak memenuhi target transport yang diminta.                                                |
| P1        | Grafik menyimpan maksimal 60 sampel sesi; 12H/24H tidak mengubah query histori                                                                               | Grafik tidak mewakili 12/24 jam dan hilang setelah reload. Gunakan histori bertimestamp dari DB, lalu append event.                                      |
| P1        | Peta memakai JPG/SVG ilustrasi dan posisi persentase `x/y`                                                                                                   | Latitude/longitude baru tidak menggerakkan marker ke lokasi geografis sebenarnya. Integrasikan map engine dan proyeksi koordinat.                        |
| P1        | Alert kartu/modals, GNSS lock, 0 uncalibrated, Kalman ON, raw packet footer sebagian hardcoded/sintetis                                                      | Ganti dengan bukti backend atau tampilkan unavailable. Jangan tampilkan diagnosis siphon sebagai fakta untuk setiap state alert.                         |
| P1        | Rata-rata CSQ/HDOP mengganti nilai kosong menjadi nol; kapasitas/fuel tanpa guard null/zero                                                                  | Ringkasan bias dan berpotensi NaN/Infinity. Gunakan denominator pembacaan valid dan tampilkan coverage.                                                  |
| P1        | Hitungan active_alerts dan query kode alert berbeda dalam pengecualian false_positive                                                                        | Samakan definisi alert aktif di kartu, detail, API dan database.                                                                                         |
| P1        | `useFleet` memuat work order maintenance meskipun halaman dashboard tidak membutuhkannya                                                                     | Kegagalan modul lain dapat mencemari status dashboard. Pisahkan bootstrap dashboard dari work order.                                                     |
| P1        | `fleetRows()` meng-query kode alert per kendaraan dan memiliki banyak correlated subquery; snapshot dihitung per subscriber                                  | Risiko query berlipat saat event/pengguna banyak. Ukur EXPLAIN dan rancang projection/event per entitas.                                                 |
| P1        | Notifikasi read/unread disimpan lokal; ekspor memakai seluruh properti objek di browser                                                                      | Persistensi read state perlu API per pengguna; whitelist kolom ekspor agar metadata internal tidak ikut.                                                 |
| P1        | Dashboard tidak merender `error`/skeleton khusus dari useFleet                                                                                               | Kegagalan awal dapat tampak sebagai fleet kosong/nilai nol. Sediakan loading, error, empty dan partial state yang berbeda.                               |

Sumber: `src/views/dashboard/DashboardView.vue`, `src/components/fleet/{FleetMap,FuelChart,VehicleTable,VehicleInspectModal}.vue`, `src/composables/{useFleet,useNotifications,useAuth}.js`, `src/services/{fleet,http,realtime,export}.js`; backend `internal/platform/{api,fleet,live_delta,server}.go`.

## Matriks integrasi setiap bagian dashboard

Semua route API berikut berprefix `/api/v1`. Route bertanda **usulan** belum tersedia.

| Bagian / value                                          | Sumber database                                                                                | API yang bisa dipakai / gap                                                                                                             | Pembaruan event target                                        |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| Identitas aset, plate, model, kode                      | vehicles, vehicle_inventory                                                                    | `/fleet/live`; normalisasi `id`, `vehicle_code`, `plate_number`, `vehicle_name`                                                         | vehicle.upsert / vehicle.remove                               |
| Jumlah monitored, moving, stopped, offline, GPS invalid | vehicle_assignments, gps_device_status, latest_positions, gps_positions                        | `/fleet/overview` sudah ada; definisi monitored dan ringkasan harus konsisten dengan daftar                                             | fleet.summary.updated                                         |
| Total liter, kapasitas, persentase, uncalibrated        | fuel_readings, sensor_telemetry, fuel_sensors, fuel_calibrations, vehicle_inventory            | `/fleet/live` punya sebagian field; agregasi valid/fresh dan coverage perlu diperluas                                                   | fuel.updated / calibration.updated / fleet.summary.updated    |
| Jumlah alert, severity, pesan dan referensi kejadian    | alert_events, fuel_events, device_events, geofence_events                                      | `/fleet/activity`, resource `/alerts`; kontrak detail harus ditetapkan                                                                  | alert.upsert / alert.remove                                   |
| CSQ, GNSS fix, satelit, HDOP                            | network_status_logs, gps_quality_logs, gps_positions; raw hanya untuk diagnosis                | `/fleet/live` menyediakan fix/satelit/HDOP; CSQ belum masuk proyeksi tersebut, cakupan parser harus diperiksa                           | gateway.updated                                               |
| Posisi marker, heading, speed                           | latest_positions, gps_positions, vehicle_assignments                                           | `/fleet/live`; map engine geografis belum terintegrasi                                                                                  | position.updated                                              |
| Geofence, lokasi, alamat                                | geofences, geofence_events, tracking_geocode_cache                                             | resource geofence tersedia; sumber label/alamat dan batas geocoding perlu kontrak khusus                                                | geofence.updated / vehicle.geofence.updated                   |
| Grafik 12H/24H, refuel/drop annotation                  | fuel_readings, sensor_telemetry, fuel_events                                                   | `/fuel/readings?vehicle_id=&from=&to=` ada, dengan pagination; **usulan** `/dashboard/fuel-history` yang bounded/downsampled            | fuel.sample.append / fuel.sample.correct / fuel.event.updated |
| Probe battery, RSSI, temperature, drift                 | sensor_telemetry, ble_sensor_status, calibration metadata                                      | `/sensor-telemetry` tersedia; join proyeksi latest per gateway/binding diperlukan. Drift harus punya perhitungan nyata atau unavailable | sensor.updated                                                |
| ACC/ignition, battery gateway                           | laporan ignition di device_raw_messages, physical_ignition_state; sumber battery harus diaudit | `/fleet/live` menyediakan ignition_status/source/reported_at; jangan samakan event ignition dengan DI1 fisik terpasang                  | ignition.updated / gateway.updated                            |
| Modal inspeksi                                          | gabungan proyeksi di atas berdasarkan vehicle_id                                               | **usulan** DTO inspeksi terotorisasi; ekspansi on-demand bila snapshot belum cukup                                                      | entitas yang sama; modal mengacu ID store, bukan salinan lama |
| Raw packet terakhir                                     | device_raw_messages, protocol_parse_logs                                                       | endpoint diagnosis terotorisasi perlu ditetapkan; tidak merekonstruksi frame palsu dari nilai UI                                        | diagnostics.updated, subscription terbatas                    |
| Bell, unread, read state                                | notifications, alert_inbox_deliveries                                                          | `/notifications` ada; verifikasi kontrak mark-read/read-all yang dipakai Vue                                                            | notification.upsert / notification.read / notification.remove |
| Filter/search/export                                    | DTO dan scope data user                                                                        | Filter lokal jika semua fleet dimuat; pagination/server search jika bounded. Ekspor visible data dibedakan dari export histori server   | patch mempertahankan filter, fokus, sorting dan selected unit |

## Kontrak data yang harus diselesaikan terlebih dahulu

- [x] Snapshot REST memakai ID database stabil; kode aset, identifier gateway, dan device ID merupakan field terpisah. Identifier tetap mengikuti masking backend. WebSocket akan memakai DTO yang sama pada tahap berikutnya.
- [x] Kontrak snapshot memiliki `schemaVersion: 1` dan satu adapter normalisasi; angka nullable diubah sekali saat masuk store.
- [x] Snapshot memisahkan movement, connectivity, GPS, sensor, calibration, ignition dan alert.
- [x] Semua status fleet backend dipertahankan; unknown/null tidak ditampilkan sebagai OFF, 0 liter atau GNSS locked.
- [x] GPS, fuel, gateway, sensor dan ignition membawa timestamp sumber yang tersedia; waktu snapshot tersedia terpisah pada metadata.
- [ ] Bedakan status koneksi socket dari freshness GPS/fuel/gateway. Socket hidup tidak menjamin sensor masih mengirim.
- [ ] Satuan eksplisit: liter, persen, volt vs millivolt, RSSI vs CSQ, Celsius, km/h. Jangan memberi label volt untuk battery_percent.
- [x] Fuel persen berbasis kapasitas valid; total kapasitas dan total liter memakai scope yang sama. Persentase fleet memakai weighted fill atas kapasitas positif yang diketahui.
- [ ] Ignition fisik NOT_INSTALLED tetap terlihat; reported ignition ON/OFF disertai sumber dan timestamp, bukan dianggap kabel DI1 tersambung.
- [ ] Ringkasan dan daftar dihitung terhadap scope/snapshot yang sama; jelaskan apakah filter tabel juga memfilter KPI.
- [ ] Error, loading awal, empty valid, nilai tidak tersedia, stale, disconnected, reconnecting dan resync mempunyai tampilan tersendiri.

## Arsitektur target tanpa polling interval

```text
Gateway / sensor
  → receiver Go → transaksi fakta GPS/fuel + latest projection + event/outbox
  → COMMIT sukses
  → wake-up andal / CDC binlog
  → projector / alert engine berbasis event
  → hub WebSocket Go terautentikasi
  → WSS /api/v1/ws
  → Vue normalized store
  → KPI + peta + tabel + grafik + modal + notifikasi
```

REST tetap untuk login, histori, perintah, ekspor, dan pembacaan eksplisit bila dibutuhkan. Tidak ada GET terjadwal untuk menyegarkan value. **Usulan bootstrap utama:** setelah subscribe WS berhasil, server mengirim snapshot lengkap dengan cursor/barrier, kemudian delta berurutan. Alternatif REST snapshot harus memakai barrier + buffer event yang terdokumentasi agar tidak ada perubahan hilang di antara fetch dan subscribe.

Istilah “tanpa interval” diterapkan pada pengambilan/pemrosesan data bisnis: tidak ada polling HTTP, polling query database, refresh dashboard berkala, atau fallback polling. Ping/pong untuk kesehatan socket, retry one-shot setelah koneksi gagal dan timer satu kali pada deadline freshness bukan polling value. Jam tampilan lokal/animasi juga tidak mengambil data; boleh dipertahankan terpisah bila diperlukan.

### Checklist WebSocket dan hub Go — P0

- [ ] Tambahkan **usulan** `GET /api/v1/ws` dengan handshake 101 dan WSS live; jangan menyebut `/realtime` SSE sebagai WebSocket.
- [ ] Autentikasi memakai session cookie yang sudah ada; validasi Origin exact allowlist, role, scope kendaraan dan topic sebelum upgrade/subscribe.
- [ ] Tangani timeout middleware HTTP 10 detik: server saat ini mengecualikan hanya route SSE. Route WS perlu lifecycle context, deadline read/write dan shutdown sendiri.
- [ ] Satu koneksi per workspace; subscription fleet, unit grafik terpilih dan notifikasi user. Unsubscribe ketika selection berubah atau workspace ditutup.
- [ ] Event envelope mencakup `schema_version`, `event_id`, `cursor`, `type`, `entity_id`, `entity_version`, `occurred_at`, `committed_at`, `payload`.
- [ ] Snapshot diambil setelah subscription/barrier aktif; buffer delta selama snapshot. Jangan membaca summary dan rows berbeda tanpa konsistensi yang ditetapkan.
- [ ] Dukungan upsert lengkap, partial patch, null eksplisit dan tombstone; event duplikat tidak menggandakan sampel/notifikasi.
- [ ] Tolak patch versi lebih lama. Bedakan histori terlambat yang valid dari perubahan latest yang tidak boleh mundur.
- [ ] Cursor memakai urutan stream ter-commit yang dapat di-replay; jangan menganggap AUTO_INCREMENT outbox = urutan commit transaksi.
- [ ] Reconnect exponential backoff dengan jitter; resume dari cursor terakhir. Cursor di luar retention → `resync_required` → snapshot baru.
- [ ] Buffer dan frame dibatasi; coalesce update latest per kendaraan jika aman, tetapi jangan membuang histori fuel/alert tanpa replay/resync.
- [ ] Ping/pong dan write deadline menutup koneksi mati; klien lambat diputus/resync agar tidak menahan semua subscriber.
- [ ] Logout/reset password/disable user/perubahan role menutup socket terkait; expiry session memakai timer deadline. Tidak bergantung polling `/auth/me` per menit.
- [ ] HTTP request gagal atau event schema rusak menampilkan status yang jelas dan mempertahankan nilai terakhir sebagai stale; tidak melabelinya live.
- [ ] Jangan memasukkan password, session token, IMEI penuh atau raw frame ke URL subscription/log tanpa kebijakan akses yang eksplisit.

### Checklist receiver, database dan keandalan event — P0/P1

- [ ] Pertahankan assignment/binding sesuai waktu kejadian; sensor tidak boleh menempel ke kendaraan lama setelah gateway dipindah.
- [x] Endpoint mutasi platform menerbitkan notifikasi setelah commit; rollback tidak menghasilkan delta. Assignment, inventory/capacity, kalibrasi, alert dan user sudah memanggil publisher setelah mutasi berhasil.
- [x] Jalur telemetry receiver sekarang menulis `dashboard_change_outbox` dalam transaksi normalisasi yang sama; channel hanya menjadi wake-up dan bukan sumber durabilitas.
- [x] Publisher menguras outbox saat startup dan ketika menerima wake-up, memberi ACK melalui `delivered_at`, retry dengan timer one-shot, serta tidak melakukan polling database periodik.
- [x] Celah commit→publish diuji lokal dan live: event di-commit tanpa sinyal, receiver direstart, startup drain mengirim invalidation dan UI WSS memperbarui projection tanpa HTTP polling.
- [x] Alert worker tidak lagi memakai ticker 1 detik. Startup, telemetry dan mutasi konfigurasi memberi wake-up; retry/lease recovery memakai timer one-shot dari deadline `available_at`/`locked_at`, dengan backoff satu detik hanya saat error.
- [x] WebSocket dashboard memakai deadline one-shot terdekat dari gateway, GPS, fuel dan BLE; timer dijadwalkan ulang pada setiap projection baru tanpa polling. Live QA membuktikan `STOPPED/ONLINE` berubah menjadi `OFFLINE` tanpa paket atau sinyal baru.
- [ ] Tetapkan versi projection per entitas dan cursor stream; perbarui latest dan event secara konsisten. Perubahan lewat jalur SQL lain juga harus masuk publisher/CDC.
- [ ] Aturan late/buffered/duplicate packet: simpan histori bila valid, tetapi jangan mengganti latest dengan kejadian yang lebih tua.
- [ ] Konsisten mengecualikan resolved/closed/false_positive sesuai kebijakan alert aktif yang disepakati.
- [ ] Profilkan query; hilangkan query kode alert per kendaraan bila terbukti bottleneck. Hindari menjalankan snapshot seluruh fleet untuk setiap pengguna pada setiap paket.
- [ ] EXPLAIN join assignment aktif, latest fuel dan histori; evaluasi index `(vehicle_id,starts_at,ends_at)` dan index latest per gateway/sensor berdasarkan plan nyata, bukan menambah index tanpa pengukuran.
- [x] Tentukan latest reading vs latest valid reading, coverage, usia dan calibration revision. Sampel invalid/stale tidak dimasukkan sebagai fuel valid; timestamp SQL `time.Time` dinormalisasi sebelum freshness dihitung.
- [ ] Histori 12/24 jam memakai UTC di DB, display Asia/Jakarta, sorting event-time, pagination/downsampling bounded; gap tetap gap, bukan digambar seolah data kontinu.
- [ ] Migrasi additive hanya jika projection/outbox memang memerlukannya; dokumentasikan retensi, recovery, backfill dan rollback. Tabel fakta yang ada dipakai ulang.

### Checklist Vue/dashboard — P0/P1

- [x] Pisahkan mode dashboard dari modul maintenance/demo; build live memakai endpoint same-origin `/api/v1`.
- [x] Ganti transport dashboard live dengan WebSocket; tidak ada polling fallback.
- [x] Workspace dashboard memakai event pencabutan session dan deadline expiry; reconnect selalu bootstrap snapshot baru.
- [x] Bootstrap snapshot ter-normalisasi dan tidak bergantung pada array demo `/vehicles`.
- [x] Store keyed by database vehicle ID, merge in-place, serta menangani upsert dan removal.
- [x] Ringkasan dashboard berasal dari projection backend yang sama dengan daftar kendaraan.
- [x] Peta Leaflet memakai lat/lon dan fix valid dari backend; loading/error tile terpisah dari WSS, koordinat nol/no-fix tidak menjadi marker, dan marker keyed by vehicle ID diperbarui tanpa membuat ulang peta.
- [x] Grafik meminta histori hanya saat kendaraan/periode 12H/24H berubah, membatalkan request lama, dan memakai waktu kejadian aktual pada sumbu X.
- [x] Histori server dibatasi 5.000 baris, SVG di-downsample bounded, gap invalid dipertahankan, dan sampel fuel terbaru di-merge dari WSS berdasarkan `vehicleId` + `sampleId`. Delta membawa waktu pengukuran/penerimaan, validitas dan calibration ID; duplikat tidak menambah titik kedua dan urutan grafik tetap event-time.
- [ ] Badge Kalman/filter hanya aktif jika backend mengirim provenance/algorithm; kalibrasi unknown bukan “Zero-point OK”.
- [x] Null/zero capacity aman pada tabel, kartu dan modal; kualitas invalid tetap diberi label.
- [x] Partial failure mempertahankan nilai terakhir; tombol resync adalah rekonsiliasi eksplisit, bukan syarat realtime.
- [x] Bell mengambil data per user; mark-read tersimpan server dan disebarkan ke tab lain melalui delta WebSocket. Mode live tidak memakai seed notifikasi demo.
- [x] Ekspor snapshot dashboard memakai whitelist kolom eksplisit; histori tetap dipisahkan dari ekspor snapshot.

### Checklist proxy, observability dan deployment — P0

- [x] Nginx live memiliki location WS, HTTP/1.1, header Upgrade/Connection, timeout 70 detik dan buffering off; `nginx -t` lulus sebelum reload.
- [x] Vite dev proxy memakai `ws: true` serta menerjemahkan Origin hanya untuk request same-origin development.
- [x] WSS domain live diuji lewat Chromium dengan cookie Secure/HttpOnly; snapshot polling tidak terjadi.
- [ ] Metrics lengkap belum selesai; endpoint `/healthz` kini mengekspos connected clients, handshakes, auth failures dan messages. Latency commit→render, queued bytes, replay dan publisher lag masih perlu instrumentasi.
- [ ] Target awal yang harus diukur/disepakati: p95 commit→render ≤1 detik pada beban representatif; ini target acceptance, bukan hasil pengujian audit.
- [ ] Bila beberapa proses web, gunakan bus lintas proses; hub in-memory dan Unix socket satu proses tidak otomatis menjangkau seluruh replica.
- [x] Build/test lokal, backup binary/frontend/config, deploy backend+proxy, lalu smoke test WSS/session dan frontend live.
- [x] Endpoint dan frontend sudah dideploy ke live; health backend dan proses receiver diperiksa, dengan backup rollback binary/frontend/Nginx di release stage.

## Checklist acceptance sebelum menandai dashboard terintegrasi

- [x] Network browser membuka `/api/v1/ws`; tidak ada GET snapshot dashboard berulang dan satu koneksi dipertahankan selama pengamatan.
- [x] Commit perubahan nama fixture menghasilkan delta dan memperbarui tabel live tanpa refresh; removal menghapus baris yang sama.
- [x] Tidak ada commit/perubahan/deadline bisnis selama periode pengamatan → tidak ada request snapshot/polling bisnis; heartbeat tetap diizinkan.
- [x] Rollback transaksi tidak memengaruhi UI; event telemetry ter-commit yang belum dipublish dipulihkan melalui outbox saat receiver startup.
- [ ] Snapshot dan delta tidak punya race: perubahan yang terjadi selama bootstrap tidak hilang/diterapkan dua kali.
- [ ] Paket out-of-order future masih memicu resync; duplicate cursor lama kini diabaikan idempotently. Null clearing, entitas baru, removal dan assignment swap perlu test paket lengkap.
- [ ] Nol, null, no-fix, offline, unassigned, uncalibrated dan false_positive memiliki hasil KPI/label yang dapat direkonsiliasi terhadap SQL fixture.
- [x] 12H dan 24H meminta rentang histori database berbeda; perubahan kendaraan membatalkan response lama dan mengganti seri, bukan membawa cache kendaraan sebelumnya.
- [x] Timer freshness mengubah status tanpa paket baru dan tanpa query polling; transition live `STOPPED/ONLINE → OFFLINE` telah diverifikasi melalui WSS.
- [ ] Putus jaringan, tab background, resume, restart server dan cursor expired kembali konsisten melalui replay atau snapshot.
- [ ] Logout/password reset/revoke role memutus socket sudah diuji; pembatasan scope telemetri per user belum diimplementasikan karena projection saat ini masih `all_active_vehicles`.
- [ ] Klien lambat diputus oleh write deadline 5 detik dan wake-up dikoaleskan ke buffer satu elemen; durable replay alarm/fuel history belum tersedia.
- [x] Loading, empty, error, map tile failure dan partial API failure diuji; dashboard tetap compact pada mobile/laptop.
- [ ] Nilai fixture dibandingkan langsung dengan sumber SQL dan dicatat sebagai bukti; mock UI tests tidak dipakai sebagai bukti integrasi produksi.

## Urutan pengerjaan yang disarankan

1. **Kontrak dan snapshot:** normalisasi ID/envelope/null/enum, definisi KPI dan scope, bootstrap yang konsisten.
2. **Event Go + WebSocket:** after-commit, reliability/replay, auth lifecycle, freshness deadline dan worker alert event-driven.
3. **Integrasi komponen:** tabel/KPI/modal, peta geografis, histori fuel, notifikasi server, ekspor.
4. **QA + rollout live:** acceptance di atas, WSS/proxy, pengukuran performa, deployment dan rollback.

## Referensi transport

[NGINX WebSocket proxying](https://nginx.org/en/docs/http/websocket.html) menjelaskan penerusan upgrade dan pengelolaan koneksi idle. [MDN WebSocket API](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API) menjelaskan keterbatasan backpressure API browser; karena itu buffer dan kebijakan klien lambat perlu ditangani oleh aplikasi.
