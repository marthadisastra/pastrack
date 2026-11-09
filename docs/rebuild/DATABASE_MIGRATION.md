# Perbandingan, mapping, migrasi dan rollback

Status: rancangan untuk pemeriksaan; belum ada SQL migrasi dieksekusi. Database live belum diperiksa ulang. Inventaris lama bersumber dari query aktual, migrasi lokal dan audit terdahulu, bukan klaim SHOW CREATE TABLE terkini.

## Perbandingan dan mapping seluruh tabel lama yang teridentifikasi

| Lama | Target | Perubahan dan mapping |
|---|---|---|
| vehicles | vehicles, vehicle_types, vehicle_groups, vehicle_group_members | Pertahankan PK/plate/name/fleet_code, metadata dan soft delete. Type/group hanya dari nilai terkonfirmasi. physical_ignition_state=NOT_INSTALLED untuk instalasi saat ini. |
| vehicle_assets | vehicle_assets, fuel_tank_profiles | Seluruh metadata/foto dipertahankan; tank_capacity_l menjadi capacity_l pada profil versi awal, tidak menghasilkan fuel readings. Foto privat disalin dengan checksum. |
| gps_devices | gps_devices, gps_device_models, vehicle_assignments, gps_device_status | IMEI/model/firmware/protocol dipertahankan; vehicle_id menjadi association saat snapshot. Waktu awal histori pemasangan tidak diketahui, jangan diberi tanggal palsu. last_seen_at menjadi last_received_at, bukan GPS time. |
| gps_positions | gps_positions, latest_positions | Pertahankan id/device/raw/gps_time/device_send_time/koordinat/speed/heading/altitude; created_at hanya dipakai received_at jika semantiknya terbukti. gps_accuracy tetap raw, bukan HDOP. satellites dipindah hanya bila terisi dan terverifikasi. Vehicle/assignment histori dibiarkan NULL bila tidak terbukti. |
| device_raw_messages | device_raw_messages, protocol_parse_logs, protocol_parser_errors | raw_payload byte-identik, received_at dan protocol_version dipertahankan; parsed_payload/parse_status/parse_error menjadi hasil parse legacy. Hash dihitung dari byte asli. legacy_id menjaga relasi dan jejak sumber. |
| device_events | device_events | event_code/value/message_type dipertahankan sebagai raw; event_time tidak diganti receive time. Tidak mengartikan status belum dikenal sebagai alarm/ignition. |
| device_fuel_sensors | ble_sensors, ble_sensor_bindings, ble_sensor_status, fuel_sensors | sensor_id menjadi sensor_identifier; MAC hanya jika semantik terverifikasi. device_id adalah binding administratif, reported_device_id observasi berbeda. last_seen_at dipertahankan. observed_raw_value bersifat eksperimen dan bukan liter; telemetry dibuat hanya jika raw asal valid tersedia. |
| fuel_calibration_points | fuel_calibrations, fuel_calibration_points | Kelompokkan per sensor/media; raw_value menjadi input_value, level_percent menjadi legacy_percent/output_value unit percent, approval_state=unverified. Jangan ubah persen tinggi air menjadi persen volume atau liter. |
| web_users | users, roles, user_roles | Pertahankan hash/algo/auth_version/soft delete/full profile. Role admin/operator/viewer menjadi role eksplisit; mapping privilege diuji. Tidak reset password otomatis. |
| web_sessions | sessions | token_hash/user/auth_version/expiry dipertahankan hanya bila format dan secret kompatibel; jika tidak, invalidasi sesi saat cutover terencana. |
| web_notifications | notifications | Pertahankan id/user/title/body/target/read_at/created_at; route target lama mendapat alias. |
| web_push_tokens | push_tokens | Hubungkan hash sesi dengan session target; token plaintext dipindah ke storage terenkripsi, tanpa output/log. Token yatim dikarantina. |
| web_push_jobs | push_jobs | Mapping notification_id/token_hash melalui ledger; pertahankan attempts/state/next_attempt. Tahan worker selama cutover agar tidak kirim dua kali. |
| pastrack_audit_log | audit_log | Seluruh actor/action/entity/request/before/after/time dipertahankan; actor_id NULL bila tidak dapat dipetakan, actor_legacy disimpan. |
| tracking_places | tracking_places | Pertahankan tempat/radius/PK/versi; jangan otomatis menjadikan tempat sebagai geofence aktif. |
| tracking_geocode_cache | tracking_geocode_cache | Pertahankan cache dan expiry; tidak kirim ulang lokasi ke provider saat backfill. |
| tracking_geocode_usage | tracking_geocode_usage | Pertahankan usage_day/requests agar limit provider tetap akurat. |

## Daftar tindakan tabel

