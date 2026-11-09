# Audit lanjutan dan target architecture

## Status bukti Phase 1

Pemeriksaan sebelumnya baru desain, bukan audit penuh. Pemeriksaan lanjutan mengonfirmasi dua modul Go: listener memakai go.mod 1.24.0, web Beego v2 dengan go.mod 1.27.3; frontend HTML template/vanilla JS/CSS/Leaflet, bukan React. Versi directive tidak membuktikan toolchain terpasang. Driver SQL MySQL; versi server aktual belum diketahui. DDL inti vehicles/gps_devices/gps_positions/raw/events tidak ditemukan dalam migrasi repo, sehingga perbandingan fisik penuh harus menunggu snapshot lokal SHOW CREATE TABLE. Tidak memakai audit November 2 sebagai fakta live terkini.

Listener durable spool dan retry sekarang at-least-once; commit lalu crash sebelum spool delete dapat menduplikasi fakta. save() mencampur raw insert, parsing, sensor discovery dan event insert. Hanya GTSTT layout tertentu menghasilkan posisi; raw disimpan untuk jenis lain. Payload GTERI CFS300B diekstrak memakai marker dan kandidat raw eksperimental. Status GTSTT disimpan sebagai event tanpa semantic verification. Latest vehicle data masih menggunakan subquery histori dan current device.vehicle_id, berisiko histori berpindah saat reassign. Fuel calibration lama memakai persen/media; ini bukan bukti volume. Unknown/error tidak boleh hilang saat normalisasi.

Web memiliki auth/session/roles sederhana, master soft delete/audit, geocoding private, photos, exports dan notifications/push; jangan hilangkan saat rebuild. SSE + Unix socket invalidation sudah tersedia. Frontend lama memakai Gentelella dan hash navigation; tetap vanilla karena tidak ada kendala teknis yang membenarkan framework baru. Kode menyertakan file konfigurasi service account; isinya tidak dibaca atau disalin dalam audit ini. Pengemasan nantinya harus menjaga secret di luar aset publik.

## Target modul

- transport: bounded TCP framing, durable receipt spool, health; tidak mengubah konfigurasi perangkat nyata.
- protocol: classifier + registry model/version, layout/mask cursor, typed nullable fields, validation errors, decoder evidence/version.
- ingestion: raw append-only, idempotent derived writes, assignment resolution by event time, transaction boundary.
- fleet, telemetry, trips, sensors, fuel, geofences, alerts, identity, reports: domain service/repository terpisah.
- query: latest state untuk live, cursor/range history untuk analytics; jangan menghitung live dengan scan histori.
- delivery: REST /api/v1, compatibility adapters, existing SSE invalidation, background workers dengan bounded batches dan structured redacted logging.

Raw disimpan meski decode gagal. Transaksi derived memuat idempotency key, fakta, latest state dan event. Worker hanya menghapus spool setelah raw receipt durable; parser failure berstatus unsupported/failed yang dapat direplay, DB failure retry dengan backoff. Semua parser diproteksi dari panic per frame, tanpa membunuh receiver. Immutability raw tidak menghalangi parse versi baru pada tabel log.

## Kontrak API umum

GET list: {data:[],page:{next_cursor:null,has_more:false},meta:{as_of:"UTC",timezone:"Asia/Jakarta"}}. Detail: {data:{...},meta:{...}}. Error: {error:{code,message,request_id,fields?}}. Limit default 50 maksimum 200, sort allowlist, from/to RFC3339 UTC dengan rentang maksimum per endpoint; tidak me-load histori tak terbatas. Chart aggregation downsample server-side dengan coverage dan units. NULL tetap NULL, bukan 0. Latency/stale didasarkan pada sumber timestamp masing-masing.

Mutasi membutuhkan auth/permission, validation, audit dan optimistic row_version; transaction memastikan assignment tidak overlap. Key request idempoten untuk command/ack/create yang dapat diretry. SSE tidak membocorkan kendaraan tanpa izin dan hanya mengirim invalidation/summary terbatas; historical query tetap terpisah. Dashboard menyatakan physical ignition NOT_INSTALLED dan tidak menghasilkan engine state dari kecepatan.

## Tahapan setelah review rancangan

1. Lengkapi inventory fisik dari snapshot lokal dan redacted raw fixture untuk mask/version aktual. Finalisasi DDL beserta constraint terhadap engine lokal.
2. Buat schema target terisolasi, test FK/index/check, backfill ledger, rehearsal migrasi/rollback tanpa menyentuh sumber.
3. Registry parser + fixtures terverifikasi → ingestion idempoten → latest-state → API parity/auth. Fuel liter tetap unavailable bila bukti unit/kalibrasi belum ada.
4. Implementasi shell + seluruh 21 layar satu per satu dengan data API; filter/pagination/map/chart/permission diuji, tanpa placeholder yang diklaim selesai.
5. End-to-end TCP→raw→parse→SQL→API→SSE/UI, malformed/unknown/no-fix/late/duplicate fixtures, load/history EXPLAIN dan visual desktop/mobile.
6. Laporan file/tests/limitations. Jalankan lokal dahulu. Tidak deploy sampai perintah pengguna; produksi juga memerlukan backup dan persetujuan migrasi eksplisit.
