# Paket review rancangan PasTrack

Tanggal: 3 November 2026 (Asia/Jakarta). Status: **rancangan awal; implementasi lokal parsial kini tersedia**, lihat [progress](../REBUILD_PROGRESS.md).

| Permintaan | Dokumen |
|---|---|
| 1. ERD lengkap | [ERD.mmd](ERD.mmd) — 65 tabel beserta kolom dan FK |
| 2. Perbandingan lama/baru | [DATABASE_MIGRATION.md](DATABASE_MIGRATION.md) |
| 3. Mapping tabel | [DATABASE_MIGRATION.md](DATABASE_MIGRATION.md) |
| 4. Dibuat/diubah/dihentikan | [DATABASE_MIGRATION.md](DATABASE_MIGRATION.md), [kamus data](DATABASE_SCHEMA.md) |
| 5. Queclink GV30CEU | [PROTOCOL_MAPPING.md](PROTOCOL_MAPPING.md) |
| 6. CFS300B BLE → sensor_telemetry/fuel_readings | [PROTOCOL_MAPPING.md](PROTOCOL_MAPPING.md) |
| 7. Seluruh layar yang teridentifikasi | [UI_PAGE_MAPPING.md](UI_PAGE_MAPPING.md) — 21 ekspor |
| 8. Layar → route/API/database/permission | [UI_PAGE_MAPPING.md](UI_PAGE_MAPPING.md) |
| 9. Migrasi/rollback | [DATABASE_MIGRATION.md](DATABASE_MIGRATION.md) |

[Audit dan target architecture](ARCHITECTURE.md) menjelaskan bukti, gap dan tahapan implementasi. Physical ignition instalasi saat ini NOT_INSTALLED. Tidak ada DROP, TRUNCATE, perubahan schema sumber, koneksi produksi atau deployment dalam pekerjaan ini.

Batas review: tidak tersedia dump/DDL inti terkini atau frame lengkap aktual yang tersanitasi pada paket audit ini. Definisi tabel inti lama diturunkan dari penggunaan kode, bukan introspeksi live. Kontrak route/API adalah target, belum endpoint yang berfungsi. Satuan CFS300B belum disimpulkan walaupun struktur byte ditemukan di manual resmi. Tidak menganggap rancangan ini sebagai audit Phase 1 lengkap atau implementasi selesai.

Gate dari permintaan pengguna: “Sebelum melakukan implementasi, saya ingin melihat” dan “Setelah seluruh rancangan diperiksa”. Paket ini diserahkan dahulu untuk pemeriksaan. Setelah review, pekerjaan berlanjut ke DDL dan implementasi lokal bertahap sesuai ARCHITECTURE.md; ini bukan penghentian tujuan rebuild pada dokumentasi.

Master prompt terbaru mengizinkan implementasi rutin tanpa review tambahan per fase. Tidak ada izin deployment produksi. Dokumen review ini adalah baseline, bukan klaim seluruh kontrak sudah diterapkan.
