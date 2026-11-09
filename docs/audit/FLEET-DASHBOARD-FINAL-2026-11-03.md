# Fleet Dashboard — checklist penyelesaian 3 November 2026

**Fungsi dashboard sesuai sumber GV30CEU/CFS300B: 20 dari 20 kriteria penerimaan lulus (100%).** Ini menyatakan checklist fungsi yang tercantum di bawah selesai, bukan jaminan bebas bug pada semua kondisi, atau bukti perangkat fisik sedang online. Halaman lain tidak berubah menjadi 100%.

## Checklist penerimaan

| No | Fungsi | Status | Bukti |
|---|---|---|---|
| 1 | Total fleet dan hubungan gateway–kendaraan menggunakan data aktif | Lulus | Query fleetRows bersama; authenticated API live |
| 2 | Kategori moving/stopped/offline/no-fix/stale/unknown/unassigned eksklusif | Lulus | Unit test status, batas speed dan jumlah kategori |
| 3 | Fuel liter/persen memakai kalibrasi, bukan nilai raw yang dianggap liter | Lulus | API/CSV dan browser: 112.618 L / 93.85% |
| 4 | Kesegaran gateway, GPS, sensor, dan fuel dibedakan | Lulus | Tes buffered fuel: penerimaan baru + pengukuran lama bukan fresh; waktu ukur null bukan fresh |
| 5 | Ignition fisik selalu NOT_INSTALLED | Lulus | Backend, tabel, ekspor dan penjelasan data |
| 6 | Chip All/Moving/Stopped/Offline/Active Alerts memfilter tabel dan berubah aktif | Lulus | DOM regression, klik browser; moving/alerts kosong, offline berisi kendaraan |
| 7 | Filter fleet memakai fleet_code sebenarnya | Lulus | Dropdown FLT-003 di browser dan fixture multi-fleet |
| 8 | Filter kondisi data bekerja bersama filter fleet | Lulus | Browser FLT-003 + ONLINE menghasilkan empty state; fixture gabungan |
| 9 | Search dan reset mengembalikan hasil dengan benar | Lulus | DOM dan browser |
| 10 | Legenda distribusi dapat digunakan sebagai filter via klik/keyboard | Lulus | Handler keyboard, DOM dan klik legenda offline di browser |
| 11 | Distribusi donut menampilkan total dan persentase sebenarnya | Lulus | 1 kendaraan, 100% offline; unit invariant kategori |
| 12 | Activity Stream memakai laporan GPS/fuel terdekode | Lulus | API GPS/fuel union, browser; tidak menebak raw event code |
| 13 | Aktivitas baru tetap berurutan terbaru dahulu tanpa mengganti kartu lama | Lulus | Tes insert incremental + identitas node tetap |
| 14 | SSE mengirim perubahan field sesudah commit, bukan polling halaman | Lulus | Tes integrasi HTTP SSE DB terisolasi; tes DOM tidak menambah fetch |
| 15 | Kedaluwarsa data memicu pembaruan tanpa interval data polling | Lulus | One-shot deadline timer; heartbeat hanya koneksi/sesi |
| 16 | Pencarian peta, zoom, Fit All dan marker menggunakan GPS sebenarnya | Lulus | Browser; fungsi marker incremental dan filter map terkait |
| 17 | Detail kendaraan, detail aktivitas, dan dialog definisi data tersedia | Lulus | Handler dan klik browser untuk detail/penjelasan; tidak mengubah data bisnis |
| 18 | CSV sesuai hasil filter, ekspor kosong ditolak | Lulus | Browser download; file diperiksa dengan csv.DictReader |
| 19 | Layout desktop/mobile sesuai dan dua kartu kanan sejajar peta | Lulus | Desktop 1280: map 640 px, side stack 640 px, masing-masing kartu 312 px + gap 16 px. Mobile 390: body width 390, tabel scroll internal |
| 20 | Admin/operator/viewer dapat membaca, anonim ditolak, sesi yang dicabut berhenti streaming | Lulus | API/DOM role tests; HTTP SSE integration revocation |

## Bukti otomatis

- `go test -race ./...` dengan database pastrack_test_local port 33307: lulus.
- `go vet ./...`: lulus.
- 24 tes DOM: lulus. Log: `dashboard-tests-20261103.txt`.
- Stream diuji pada database terisolasi: snapshot pertama, update transaksi lalu Notify, delta hanya field berubah, auth_version dicabut lalu stream ditutup.
- Data bisnis produksi tidak ditambah sebagai fixture pengujian.

## Penyelesaian kontrol yang sebelumnya kosong

- Dropdown wilayah tanpa data diganti filter kondisi data yang benar-benar tersedia.
- Dropdown fleet menggunakan kode fleet tersimpan.
- Config dan menu informasi membuka definisi metrik/kemampuan perangkat, bukan form palsu.
- Traffic/satellite/corridor tidak lagi dipajang sebagai tombol mati. Itu layanan peta eksternal, bukan data GV30CEU/CFS300B. Penjelasan sumber peta tersedia di dialog.

## Batas operasional, bukan data yang dibuat-buat

- Saat QA, gateway masih offline dan GPS/fuel terakhir masih lama. Tidak ada klaim uji perangkat fisik bergerak secara langsung selama sesi ini.
- Kalibrasi 229–4992 untuk 120 L dipakai sesuai konfirmasi pemilik; angka di luar rentang tidak dipaksa menjadi volume valid.
- Waktu pengukuran sensor yang tidak ada tetap tidak tersedia. Timestamp GPS pada paket dapat dipakai sebagai konteks pengukuran untuk pemeriksaan freshness, bukan diklaim sebagai timestamp independen dari sensor.
- Uji beban skala besar, gangguan jaringan panjang dan validasi metrologi fisik adalah pekerjaan operasional terpisah; tidak termasuk 20 checklist fungsi ini.

## Deployment

Backend web dan aset sudah deploy ke server live. Source lokal menggunakan versi yang sama serta database/stream live yang sama. Receiver tidak memerlukan build baru pada perubahan ini karena jalur event setelah commit sudah dipakai. Tidak ada DROP/TRUNCATE atau migrasi destruktif.