- **Dibuat:** seluruh 65 tabel pada DATABASE_SCHEMA.md di schema lokal terpisah `pastrack_v2_local`, termasuk nama yang sama dengan tabel lama. Ini copy/mapping, bukan ALTER in-place.
- **Diubah secara logis pada target:** vehicles, gps_devices, gps_positions, device_raw_messages, device_events, vehicle_assets, fuel_calibration_points, tracking_places/cache/usage sesuai tabel di atas. Struktur sumber tetap utuh.
- **Dihentikan dari jalur aplikasi setelah cutover disetujui:** device_fuel_sensors, web_users, web_sessions, web_notifications, web_push_tokens, web_push_jobs, pastrack_audit_log; fungsi digantikan target. Tabel sumber tidak dihapus.
- **Tidak dibuat:** organizations sampai kebutuhan multitenancy terbukti; ble_telemetry karena sensor_telemetry kanonis; tabel agregasi per halaman yang hanya menduplikasi fakta.
- **Tidak dihentikan:** spool TCP, tracker configuration, authentication, SSE, foto privat, geocoding, inbox/push, audit dan ekspor laporan. Semua harus mendapat pengujian kompatibilitas.

## Runbook migrasi lokal

1. Verifikasi engine/version lewat koneksi lokal read-only, inventaris SHOW CREATE TABLE/index/FK/count/min/max dan timezone. Jangan menganggap server adalah MySQL 8: migrasi lama memakai sintaks yang mungkin MariaDB. Uji DDL terhadap versi sebenarnya.
2. Buat backup konsisten snapshot sumber: dump transactional untuk InnoDB, routines/triggers/events beserta manifest versi dan checksum; simpan privat. Untuk tabel nontransactional, gunakan snapshot/lock terencana. Backup privat/spool/foto/config disimpan terpisah. Restore ke database lokal terisolasi, uji restore, baru lanjut.
3. Buat target kosong melalui migrasi berversi manual; tidak menjalankan migrasi saat aplikasi startup. Tambah tabel migration_batches/record_map dengan watermark per tabel dan sumber snapshot.
4. Backfill lookup/auth/master; raw receipts; assignment terkonfirmasi; posisi/event; BLE/fuel/calibration; notifications/push/audit/cache. Insert batch terbatas, keyset berdasarkan PK, checksum per row. Tangani siklus FK lewat insert tahap pertama nullable lalu resolve; raw.device_id dan device_status.last_raw tidak mengharuskan siklus insert.
5. Isi record_map untuk setiap pemetaan satu-ke-banyak. Restart batch memakai key unik ledger; tidak menimpa manual edits. Data rusak tetap di sumber dan ledger status quarantined; laporkan, jangan hilangkan diam-diam.
6. Historis vehicle assignment tidak boleh direkonstruksi dari current device.vehicle_id saja. Ambil bukti assignment; bila tidak ada simpan vehicle_id=NULL pada histori dan tampilkan unresolved. Histori legacy tanpa raw tetap dipertahankan dengan source=legacy_unlinked, bukan dibuatkan raw palsu.
7. Jalankan replay raw pada target dalam mode no-side-effects. Identifikasi decoder/version dan bandingkan hasil dengan legacy. ACK/outbound commands, notification delivery dan tracker config dinonaktifkan pada replay.
8. Jalankan shadow-read lokal terhadap API lama/baru. Validasi count total = migrated + quarantined, FK, checksum raw, timestamps, koordinat, device association, role permissions, fuel NULL, physical ignition, out-of-order latest, replay idempotency.
9. Cutover **lokal** dengan writer berhenti singkat: drain spool, catat watermark akhir, migrasikan delta, verifikasi, arahkan konfigurasi lokal eksplisit ke target. Sumber tetap read-only dan utuh. Tidak mengubah environment produksi.
10. Jalankan receiver/web lokal tanpa membuka listener publik. Uji seluruh pipeline menggunakan fixture tersanitasi, bukan mengirim command ke tracker nyata. Produksi memerlukan instruksi deploy tersendiri dan otorisasi migrasi eksplisit.

## Rollback yang menjaga data baru

- Sebelum cutover: kembalikan binary/config lokal lama; target dibiarkan utuh untuk diagnosis. Tidak perlu down migration yang menghapus tabel.
- Setelah cutover: hentikan writer/worker baru, bekukan target, backup delta dan spool dengan watermark. Jangan langsung switch jika target sudah menerima data baru.
- Replay delta raw ke ingestion lama melalui adapter idempoten dan ledger; rekonsiliasi master edits, akun, acknowledgement dan notification state melalui mapping. Target-only entity yang belum kompatibel tetap diarsipkan; laporkan sebelum rollback dinyatakan lengkap.
- Jangan mengaktifkan pengiriman push/command selama replay. Validasi checksum/count/last receipt dan latest state, lalu aktifkan writer lama dan lanjutkan spool dari watermark. Jika reverse mapping belum tersedia, rollback diblokir dan gunakan roll-forward; jangan mengorbankan data.
- Restore backup hanya ke schema terpisah untuk pemulihan/komparasi, tidak menimpa database aktif. Simpan kedua schema sampai masa retensi yang disetujui.

## Gate penerimaan

Rancangan diperiksa pengguna → DDL target dan pengujian lokal → backfill rehearsal + rollback rehearsal → backend bertahap → UI per halaman → integrasi/QA. Review ini bukan izin migrasi produksi, DROP, TRUNCATE, atau deploy. Durasi retensi, kapasitas disk dan SLA cutover harus ditetapkan dari pengukuran snapshot lokal.
