# Audit dashboard pada kiriman baru — 3 November 2026

## Hasil live

- Gateway ONLINE; GPS valid; kendaraan STOPPED (speed 0); sensor connected.
- Raw terbaru yang diperiksa 6427–6438, payload CFS300B berhasil didekode.
- Kalibrasi aktif ID4: fuel_height_raw Empty229=0L, Full4992=120L.
- Raw di atas Full menghasilkan validity out_of_range dan liters/percent NULL. Ini penolakan konversi, bukan kehilangan kiriman.
- Pemilik mengonfirmasi agar kalibrasi229–4992 dipertahankan. Tidak mengubah titik, mengekstrapolasi, mengapit ke100%, atau menyamarkan nilai sebelumnya sebagai kondisi saat ini.
- Summary live konsisten: total1, online1, stopped1, offline0, GPS coverage1, valid current fuel coverage0.

## Perbaikan source lokal

- [x] Kolom Sensor Status hanya koneksi Terhubung/Terputus/Data lama, tanpa validity fuel dan timestamp duplikat.
- [x] Fuel di luar rentang ditampilkan sebagai Di atas batas kalibrasi/Di luar rentang kalibrasi, bukan N/A tanpa alasan.
- [x] Tombol Detail pembacaan menampilkan nilai raw, Empty, Full, calibration ID, waktu penerimaan dan alasan liter/persen tidak dihitung.
- [x] API fleet memasok raw dan batas kalibrasi untuk penjelasan tersebut.
- [x] 31tesDOM, go test-race, go vet lulus.
- [ ] QA visual browser dan klik dialog: tertahan karena Local browser use disabled in Work settings.
- [ ] Deployment belum dilakukan; gate100% belum terpenuhi untuk perubahan ini.

Perbaikan Telematics Overview dari pekerjaan sebelumnya juga masih lokal/belum deploy. Jangan menyatakan kedua perubahan sudah tampil di server live.
