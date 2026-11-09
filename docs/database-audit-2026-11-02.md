# Audit database produksi — 2 November 2026

Database: `pastrack` pada `76.13.195.48`. Cadangan sebelum pembersihan: `/opt/pastrack/backups/pre-cleanup-2026-11-02.sql.gz` (milik root, mode privat, uji `gzip -t` berhasil).

## Skema dan relasi

15 tabel diperiksa. Relasi inti tracking lengkap: `vehicles` → `gps_devices` → `gps_positions`, `device_raw_messages`, `device_events`, dan `device_fuel_sensors`. Pesan raw terhubung ke posisi/event/sensor melalui `raw_message_id`. Semua foreign key inti ada dan pemeriksaan tidak menemukan posisi yatim, pesan raw dengan IMEI yang berbeda dari device terkait, atau sensor dengan raw message dari device lain. Satu GPS dapat memiliki banyak sensor CFS300B melalui keunikan `(device_id,sensor_id)`.

Tabel aplikasi web (pengguna, sesi, notifikasi, push, geocoding, lokasi tracking, dan audit) juga diperiksa. Tidak ada push job tanpa notifikasi maupun token push tanpa sesi. `web_push_jobs.notification_id` dan `web_push_tokens.session_hash` tidak memiliki foreign key; saat audit tidak ada baris yatim. Keduanya tidak mempengaruhi relasi tracking.

## Pembersihan

Atas permintaan pengguna, enam perangkat `SIMULATOR` dan enam kendaraan simulasi dihapus permanen bersama 29.817 posisi, 29.817 pesan raw, dan 2.590 event terkait. Pembersihan dibatasi oleh `model='SIMULATOR'` dan pola IMEI simulasi. Sebanyak 54 posisi duplikat dari GV30CEU nyata dihapus dengan mempertahankan titik paling awal; pesan raw asal tetap disimpan. Dua puluh tujuh pesan `GTSTT` GV30CEU versi `80201E0306` tanpa fix GPS diubah dari `failed` menjadi `partial`. Parser backend kini menerima pesan tersebut sebagai laporan valid tanpa koordinat.

Setelah pembersihan tersisa satu kendaraan nyata (`B 1307 KNQ`), dua GV30CEU, dan satu CFS300B yang teramati. IMEI `861252080093813` memiliki riwayat posisi; IMEI `861252080093110` aktif mengirim laporan CFS300B tetapi belum menghasilkan fix GPS. Ringkasan web menggabungkan posisi terakhir kendaraan, kecepatan saat fix itu, dan sensor dari semua GPS yang terhubung. Pada verifikasi, posisi terakhir bertanggal 28 Oktober 2026 dan harus diperlakukan sebagai GPS lama.

Sebanyak 41 pesan raw tanpa IMEI dipertahankan sebagai bukti diagnostik; isinya termasuk ACK dan trafik biner yang tidak cocok dengan parser. Sebanyak 24 posisi lama tanpa `raw_message_id` juga dipertahankan karena identitas sumbernya tidak dapat dibuktikan. Volume BBM dalam liter belum tersedia: arti field sensor dan kalibrasi tangki belum diverifikasi.

## Verifikasi

`go test ./...` lulus pada backend dan web; `node --check` lulus untuk UI. Kedua layanan produksi melaporkan HTTP 200 pada `/healthz`. Query ringkasan kendaraan menghasilkan satu baris dengan dua GPS, satu sensor `E04E7A648774`, dan koordinat/kecepatan terakhir dari IMEI lama.

## Pembaruan setelah audit awal

Atas instruksi berikutnya, IMEI lama `861252080093813` dihapus permanen bersama 49 posisi, 124 pesan raw, dan 79 event. Cadangan baru dibuat sebelum penghapusan: `/opt/pastrack/backups/pre-old-imei-removal-2026-11-02.sql.gz` (uji `gzip -t` berhasil). Kendaraan `B 1307 KNQ` tetap ada dan kini hanya terhubung ke IMEI terbaru `861252080093110` serta sensor `E04E7A648774`. Posisi dan kecepatan GPS dari perangkat lama tidak lagi muncul di ringkasan kendaraan. Hingga perangkat baru mengirim fix GPS valid, status lokasi harus ditampilkan sebagai belum tersedia.
