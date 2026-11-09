# Pemeriksaan live — 5 November 2026

## Cakupan pemeriksaan

Pemeriksaan pembukaan halaman, pemuatan data API, error JavaScript, dan pembukaan Detail jika tersedia. Hasil ini bukan verifikasi seluruh operasi perubahan data atau kondisi fisik perangkat.

| Halaman | Status baca/tampilan | Data teramati |
|---|---|---|
| Fleet Overview | Lulus | 2 |
| Fleet Overview | Lulus | 2 |
| Live Tracking | Lulus | 2 |
| Trip History | Lulus | 1 |
| Geofences | Lulus | 0 |
| Fuel Monitoring | Lulus | 50 |
| Fuel Analytics | Lulus | 0 |
| Fuel Events | Lulus | Sinkronisasi: 5 Nov 2026, 11.28.04 WIB · 1015 pembacaan · 580 titik GPS |
| GPS Quality | Lulus | 50 |
| Device Connectivity | Lulus | 50 |
| Data Activity | Lulus | 50 |
| Raw Protocol Logs | Lulus | 50 |
| Alert Center | Lulus | 0 |
| Reports | Lulus | 0 |
| Device Management | Lulus | 2 |
| Sensor Management | Lulus | 2 |
| User Management | Lulus | 1 |

## Perbaikan dan validasi

- [x] Ringkasan kendaraan memisahkan GPS online, offline, belum terhubung serta kondisi pergerakan/kualitas posisi.
- [x] Status diagnostik diperbarui melalui stream perubahan data dan kedaluwarsa waktu.
- [x] Nama sensor dibaca dari field nama pada raw, dipetakan berdasarkan MAC, serta ditampilkan di daftar, Detail, dan kalibrasi.
- [x] Pencarian nama sensor BBM_SATU pada API live berhasil.
- [x] Nama BBM_SATU berasal dari sensor ••••85FC melalui GPS ••••3110; CFS300B berasal dari ••••8774 melalui GPS ••••3813 pada snapshot pemeriksaan.
- [x] Kalibrasi mengunci sensor, GPS penerima, kendaraan, kapasitas kendaraan, dan sumber sinyal.
- [x] Empty otomatis stabil 10 detik; Full memerlukan konfirmasi tangki penuh kemudian laporan stabil 10 detik.
- [x] Pengujian integrasi database lokal memverifikasi isolasi sensor, penyimpanan atomik, perubahan sumber, dan pencegahan penyimpanan ganda.
- [x] 46 pengujian UI dan seluruh pengujian Go dengan database lokal terisolasi lulus.
- [x] Dialog live menampilkan identitas sumber dan kapasitas 400 L kendaraan tanpa input kapasitas.
- [x] Receiver dan web telah diperbarui; health check berhasil.

## Batas verifikasi

- Kalibrasi fisik belum dilakukan pada perangkat live. Dibutuhkan tangki benar-benar kosong saat Start dan konfirmasi setelah pengisian sampai penuh.
- Perintah remote mesin, hapus data, dan perubahan hubungan kendaraan live tidak dijalankan dalam audit baca ini.
- Halaman Geofences, Fuel Analytics, Alert Center, dan Reports memberikan daftar kosong pada rentang bawaan. Respons kosong tidak membuktikan fungsi yang memerlukan data sudah diuji.
- Nilai historis tetap memakai kalibrasi yang berlaku saat pembacaan. Mengubah kapasitas kendaraan tidak mengganti hasil kalibrasi historis secara otomatis.
