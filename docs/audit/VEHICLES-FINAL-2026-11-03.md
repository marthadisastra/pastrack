# Vehicles — penyelesaian registry kendaraan, 3 November 2026

## Penjelasan bagian atas yang membingungkan

Template lama menggabungkan inspector satu kendaraan dengan daftar seluruh armada. Kode juga memilih kendaraan pertama secara otomatis, sehingga bagian atas terlihat seperti kendaraan khusus yang selalu aktif. Sebagian teks panel tersebut masih berupa contoh spesifikasi dan belum punya data pendukung.

**Alur akhir:** `#vehicles` hanya daftar seluruh kendaraan. Klik **Inspect** pada baris membuka `#vehicles/{id}/overview`. Tidak ada inspector otomatis, picker kendaraan terpilih, peta, atau Tank Spec pada halaman daftar. Halaman detail yang dibuka tersebut tetap mempunyai audit tersendiri; tidak dianggap selesai seluruhnya oleh pekerjaan registry ini.

## Checklist registry Vehicles: 15/15 lulus — 100%

| No | Fungsi | Hasil / bukti |
|---|---|---|
| 1 | Daftar armada terpisah dari inspector | Inspector otomatis dihapus; kembali dari Inspect tetap menampilkan daftar, bukan panel kendaraan terakhir |
| 2 | Informasi perangkat mengikuti hubungan GPS–kendaraan | Query telemetry mengikuti assignment GPS aktif; data sensor memakai gateway tersebut |
| 3 | Kolom sesuai data nyata | Plat/nama, fleet code, model/ID GPS, movement, speed, ignition, fuel L/%, sensor, GPS dan last ping |
| 4 | Tidak ada spesifikasi/demo yang tidak didukung | Tank Spec, driver, depot Cakung, contoh waktu perjalanan/suhu/polling dihapus dari daftar |
| 5 | Tambah kendaraan | Browser QA: QA UI VEH 0311A dibuat pada pastrack_test_local; input hanya plat/nama/fleet |
| 6 | Edit kendaraan | Browser QA memperbarui nama; backend trim/validation, nomor polisi unik dan row_version bertambah |
| 7 | Inspect per baris | Browser membuka #vehicles/158/overview untuk baris QA yang dipilih |
| 8 | Search plat/nama/kode fleet | Backend CONCAT_WS + parameter SQL; browser dan integrasi lulus |
| 9 | Filter fleet dan movement | Diterapkan sebelum LIMIT/cursor, bukan hanya terhadap 50 baris pada client |
| 10 | Pagination | Keyset server; Next/Previous diuji pada lebih dari 150 kendaraan QA; metadata page ikut dikirim SSE |
| 11 | Reset/Refresh dan Kelola GPS | Mengembalikan filter/cursor; tautan GPS tetap ke Device Management |
| 12 | Ekspor seluruh hasil filter | Browser file pertama 1 kendaraan QA-UI, file kedua 158 kendaraan lintas halaman, bukan hanya 50 baris yang terlihat |
| 13 | Pembaruan SSE per halaman | Subscription Vehicles hanya resource kendaraan terpaginasinya; mempertahankan row/cell dan mengirim perubahan metadata pagination |
| 14 | Hak akses | Viewer dapat membaca/Inspect, Register disembunyikan dan Edit tidak ditampilkan; backend viewer POST ditolak |
| 15 | Empty/error dan responsive | Moving tanpa hasil menampilkan empty state; input wajib/konflik ditangani; desktop body1280, mobile body390 tanpa overflow dokumen; tabel scroll internal |

## Skala dan batas yang disampaikan dengan jelas

- Daftar dirender per 50 kendaraan; API dibatasi maksimum 200 per permintaan. Tidak mengambil seluruh telemetri fleet untuk merender setiap halaman Vehicles.
- Filter kode fleet mendapat opsi dari distinct fleet_code yang tersimpan, bukan contoh Wingbox/Tanker dan bukan hanya kode pada halaman pertama.
- Ekspor membaca hasil filter dengan keyset pagination 200 per permintaan. Ini pembacaan berurutan, bukan snapshot transaksi tunggal jika data berubah selama ekspor.
- Kendaraan tanpa gateway berstatus UNASSIGNED dan tidak diberi GPS/fuel palsu.
- Ignition tetap NOT_INSTALLED. Riwayat pengukuran lama tetap data lama.
- Tidak menambahkan tombol hapus kendaraan pada pekerjaan ini. Tidak ada penghapusan data produksi, DROP, TRUNCATE, atau migrasi destruktif.
- Pengujian tambah/edit dilakukan pada database pengujian terisolasi port33307, bukan database live. Aplikasi QA memakai port18091. Aplikasi lokal biasa port18090 tetap memakai database live.
- Uji skala sangat besar dan pengujian seluruh halaman detail kendaraan bukan klaim dari skor registry15/15 ini.

## Pengujian

- `go test -race ./...` dengan database terisolasi: lulus.
- `go vet ./...`: lulus.
- 29 tes DOM: lulus (log vehicles-tests-20261103.txt).
- Tes inventory SQL: search nama/fleet, status MOVING menggunakan fixture GPS nyata dalam DB uji, filter UNASSIGNED, keyset2halaman, validasi input, viewer403, edit dan ignition.
- Tes delta pagination: has_more/next_cursor berubah walaupun field baris tidak berubah.
- Browser QA: tambah, edit, cari, filter, reset, next/previous, inspect, ekspor1dan158baris, responsive390px, tanpa console error.

**Deployment gate:** checklist registry dinyatakan15/15 sebelum deployment. Perubahan baru diterapkan ke live setelah gate ini lulus; halaman lain tidak dinaikkan skornya tanpa audit.
