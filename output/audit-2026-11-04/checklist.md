# Checklist fungsi PasTrack — 4 November 2026 WIB

Chrome 156.0.8078.80, desktop 1366×768 dan mobile 390×844. Audit mencakup 33 rute, 1.066 instans kontrol dan 688 kelompok kontrol. Kontrol berulang diperiksa dengan sampel per jenis. Font diukur dari hasil render. Alur tulis tertentu diperiksa pada database lokal.

**Cara membaca hasil**. Respons UI berarti dialog, navigasi atau permintaan data teramati. Bukti itu belum membuktikan hasil akhir. Status tanpa efek mencakup kondisi ketika hasilnya tetap sama atau membutuhkan data. Skenario terarah di bawah memperjelas beberapa hasil pemindaian umum.

## Pemeriksaan terarah

- **global-account-keyboard** · live · Gagal · Enter pada avatar role=button tidak membuka menu akun
- **global-account-menu** · live · Respons UI teramati · ['Profil saya', 'Notifikasi', 'Audit log', 'Keluar']
- **global-profile** · live · Respons UI teramati · Dialog profil terbuka
- **fuel-spa-delegation** · live · Gagal · {'playbackMoves': True, 'toast': 'Belum ada data atau konfigurasi tervalidasi untuk aksi ❚❚ Pause.'}
- **fuel-play-pause** · live · Unduhan terverifikasi · Waktu bergerak dan tombol beralih Play/Pause pada 721 pembacaan kendaraan terpilih.
- **fuel-event-focus** · live · Unduhan terverifikasi · ↗ 0 km/h · -6.46289, 107.81485 \| Sensor 1746: 244,15 L (61,04 %) \| ⚠ EVT-1 · -40 L
- **fuel-investigation-dialog** · live · Respons UI teramati · {'fields': 2}
- **fuel-bap-download** · live · Unduhan terverifikasi · {'bytes': 1542, 'filename': 'BAP-Fuel-EVT-1.html', 'correctVehicle': True}
- **fuel-map-tooltip** · live · Belum terverifikasi · {'tooltip': []}
- **trip-history-scope** · live · Unduhan terverifikasi · 721 titik GPS · 2 perjalanan · Pilih perjalanan atau tekan Play.
- **trip-journey-playback** · live · Unduhan terverifikasi · Putar per perjalanan, waktu berjalan dan Pause teruji.
- **live-tracking-mobile-width** · live · Gagal · Lebar halaman 616 px pada viewport 390 px.
- **vehicle-create-persist** · local · Unduhan terverifikasi · {'count': 1}
- **report-csv** · local · Unduhan terverifikasi · {'status': 200, 'bytes': 79956, 'signatureValid': True, 'layoutVerified': False}
- **report-xlsx** · local · Unduhan terverifikasi · {'status': 200, 'bytes': 59570, 'signatureValid': True, 'layoutVerified': False}
- **report-pdf** · local · Unduhan terverifikasi · {'status': 200, 'bytes': 144696, 'signatureValid': True, 'layoutVerified': False}
- **live-js-errors** · live · Gagal · Kesalahan Leaflet _leaflet_pos setelah perpindahan ke Live Tracking.

## Temuan lintas halaman

- Reports **Generate** memunculkan pesan aksi belum tersedia. Pembuatan dan unduhan CSV/XLSX/PDF melalui API lokal berhasil. Tata letak berkas laporan belum diperiksa.
- Setelah masuk Fuel Events dari Vehicles lewat sidebar, Play/Pause berjalan tetapi juga memunculkan pesan aksi belum tersedia. Hasil pengujian terarah mengonfirmasi fungsi playback dan masalah pesan secara terpisah.
- Pada 390 px, Live Tracking melebar ke 616 px dan Raw Protocol Logs ke 452 px.
- Avatar akun dapat dibuka dengan klik. Enter pada avatar tidak membuka menu.
- Role operator menerima HTTP 403 saat membuat kendaraan pada pengujian lokal. Hak yang diinginkan perlu ditetapkan sebelum menyebutnya cacat.
- Judul utama berkisar 18–24 px pada halaman yang memilikinya. Fuel Events memakai 9–11 px untuk beberapa label, metadata dan kontrol. Usulan skala untuk penyelarasan adalah judul 24 px, judul panel 16–18 px, isi dan kontrol 13–14 px, serta metadata 11–12 px.
- Tes Go mencatat 75 hasil lulus termasuk subtes. Tes DOM mencatat 42 lulus. Cakupan browser memakai Chrome. Perintah perangkat, kirim email, dan mutasi produksi tidak dieksekusi.

## Checklist per halaman

### fleet-dashboard

34 kelompok kontrol. Respons UI teramati 16, Tidak ada efek teramati 11, Input bisa diubah 3, Belum terverifikasi 1, Terbatas 1, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 2. Tombol aktif di bawah 24 px: 1. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 75.
Ukuran font desktop (px) · button 11–16, h1 24–24, p 12–14, h2 15–15, th 11–11, td 13–13, small 10–10.4.

[Desktop](screenshots/fleet-dashboard-desktop.png) · [Mobile](screenshots/fleet-dashboard-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · CONFIG tune | Respons UI teramati | Dialog terbuka: Definisi data Fleet Dashboard; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · All Vehicles 3 | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Moving 0 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Stopped 0 | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Offline 3 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Active Alerts 0 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input text · Cari kendaraan di peta | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · crop_free Fit All | Respons UI teramati | Belum ada posisi GPS valid. |
| button button · add | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · remove | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button button · more_vert | Respons UI teramati | Dialog terbuka: Definisi data Fleet Dashboard; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| div · Bergerak (&gt;3 km/jam) 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Berhenti (≤3 km/jam) 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · GPS offline 3 · 100.0% · title Filter kendaraan berdasarkan status ini | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| div · Online · GPS lama 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| div · Online · tanpa GPS fix 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Kecepatan tidak tersedia 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Belum terpasang GPS 0 · 0.0% · title Filter kendaraan berdasarkan status ini | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.14 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.13 WIB 0 km/h · -6.1334610, 106.8724130 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Buka detail kendaraan |
| button button · B 1307 KNQ · Pembacaan fuel CFS300B Diterima 3 Nov 2026, 18.58.14 WIB · Waktu ukur tidak tersedia 2,7 L · 2,25 % · valid | Respons UI teramati | Dialog terbuka: Pembacaan fuel CFS300B; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Buka detail kendaraan |
| button button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.09 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.07 WIB 0 km/h · -6.1334810, 106.8724130 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Buka detail kendaraan |
| button button · B 1307 KNQ · Pembacaan fuel CFS300B Diterima 3 Nov 2026, 18.58.09 WIB · Waktu ukur tidak tersedia 2,7 L · 2,25 % · valid | Respons UI teramati | Dialog terbuka: Pembacaan fuel CFS300B; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Buka detail kendaraan |
| button button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.04 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.02 WIB 0 km/h · -6.1333210, 106.8723640 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Buka detail kendaraan |
| input text · Filter plate / fleet... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · Filter kode fleet ×2 · opsi Semua fleet, FLT-003, Logistik Antar Kota | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · download Export CSV | Terbatas | Belum ada data untuk diekspor. |
| button button · Reset filter | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Aktifkan Mesin ×3 · title Status remote dari output relay terakhir · 3 Nov 2026, 18.03.56 WIB; ignition ditampilkan terpisah. | Respons UI teramati | GPS belum memiliki koneksi TCP aktif |
| button · Detail ×3 | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps
- Filter kendaraan berdasarkan status ini
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · 3 Nov 2026, 18.03.56 WIB; ignition ditampilkan terpisah.
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · belum ada laporan; ignition ditampilkan terpisah.

### vehicles

12 kelompok kontrol. Input bisa diubah 1, Respons UI teramati 8, Unduhan terverifikasi 1, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 43.
Ukuran font desktop (px) · h1 18–18, p 11–13, button 11–16, h2 15–15, th 11–11, td 13–13, small 10–10.4.

[Desktop](screenshots/vehicles-desktop.png) · [Mobile](screenshots/vehicles-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input text · Cari kendaraan | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · Filter fleet kendaraan ×2 · opsi Semua fleet, FLT-003, Logistik Antar Kota | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · file_download Export Log | Unduhan terverifikasi | File diunduh; metadata dan ukuran dicatat. |
| button button · add_circle Register Unit | Respons UI teramati | Dialog terbuka: Tambah Vehicles; penyimpanan belum diklaim. · Isi dialog INPUT / plate_number / Nomor polisi, INPUT / vehicle_name / Nama kendaraan, INPUT / fleet_code / Kode fleet (opsional), BUTTON / Batal, BUTTON / Simpan |
| button button · Reset filter | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Refresh | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Kelola GPS | Respons UI teramati | Navigasi ke #device-management |
| button button · Aktifkan Mesin ×3 · title Status remote dari output relay terakhir · belum ada laporan; ignition ditampilkan terpisah. | Respons UI teramati | GPS belum memiliki koneksi TCP aktif |
| button · Inspect ×3 | Respons UI teramati | Navigasi ke #vehicles/3/overview |
| button · Edit ×3 | Respons UI teramati | Dialog terbuka: Edit Vehicles; penyimpanan belum diklaim. · Isi dialog INPUT / plate_number / Nomor polisi, INPUT / vehicle_name / Nama kendaraan, INPUT / fleet_code / Kode fleet (opsional), BUTTON / Batal, BUTTON / Simpan |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · belum ada laporan; ignition ditampilkan terpisah.
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · 3 Nov 2026, 18.03.56 WIB; ignition ditampilkan terpisah.

### live-tracking

12 kelompok kontrol. Respons UI teramati 4, Aksi belum tersedia 3, Terbatas 2, Tidak ada efek teramati 2, Belum terverifikasi 1.
Mobile 616 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 7. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 33.
Ukuran font desktop (px) · button 11–11.

[Desktop](screenshots/live-tracking-desktop.png) · [Mobile](screenshots/live-tracking-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · Semua N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Bergerak N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Ignition: NOT_INSTALLED N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Ignition: NOT_INSTALLED N/A. |
| button button · Parkir N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Alert Anomali N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Alert Anomali N/A. |
| button button · fence Geofence · title Geofence SPBU & Depo | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Geofence. |
| button button · local_fire_department Heatmap BBM · title Heatmap BBM Konsumsi | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| button button · traffic Traffic Live · title Jalur Bebas Hambatan Tol | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button button · Buka Kasus | Respons UI teramati | Dialog terbuka: Alert #1; penyimpanan belum diklaim. · Isi dialog BUTTON / × |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Geofence SPBU & Depo
- Heatmap BBM Konsumsi
- Jalur Bebas Hambatan Tol
- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### trip-history

13 kelompok kontrol. Respons UI teramati 3, Input bisa diubah 2, Belum terverifikasi 2, Aksi belum tersedia 1, Tidak ada efek teramati 2, Nonaktif 3.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 22.
Ukuran font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, button 12–13, h2 16–16, th 12–12, td 12–12.

[Desktop](screenshots/trip-history-desktop.png) · [Mobile](screenshots/trip-history-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan · opsi Pilih kendaraan, B 9102 PTR · Hino 500 Jakarta–Cirebon, B 9101 PTR · Hino 500 Jakarta–Cirebon, B 1307 KNQ · DT AXOR | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input date · Dari (WIB) | Respons UI teramati | Perubahan input memicu permintaan data. |
| input date · Sampai (WIB) | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button submit · Tampilkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Lihat tanggal data terakhir | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |
| button button · Lihat seluruh rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Lihat seluruh rute. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan · opsi 60×, 300×, 600×, 1800× | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input range · Waktu playback GPS | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### geofences

29 kelompok kontrol. Aksi belum tersedia 12, Terbatas 3, Respons UI teramati 6, Input bisa diubah 1, Belum terverifikasi 2, Tidak ada efek teramati 3, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 4. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 63.
Ukuran font desktop (px) · h1 18–18, button 11–16, p 12–12, th 11–11, td 12–12.

[Desktop](screenshots/geofences-desktop.png) · [Mobile](screenshots/geofences-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · upload_file Import KML / GeoJSON | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Import KML / GeoJSON. |
| button button · download Ekspor Log Kunjungan | Terbatas | Belum ada data untuk diekspor. |
| button button · add_circle + Buat Geofence Baru | Respons UI teramati | Dialog terbuka: Tambah Geofences; penyimpanan belum diklaim. · Isi dialog INPUT / name / name, SELECT / shape / circle polygon, INPUT / center_lat / center lat, INPUT / center_lon / center lon, INPUT / radius_m / radius m, TEXTAREA / polygon_geojson / polygon geojson, SELECT / is_active / true false, BUTTON / Batal, BUTTON / Simpan |
| button button · Semua | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Pool & Depo | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pool & Depo. |
| button button · Customer Warehouse | Respons UI teramati | Dialog terbuka: Rentang waktu · Asia/Jakarta; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, INPUT / from / Dari, INPUT / to / Sampai, BUTTON / Terapkan |
| button button · SPBU / Refuel Point | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi SPBU / Refuel Point. |
| button button · Restricted / Blackzone | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Restricted / Blackzone. |
| input text · Cari geofence, kode, atau POI... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · splitscreen · title Split Editor Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi splitscreen. |
| button button · map · title Full Map Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi map. |
| button button · table_rows · title List Registry Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi table_rows. |
| button button · Filter Aturan tune | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · polyline Poligon (Aktif) · title Gambar Poligon Kustom | Respons UI teramati | Klik titik poligon pada peta, klik dua kali untuk menyelesaikan. |
| button button · radio_button_unchecked Radius · title Gambar Radius Lingkaran | Respons UI teramati | Klik pusat radius, lalu klik batas radius pada peta. |
| button button · route Koridor · title Koridor Buffer Rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Koridor. |
| button button · edit · title Edit Titik Vertex | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi edit. |
| button button · straighten · title Penggaris / Ukur Jarak | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi straighten. |
| button button · delete · title Hapus Gambar | Belum terverifikasi | Aksi tulis/otorisasi/perintah fisik tidak dieksekusi di live. |
| button button · Vector Fleet | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Satelit | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| button button · Lalu Lintas | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button button · edit_note Ubah Batas Vertex | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Ubah Batas Vertex. |
| button button · notifications_active Konfigurasi Notifikasi | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Konfigurasi Notifikasi. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Split Editor Mode
- Full Map Mode
- List Registry Mode
- Gambar Poligon Kustom
- Gambar Radius Lingkaran
- Koridor Buffer Rute
- Edit Titik Vertex
- Penggaris / Ukur Jarak
- Hapus Gambar
- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### fuel-monitoring

25 kelompok kontrol. Respons UI teramati 19, Tidak ada efek teramati 1, Nonaktif 2, Aksi belum tersedia 2, Input bisa diubah 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 56. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 703.
Ukuran font desktop (px) · h1 24–24, p 13–13, button 11–12, h2 18–18, th 11–11, td 13–13, small 10–10, h3 18–18.

[Desktop](screenshots/fuel-monitoring-desktop.png) · [Mobile](screenshots/fuel-monitoring-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · Today (24h) | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Last 7 Days | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Last 30 Days | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · tune Custom | Respons UI teramati | Dialog terbuka: Rentang waktu · Asia/Jakarta; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, INPUT / from / Dari, INPUT / to / Sampai, BUTTON / Terapkan |
| button button · file_download Audit Export | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Liters | Respons UI teramati | Grafik hanya memakai pembacaan tervalidasi. Sinyal raw tidak dikonversi menjadi liter tanpa kalibrasi. |
| button button · % Fill | Respons UI teramati | Grafik hanya memakai pembacaan tervalidasi. Sinyal raw tidak dikonversi menjadi liter tanpa kalibrasi. |
| button button · Kalman Filter · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Raw BLE | Respons UI teramati | Grafik hanya memakai pembacaan tervalidasi. Sinyal raw tidak dikonversi menjadi liter tanpa kalibrasi. |
| button button · 1h | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi 1h. |
| button button · 6h | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 24h | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Flag Anomaly | Respons UI teramati | Dialog terbuka: Alert #1743; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Verify Normal | Respons UI teramati | Dialog terbuka: Alert #1743; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · View All Historical Fuel Audits arrow_forward | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi View All Historical Fuel Audits. |
| button button · All Vehicles | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Normal | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Refueling | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Fuel Drop Alert | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Sensor Disconnected | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input text · Filter plate, driver, status... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · filter_list Columns | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button · Detail ×50 | Respons UI teramati | Dialog terbuka: fuel/readings #1743; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Moving (avg 62 km/h)
- Stopped (Rest Area KM 19)
- Moving (avg 55 km/h)
- Stopped (Anomaly Alert KM 57)
- Stopped (SPBU KM 102A Refueling)
- Moving (avg 68 km/h)

### fuel-analytics

23 kelompok kontrol. Respons UI teramati 8, Nonaktif 3, Unduhan terverifikasi 1, Aksi belum tersedia 3, Input bisa diubah 3, Tersembunyi 5.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 1. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 76.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–12, h2 18–18, th 11–11, td 13–13, small 10–10.

[Desktop](screenshots/fuel-analytics-desktop.png) · [Mobile](screenshots/fuel-analytics-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · Hari Ini | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 7 Hari | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 28 Hari | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Rentang calendar_today | Respons UI teramati | Dialog terbuka: Rentang waktu · Asia/Jakarta; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, INPUT / from / Dari, INPUT / to / Sampai, BUTTON / Terapkan |
| button button · description Ekspor BAP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · download Download Rekap Audit | Unduhan terverifikasi | File diunduh; metadata dan ukuran dicatat. |
| button button · Semua Insiden N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Perlu Respon N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Pengisian Tak Wajar | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pengisian Tak Wajar. |
| button button · Dinyatakan Valid | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Dinyatakan Valid. |
| input search · Cari data | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input datetime-local · Dari waktu | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input datetime-local · Sampai waktu | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Apply | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Apply. |
| button button · Refresh | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Respons UI teramati | Dialog terbuka: fuel/events #1; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · close | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| select · select · opsi All / belum dikonfigurasi | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| textarea · Tulis catatan konfirmasi telepon supir, foto kondisi segel tangki, dsb... | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| button button · Batal | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| button button · Simpan Berita Acara (BAP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.

### fuel-events

22 kelompok kontrol. Nonaktif 7, Input bisa diubah 4, Respons UI teramati 3, Aksi belum tersedia 1, Tidak ada efek teramati 1, Belum terverifikasi 6.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 38.
Ukuran font desktop (px) · h1 20–20, button 10–11, label 10–11, p 9–12, small 9–10, h2 14–14.

[Desktop](screenshots/fuel-events-desktop.png) · [Mobile](screenshots/fuel-events-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · ↓ Unduh BAP Telematika | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ◎ Investigasi Anomali | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kendaraan · opsi All, B 1307 KNQ · DT AXOR, B 9101 PTR · Hino 500 Jakarta–Cirebon, B 9102 PTR · Hino 500 Jakarta–Cirebon | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input date · Start date (WIB) | Respons UI teramati | Perubahan input memicu permintaan data. |
| input date · End date (WIB) | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · ↻ Terapkan | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ↻ Terapkan. |
| input checkbox · Geofence | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| a · + · title Zoom in | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · −1m | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · +1m | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan · opsi 60×, 120×, 300×, 600×, 1800× | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input range · Waktu playback rute dan BBM | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| summary · Event & detail perjalanan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Semua Event (0) | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |
| button button · Anomali & Drop (0) | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |
| button button · Refueling (0) | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |
| button button · Idle (0) | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |
| summary · Aturan deteksi & dasar perhitungan | Belum terverifikasi | locator.click: Timeout 1800ms exceeded. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### gps-quality

13 kelompok kontrol. Aksi belum tersedia 4, Input bisa diubah 2, Respons UI teramati 3, Tidak ada efek teramati 1, Belum terverifikasi 2, Nonaktif 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 50. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 365.
Ukuran font desktop (px) · p 13–13, button 11–12, h2 18–18, th 11–11, td 13–13, small 10–10.

[Desktop](screenshots/gps-quality-desktop.png) · [Mobile](screenshots/gps-quality-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · satellite_alt GPS Quality N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi GPS Quality N/A. |
| button button · wifi_tethering Device Connectivity | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Device Connectivity. |
| button button · network_check Data Activity N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Data Activity N/A. |
| button button · terminal Raw Protocol Logs LIVE | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Raw Protocol Logs LIVE. |
| input text · Filter Plate / IMEI / Driver... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · filter_list All Statuses | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · download Export CSV | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Detail ×50 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| select · device-imei-select ×2 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| input text · Search hex / CRC... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · autorenew Auto-Scroll: ON | Respons UI teramati | Auto-scroll dinonaktifkan. |

### device-connectivity

11 kelompok kontrol. Nonaktif 2, Unduhan terverifikasi 1, Respons UI teramati 6, Input bisa diubah 1, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 50. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 397.
Ukuran font desktop (px) · h1 24–24, p 13–13, button 11–12, h2 18–18, th 11–11, td 13–13, small 10–10.

[Desktop](screenshots/device-connectivity-desktop.png) · [Mobile](screenshots/device-connectivity-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · wifi_protected_setup Ping Delayed · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · file_download Export CSV | Unduhan terverifikasi | File diunduh; metadata dan ukuran dicatat. |
| a · satellite_alt GPS Quality | Respons UI teramati | Navigasi ke |
| a · cell_tower Device Connectivity | Respons UI teramati | Navigasi ke |
| a · network_check Data Activity 0 | Respons UI teramati | Navigasi ke |
| a · terminal Raw Protocol Logs | Respons UI teramati | Navigasi ke |
| input text · Filter IMEI / Plate / Carrier / Firmware... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · statusFilter ×3 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button · Detail ×50 | Respons UI teramati | Dialog terbuka: devices/connectivity #58; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- 4G LTE Cat-1: 142 units
- 2G GSM Fallback: 21 units
- No Signal / Search: 5 units
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya

### data-activity

16 kelompok kontrol. Nonaktif 2, Tidak ada efek teramati 1, Respons UI teramati 10, Input bisa diubah 1, Belum terverifikasi 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 54. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 286.
Ukuran font desktop (px) · h1 24–24, button 11–16, p 11–13, h2 15–15, th 11–11, td 13–13, small 10–10.

[Desktop](screenshots/data-activity-desktop.png) · [Mobile](screenshots/data-activity-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · bolt Force Queue Flush · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · file_download Export Ingestion Metrics (CSV) | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · satellite_alt GPS Quality | Respons UI teramati | Navigasi ke |
| a · wifi_tethering Device Connectivity | Respons UI teramati | Navigasi ke |
| a · network_check Data Activity N/A | Respons UI teramati | Navigasi ke |
| a · terminal Raw Protocol Logs | Respons UI teramati | Navigasi ke |
| button button · Last 1h | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 6h | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 24h (Live) | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · 7 Days | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input text · Filter IMEI or Plate... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×2 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · refresh · title Reload Feed | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail ×50 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Reload Feed

### raw-protocol-logs

63 kelompok kontrol. Respons UI teramati 7, Aksi belum tersedia 2, Input bisa diubah 2, Belum terverifikasi 51, Nonaktif 1.
Mobile 452 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 56.
Ukuran font desktop (px) · h1 24–24, p 12–13, button 11–16.

[Desktop](screenshots/raw-protocol-logs-desktop.png) · [Mobile](screenshots/raw-protocol-logs-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · satellite_alt GPS Quality | Respons UI teramati | Navigasi ke |
| a · wifi_tethering Device Connectivity | Respons UI teramati | Navigasi ke |
| a · network_check Data Activity N/A | Respons UI teramati | Navigasi ke |
| a · terminal Raw Protocol Logs LIVE STREAMING | Respons UI teramati | Navigasi ke |
| button button · Stream Active | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Stream Active. |
| button button · south Auto-Scroll: ON | Respons UI teramati | Auto-scroll dinonaktifkan. |
| input text · Filter hex / descriptor / CRC / IMEI... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · delete_sweep Clear · title Clear Console Memory | Belum terverifikasi | Aksi tulis/otorisasi/perintah fisik tidak dieksekusi di live. |
| button button · content_copy Copy Raw · title Copy Selected Payload | Respons UI teramati | Payload dengan identitas disamarkan telah disalin. |
| button button · download Export .PCAP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · 31 Okt 2026, 12.00.00 WIB UNKNOWN · 80201E0306 Raw #45344 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.59.30 WIB UNKNOWN · 80201E0306 Raw #45343 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.59.00 WIB UNKNOWN · 80201E0306 Raw #45342 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.58.30 WIB UNKNOWN · 80201E0306 Raw #45341 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.58.00 WIB UNKNOWN · 80201E0306 Raw #45340 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.57.30 WIB UNKNOWN · 80201E0306 Raw #45339 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.57.00 WIB UNKNOWN · 80201E0306 Raw #45338 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.56.30 WIB UNKNOWN · 80201E0306 Raw #45337 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.56.00 WIB UNKNOWN · 80201E0306 Raw #45336 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.55.30 WIB UNKNOWN · 80201E0306 Raw #45335 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.55.00 WIB UNKNOWN · 80201E0306 Raw #45334 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.54.30 WIB UNKNOWN · 80201E0306 Raw #45333 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.54.00 WIB UNKNOWN · 80201E0306 Raw #45332 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.53.30 WIB UNKNOWN · 80201E0306 Raw #45331 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.53.00 WIB UNKNOWN · 80201E0306 Raw #45330 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.52.30 WIB UNKNOWN · 80201E0306 Raw #45329 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.52.00 WIB UNKNOWN · 80201E0306 Raw #45328 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.51.30 WIB UNKNOWN · 80201E0306 Raw #45327 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.51.00 WIB UNKNOWN · 80201E0306 Raw #45326 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.50.30 WIB UNKNOWN · 80201E0306 Raw #45325 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.50.00 WIB UNKNOWN · 80201E0306 Raw #45324 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.49.30 WIB UNKNOWN · 80201E0306 Raw #45323 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.49.00 WIB UNKNOWN · 80201E0306 Raw #45322 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.48.30 WIB UNKNOWN · 80201E0306 Raw #45321 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.48.00 WIB UNKNOWN · 80201E0306 Raw #45320 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.47.30 WIB UNKNOWN · 80201E0306 Raw #45319 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.47.00 WIB UNKNOWN · 80201E0306 Raw #45318 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.46.30 WIB UNKNOWN · 80201E0306 Raw #45317 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.46.00 WIB UNKNOWN · 80201E0306 Raw #45316 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.45.30 WIB UNKNOWN · 80201E0306 Raw #45315 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.45.00 WIB UNKNOWN · 80201E0306 Raw #45314 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.44.30 WIB UNKNOWN · 80201E0306 Raw #45313 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.44.00 WIB UNKNOWN · 80201E0306 Raw #45312 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.43.30 WIB UNKNOWN · 80201E0306 Raw #45311 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.43.00 WIB UNKNOWN · 80201E0306 Raw #45310 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.42.30 WIB UNKNOWN · 80201E0306 Raw #45309 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.42.00 WIB UNKNOWN · 80201E0306 Raw #45308 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.41.30 WIB UNKNOWN · 80201E0306 Raw #45307 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.41.00 WIB UNKNOWN · 80201E0306 Raw #45306 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.40.30 WIB UNKNOWN · 80201E0306 Raw #45305 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.40.00 WIB UNKNOWN · 80201E0306 Raw #45304 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.39.30 WIB UNKNOWN · 80201E0306 Raw #45303 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.39.00 WIB UNKNOWN · 80201E0306 Raw #45302 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.38.30 WIB UNKNOWN · 80201E0306 Raw #45301 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.38.00 WIB UNKNOWN · 80201E0306 Raw #45300 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.37.30 WIB UNKNOWN · 80201E0306 Raw #45299 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.37.00 WIB UNKNOWN · 80201E0306 Raw #45298 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.36.30 WIB UNKNOWN · 80201E0306 Raw #45297 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.36.00 WIB UNKNOWN · 80201E0306 Raw #45296 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · 31 Okt 2026, 11.35.30 WIB UNKNOWN · 80201E0306 Raw #45295 · Klik untuk payload dan hasil parsing | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · schema · title Reload Schema | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| input text · input | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · Send send | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Send. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Clear Console Memory
- Copy Selected Payload
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Reload Schema

### alert-center

20 kelompok kontrol. Respons UI teramati 6, Aksi belum tersedia 5, Input bisa diubah 2, Tidak ada efek teramati 1, Nonaktif 5, Terbatas 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 54.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16.

[Desktop](screenshots/alert-center-desktop.png) · [Mobile](screenshots/alert-center-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · tune Konfigurasi Aturan Eskalasi | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · file_download Unduh Rekap Insiden Jam (PDF/Excel) | Respons UI teramati | Dialog terbuka: Tambah Laporan armada; penyimpanan belum diklaim. · Isi dialog SELECT / report_type / positions trips fuel events, SELECT / format / csv xlsx pdf, INPUT / from / from, INPUT / to / to, INPUT / vehicle_id / vehicle id, BUTTON / Batal, BUTTON / Simpan |
| button button · add_alert + Buat Tiket Manual / Lapor Insiden | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi + Buat Tiket Manual / Lapor Insiden. |
| button button · Semua Alert N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Kritis / Butuh Respon N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · local_gas_station Pencurian & Fluktuasi BBM N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pencurian & Fluktuasi BBM N/A. |
| button button · explore Geofence & Rute N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Geofence & Rute N/A. |
| button button · cable Sensor Tamper & GPS N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Sensor Tamper & GPS N/A. |
| button button · task_alt Terselesaikan Hari Ini N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input search · Cari no tiket, plat nomor, lokasi, atau jenis insiden... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×2 · opsi Semua status, MOVING, STOPPED, STALE, valid, uncalibrated, open, acknowledged | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · refresh · title Refresh Feed | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · volume_off · title Bisu Alarm Suara | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi volume_off. |
| button button · open_in_full · title Buka Layar Penuh | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · chat WhatsApp · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · call Direct VoIP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · sms SMS Ping · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Kunci Kontak (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · fact_check Terbitkan Berita Acara (BAP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · check Tandai False Alarm / Terverifikasi Legal | Terbatas | Belum ada insiden untuk diproses. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Refresh Feed
- Bisu Alarm Suara
- Buka Layar Penuh
- GPS Phone Online
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi.
- Senin: 4 Kasus
- Selasa: 5 Kasus
- Rabu: 3 Kasus
- Kamis: 7 Kasus
- Jumat: 4 Kasus
- Sabtu: 2 Kasus
- Hari Ini: 5 Kasus
- Ahmad Fauzi (Lead)
- Rian Gunawan
- Dewi Lestari
- Siti Rahma

### reports

22 kelompok kontrol. Nonaktif 5, Respons UI teramati 5, Aksi belum tersedia 10, Belum terverifikasi 1, Input bisa diubah 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 1. Tombol aktif di bawah 24 px: 6. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 64.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, label 11–11, h2 18–18, th 11–11, td 12–12.

[Desktop](screenshots/reports-desktop.png) · [Mobile](screenshots/reports-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · schedule_send Jadwalkan Laporan (Email/WA) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · file_download Ekspor Data (.xlsx) | Respons UI teramati | Dialog terbuka: Tambah Laporan armada; penyimpanan belum diklaim. · Isi dialog SELECT / report_type / positions trips fuel events, SELECT / format / csv xlsx pdf, INPUT / from / from, INPUT / to / to, INPUT / vehicle_id / vehicle id, BUTTON / Batal, BUTTON / Simpan |
| button button · arrow_drop_down | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi arrow_drop_down. |
| select · select ×2 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · tune Filter Lanjut | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · refresh · title Muat Ulang Data | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · local_gas_station Rekapitulasi Konsumsi BBM N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Rekapitulasi Konsumsi BBM N/A. |
| button button · gpp_maybe Audit Anomali & Kasus Pencurian Kasus | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Audit Anomali & Kasus Pencurian Kasus. |
| button button · route Utilitas & Jarak Tempuh (Mileage) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Utilitas & Jarak Tempuh (Mileage). |
| button button · speed Skor Pengemudi & Safety (Eco-Driving) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Skor Pengemudi & Safety (Eco-Driving). |
| button button · tune Riwayat Kalibrasi & Akurasi Sensor | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Lihat Rincian Rute → | Respons UI teramati | Navigasi ke #playback/1 |
| button button · Download Split Cost | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Download Split Cost. |
| input text · Filter plat nomor, nama pengemudi, atau nomor lambung unit... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · filter_list Status BBM | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Status BBM. |
| button button · print Format Pertamina/SKK · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · download Unduh Semua Paket (ZIP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Generate arrow_forward ×4 | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Generate. |
| button button · send Kirim Sekarang ke Email Direksi | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Kirim Sekarang ke Email Direksi. |
| button button · Konfigurasi Webhook & Jadwal | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Konfigurasi Webhook & Jadwal. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Muat Ulang Data

### device-management

22 kelompok kontrol. Aksi belum tersedia 2, Nonaktif 5, Respons UI teramati 4, Input bisa diubah 7, Belum terverifikasi 4.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 6. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 90.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, th 11–11, td 13–13, small 10–10, h2 15–15, label 11–16.

[Desktop](screenshots/device-management-desktop.png) · [Mobile](screenshots/device-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · upload_file Batch Upload (CSV/Excel) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Batch Upload (CSV/Excel). |
| button button · terminal Kirim Command SMS/OTA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · add_circle + Tambah Perangkat Baru | Respons UI teramati | Dialog terbuka: Tambah Device Management; penyimpanan belum diklaim. · Isi dialog INPUT / imei / imei, INPUT / model_id / model id, INPUT / firmware_version / firmware version, INPUT / protocol_version / protocol version, SELECT / is_active / true false, BUTTON / Batal, BUTTON / Simpan |
| button button · Push Update · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input search · Cari IMEI, Serial Number, Nomor SIM, atau Plat Nomor... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · tune Filter Lanjut · title Kustomisasi kolom & filter mendalam | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · refresh · title Muat ulang telemetry device | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input checkbox · input | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input checkbox · Pilih record 10 | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Detail ×3 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button · Hubungkan kendaraan ×3 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit ×3 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| input checkbox · Pilih record 9 | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input checkbox · Pilih record 8 | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Jalankan Paksa Batch OTA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input text · input | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input checkbox · N/A | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · send Eksekusi Perintah | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Eksekusi Perintah. |
| a · Unduh Audit Log Gateway | Respons UI teramati | Navigasi ke |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Kustomisasi kolom & filter mendalam
- Muat ulang telemetry device

### sensor-management

15 kelompok kontrol. Respons UI teramati 3, Aksi belum tersedia 4, Belum terverifikasi 3, Tidak ada efek teramati 1, Input bisa diubah 1, Nonaktif 3.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 10. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 73.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, label 11–11, th 11–11, td 13–13, small 10–10, h2 18–18.

[Desktop](screenshots/sensor-management-desktop.png) · [Mobile](screenshots/sensor-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · science Diagnostik Raw Signal | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · upload_file Impor Tabel Kalibrasi (CSV) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Impor Tabel Kalibrasi (CSV). |
| button button · add_circle + Pasang Sensor Baru | Respons UI teramati | Dialog terbuka: Tambah Sensor Management; penyimpanan belum diklaim. · Isi dialog INPUT / sensor_identifier / sensor identifier, INPUT / mac / mac, INPUT / model / model, SELECT / is_active / true false, BUTTON / Batal, BUTTON / Simpan |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · download Unduh Rekap Sensor | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · flaky Pengujian Deviasi Massal | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pengujian Deviasi Massal. |
| input text · Filter ID Sensor / Plat Kendaraan... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · sync · title Reload Live Telemetry | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi sync. |
| button · Detail ×3 | Respons UI teramati | Dialog terbuka: Kalibrasi fuel · Empty / Full; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, INPUT / capacity_l / Kapasitas tangki (liter), SELECT / quantity / Volume raw (nilai sensor) Fuel height raw (nilai sensor), BUTTON / Perbarui nilai, BUTTON / Set Empty, BUTTON / Set Full, BUTTON / Simpan |
| button · Kalibrasi ×3 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit ×3 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · download Sertifikat Metrologi (PDF) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Edit Formula Tangki chevron_right | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Edit Formula Tangki. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- -85 dBm
- -78 dBm
- -72 dBm
- -68 dBm
- -65 dBm
- -61 dBm
- Reload Live Telemetry
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.

### user-management

13 kelompok kontrol. Respons UI teramati 5, Input bisa diubah 1, Aksi belum tersedia 3, Belum terverifikasi 2, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 2. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 57.
Ukuran font desktop (px) · h1 24–24, p 13–14, button 11–16, th 11–11, td 12–13, small 10–10, h2 15–15.

[Desktop](screenshots/user-management-desktop.png) · [Mobile](screenshots/user-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · history Audit Log Aktivitas | Respons UI teramati | Navigasi ke #audit |
| button button · shield_person Kelola Peran & Izin | Respons UI teramati | Dialog terbuka: Role Permission Matrix; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · person_add + Tambah Pengguna Baru | Respons UI teramati | Dialog terbuka: Tambah User Management; penyimpanan belum diklaim. · Isi dialog INPUT / username / username, INPUT / full_name / full name, INPUT / email / email, INPUT / password / password, SELECT / role / admin operator viewer, SELECT / is_active / true false, BUTTON / Batal, BUTTON / Simpan |
| input search · Cari nama lengkap, email, NIK karyawan, atau nomor telepon... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×3 · opsi Semua peran, admin, operator, viewer | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · Terapkan Filter | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Terapkan Filter. |
| button button · restart_alt · title Reset Filter | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi restart_alt. |
| button · Detail | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Ubah Skema Hak Akses → | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Ubah Skema Hak Akses →. |
| a · Buka Konsol Audit Lengkap (SIEM Export) open_in_new | Respons UI teramati | Navigasi ke |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- 32 Dispatcher
- 8 Managers
- 5 IoT Engineers
- 3 Superadmins
- Reset Filter

### notifications

21 kelompok kontrol. Respons UI teramati 6, Aksi belum tersedia 5, Input bisa diubah 2, Belum terverifikasi 1, Tidak ada efek teramati 1, Nonaktif 5, Terbatas 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 55.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16.

[Desktop](screenshots/notifications-desktop.png) · [Mobile](screenshots/notifications-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · tune Konfigurasi Aturan Eskalasi | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · file_download Unduh Rekap Insiden Jam (PDF/Excel) | Respons UI teramati | Dialog terbuka: Tambah Laporan armada; penyimpanan belum diklaim. · Isi dialog SELECT / report_type / positions trips fuel events, SELECT / format / csv xlsx pdf, INPUT / from / from, INPUT / to / to, INPUT / vehicle_id / vehicle id, BUTTON / Batal, BUTTON / Simpan |
| button button · add_alert + Buat Tiket Manual / Lapor Insiden | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi + Buat Tiket Manual / Lapor Insiden. |
| button button · Semua Alert N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Kritis / Butuh Respon N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · local_gas_station Pencurian & Fluktuasi BBM N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pencurian & Fluktuasi BBM N/A. |
| button button · explore Geofence & Rute N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Geofence & Rute N/A. |
| button button · cable Sensor Tamper & GPS N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Sensor Tamper & GPS N/A. |
| button button · task_alt Terselesaikan Hari Ini N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input search · Cari no tiket, plat nomor, lokasi, atau jenis insiden... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×2 · opsi Semua status, MOVING, STOPPED, STALE, valid, uncalibrated, open, acknowledged | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · refresh · title Refresh Feed | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Record 1 N/A N/A | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · volume_off · title Bisu Alarm Suara | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi volume_off. |
| button button · open_in_full · title Buka Layar Penuh | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · chat WhatsApp · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · call Direct VoIP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · sms SMS Ping · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Kunci Kontak (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · fact_check Terbitkan Berita Acara (BAP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · check Tandai False Alarm / Terverifikasi Legal | Terbatas | Belum ada insiden untuk diproses. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Refresh Feed
- Bisu Alarm Suara
- Buka Layar Penuh
- GPS Phone Online
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi.
- Senin: 4 Kasus
- Selasa: 5 Kasus
- Rabu: 3 Kasus
- Kamis: 7 Kasus
- Jumat: 4 Kasus
- Sabtu: 2 Kasus
- Hari Ini: 5 Kasus
- Ahmad Fauzi (Lead)
- Rian Gunawan
- Dewi Lestari
- Siti Rahma

### audit

12 kelompok kontrol. Tidak ada efek teramati 1, Respons UI teramati 4, Terbatas 1, Input bisa diubah 1, Aksi belum tersedia 3, Belum terverifikasi 1, Nonaktif 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 50. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 252.
Ukuran font desktop (px) · h1 24–24, p 13–14, button 11–16, th 11–11, td 12–13, small 10–10, h2 15–15.

[Desktop](screenshots/audit-desktop.png) · [Mobile](screenshots/audit-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · history Audit Log Aktivitas | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · shield_person Kelola Peran & Izin | Respons UI teramati | Dialog terbuka: Role Permission Matrix; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · person_add + Tambah Pengguna Baru | Terbatas | Tidak memiliki akses membuat record pada halaman ini. |
| input search · Cari nama lengkap, email, NIK karyawan, atau nomor telepon... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×3 · opsi Semua peran, admin, operator, viewer | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · Terapkan Filter | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Terapkan Filter. |
| button button · restart_alt · title Reset Filter | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi restart_alt. |
| button · Detail ×50 | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Ubah Skema Hak Akses → | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Ubah Skema Hak Akses →. |
| a · Buka Konsol Audit Lengkap (SIEM Export) open_in_new | Respons UI teramati | Navigasi ke |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- 32 Dispatcher
- 8 Managers
- 5 IoT Engineers
- 3 Superadmins
- Reset Filter

### models

19 kelompok kontrol. Aksi belum tersedia 2, Nonaktif 5, Respons UI teramati 4, Input bisa diubah 5, Belum terverifikasi 3.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 2. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 73.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, th 11–11, td 13–13, small 10–10, h2 15–15, label 11–16.

[Desktop](screenshots/models-desktop.png) · [Mobile](screenshots/models-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · upload_file Batch Upload (CSV/Excel) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Batch Upload (CSV/Excel). |
| button button · terminal Kirim Command SMS/OTA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · add_circle + Tambah Perangkat Baru | Respons UI teramati | Dialog terbuka: Tambah Device Models; penyimpanan belum diklaim. · Isi dialog INPUT / manufacturer / manufacturer, INPUT / model / model, INPUT / protocol_family / protocol family, BUTTON / Batal, BUTTON / Simpan |
| button button · Push Update · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input search · Cari IMEI, Serial Number, Nomor SIM, atau Plat Nomor... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · tune Filter Lanjut · title Kustomisasi kolom & filter mendalam | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · refresh · title Muat ulang telemetry device | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input checkbox · input | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input checkbox · Pilih record 1 | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Detail | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Jalankan Paksa Batch OTA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input text · input | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input checkbox · N/A | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · send Eksekusi Perintah | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Eksekusi Perintah. |
| a · Unduh Audit Log Gateway | Respons UI teramati | Navigasi ke |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Kustomisasi kolom & filter mendalam
- Muat ulang telemetry device

### tanks

14 kelompok kontrol. Respons UI teramati 4, Aksi belum tersedia 4, Belum terverifikasi 1, Tidak ada efek teramati 1, Input bisa diubah 1, Nonaktif 3.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 11. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 80.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, label 11–11, th 11–11, td 13–13, small 10–10, h2 18–18.

[Desktop](screenshots/tanks-desktop.png) · [Mobile](screenshots/tanks-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · science Diagnostik Raw Signal | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · upload_file Impor Tabel Kalibrasi (CSV) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Impor Tabel Kalibrasi (CSV). |
| button button · add_circle + Pasang Sensor Baru | Respons UI teramati | Dialog terbuka: Tambah Fuel Tank Profiles; penyimpanan belum diklaim. · Isi dialog INPUT / vehicle_id / vehicle id, INPUT / name / name, INPUT / capacity_l / capacity l, INPUT / revision / revision, INPUT / effective_from / effective from, INPUT / effective_to / effective to, BUTTON / Batal, BUTTON / Simpan |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · download Unduh Rekap Sensor | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · flaky Pengujian Deviasi Massal | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pengujian Deviasi Massal. |
| input text · Filter ID Sensor / Plat Kendaraan... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · sync · title Reload Live Telemetry | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi sync. |
| button · Detail ×5 | Respons UI teramati | Dialog terbuka: tank-profiles #5; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button · Edit ×5 | Respons UI teramati | Dialog terbuka: Edit Fuel Tank Profiles; penyimpanan belum diklaim. · Isi dialog INPUT / vehicle_id / vehicle id, INPUT / name / name, INPUT / capacity_l / capacity l, INPUT / revision / revision, INPUT / effective_from / effective from, INPUT / effective_to / effective to, BUTTON / Batal, BUTTON / Simpan |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · download Sertifikat Metrologi (PDF) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Edit Formula Tangki chevron_right | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Edit Formula Tangki. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- -85 dBm
- -78 dBm
- -72 dBm
- -68 dBm
- -65 dBm
- -61 dBm
- Reload Live Telemetry
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.

### calibrations

29 kelompok kontrol. Input bisa diubah 1, Aksi belum tersedia 4, Respons UI teramati 13, Tidak ada efek teramati 2, Nonaktif 8, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 3. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 173.
Ukuran font desktop (px) · h1 18–18, p 11–13, button 11–16, h3 15–15, th 10–11, td 13–13, small 10–10.4, h2 15–15.

[Desktop](screenshots/calibrations-desktop.png) · [Mobile](screenshots/calibrations-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input text · Filter Plate / Chassis / Driver... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · local_shipping All Chassis | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi All Chassis. |
| button button · Wingbox | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Wingbox. |
| button button · Tanker | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Tanker. |
| button button · All Status | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Moving | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · file_download Export Log | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · add_circle Register Unit | Respons UI teramati | Dialog terbuka: Tambah Sensor Calibration; penyimpanan belum diklaim. · Isi dialog INPUT / fuel_sensor_id / fuel sensor id, INPUT / tank_profile_id / tank profile id, INPUT / revision / revision, SELECT / media / fuel_tank water_test, SELECT / input_quantity / fuel_height_raw volume_raw, INPUT / input_unit / input unit, SELECT / output_unit / liters percent, BUTTON / Batal, BUTTON / Simpan |
| button button · send_time_extension Send Poll Command · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · tune Calibrate BLE · title Re-sync BLE CFS300B raw capacitance profile | Respons UI teramati | Navigasi ke #vehicles/1/calibration |
| button button · share Share Dispatch Link | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Share Dispatch Link. |
| button button · more_vert | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · space_dashboard Telematics Overview | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| button button · history Trip History km • | Respons UI teramati | Navigasi ke #vehicles/1/trips |
| button button · local_gas_station Fuel & BLE CFS300B L | Respons UI teramati | Navigasi ke #vehicles/1/fuel |
| button button · sensors Sensor Calibration -Point Table • Kalman K= · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · notifications_active Telemetry Events N/A | Respons UI teramati | Navigasi ke #vehicles/1/events |
| button button · developer_board Queclink Device Info | Respons UI teramati | Navigasi ke #vehicles/1/device |
| button button · file_download Export CSV | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · picture_as_pdf Sertifikat PDF · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · sync Re-sync OTA Calibration Table · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · restart_alt Simulate Zero Frequency (Empty Reset) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · chevron_left | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · N/A ×3 | Respons UI teramati | Navigasi ke #vehicles/1/events |
| button button · chevron_right | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Inspect ×3 | Respons UI teramati | Navigasi ke #vehicles/3/overview |
| button · Edit ×3 | Respons UI teramati | Dialog terbuka: Edit Sensor Calibration; penyimpanan belum diklaim. · Isi dialog INPUT / fuel_sensor_id / fuel sensor id, INPUT / tank_profile_id / tank profile id, INPUT / revision / revision, SELECT / media / fuel_tank water_test, SELECT / input_quantity / fuel_height_raw volume_raw, INPUT / input_unit / input unit, SELECT / output_unit / liters percent, BUTTON / Batal, BUTTON / Simpan |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Re-sync BLE CFS300B raw capacitance profile
- Wiring harness is intentionally unwired in this hardware batch
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya

### event-types

26 kelompok kontrol. Respons UI teramati 7, Aksi belum tersedia 6, Input bisa diubah 2, Nonaktif 8, Terbatas 2, Tidak ada efek teramati 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 84.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16.

[Desktop](screenshots/event-types-desktop.png) · [Mobile](screenshots/event-types-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · tune Konfigurasi Aturan Eskalasi | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · file_download Unduh Rekap Insiden Jam (PDF/Excel) | Respons UI teramati | Dialog terbuka: Tambah Laporan armada; penyimpanan belum diklaim. · Isi dialog SELECT / report_type / positions trips fuel events, SELECT / format / csv xlsx pdf, INPUT / from / from, INPUT / to / to, INPUT / vehicle_id / vehicle id, BUTTON / Batal, BUTTON / Simpan |
| button button · add_alert + Buat Tiket Manual / Lapor Insiden | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi + Buat Tiket Manual / Lapor Insiden. |
| button button · Semua Alert N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Kritis / Butuh Respon N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · local_gas_station Pencurian & Fluktuasi BBM N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pencurian & Fluktuasi BBM N/A. |
| button button · explore Geofence & Rute N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Geofence & Rute N/A. |
| button button · cable Sensor Tamper & GPS N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Sensor Tamper & GPS N/A. |
| button button · task_alt Terselesaikan Hari Ini N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input search · Cari no tiket, plat nomor, lokasi, atau jenis insiden... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×2 · opsi Semua status, MOVING, STOPPED, STALE, valid, uncalibrated, open, acknowledged | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · refresh · title Refresh Feed | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · phone_in_talk Telepon Supir · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Matikan Mesin (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · Triage Segera | Terbatas | Belum ada insiden untuk diproses. |
| button button · Lihat Log Diagnostik Sensor | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · Audit Koordinat Rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Audit Koordinat Rute. |
| button button · Request Foto Struk WA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · volume_off · title Bisu Alarm Suara | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi volume_off. |
| button button · open_in_full · title Buka Layar Penuh | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · chat WhatsApp · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · call Direct VoIP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · sms SMS Ping · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Kunci Kontak (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · fact_check Terbitkan Berita Acara (BAP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · check Tandai False Alarm / Terverifikasi Legal | Terbatas | Belum ada insiden untuk diproses. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Refresh Feed
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi.
- Bisu Alarm Suara
- Buka Layar Penuh
- GPS Phone Online
- Senin: 4 Kasus
- Selasa: 5 Kasus
- Rabu: 3 Kasus
- Kamis: 7 Kasus
- Jumat: 4 Kasus
- Sabtu: 2 Kasus
- Hari Ini: 5 Kasus
- Ahmad Fauzi (Lead)
- Rian Gunawan
- Dewi Lestari
- Siti Rahma

### event-severities

26 kelompok kontrol. Respons UI teramati 7, Aksi belum tersedia 6, Input bisa diubah 2, Nonaktif 8, Terbatas 2, Tidak ada efek teramati 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 84.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16.

[Desktop](screenshots/event-severities-desktop.png) · [Mobile](screenshots/event-severities-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · tune Konfigurasi Aturan Eskalasi | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · file_download Unduh Rekap Insiden Jam (PDF/Excel) | Respons UI teramati | Dialog terbuka: Tambah Laporan armada; penyimpanan belum diklaim. · Isi dialog SELECT / report_type / positions trips fuel events, SELECT / format / csv xlsx pdf, INPUT / from / from, INPUT / to / to, INPUT / vehicle_id / vehicle id, BUTTON / Batal, BUTTON / Simpan |
| button button · add_alert + Buat Tiket Manual / Lapor Insiden | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi + Buat Tiket Manual / Lapor Insiden. |
| button button · Semua Alert N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Kritis / Butuh Respon N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · local_gas_station Pencurian & Fluktuasi BBM N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pencurian & Fluktuasi BBM N/A. |
| button button · explore Geofence & Rute N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Geofence & Rute N/A. |
| button button · cable Sensor Tamper & GPS N/A | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Sensor Tamper & GPS N/A. |
| button button · task_alt Terselesaikan Hari Ini N/A | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input search · Cari no tiket, plat nomor, lokasi, atau jenis insiden... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · select ×2 · opsi Semua status, MOVING, STOPPED, STALE, valid, uncalibrated, open, acknowledged | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · refresh · title Refresh Feed | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · phone_in_talk Telepon Supir · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Matikan Mesin (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · Triage Segera | Terbatas | Belum ada insiden untuk diproses. |
| button button · Lihat Log Diagnostik Sensor | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · Audit Koordinat Rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Audit Koordinat Rute. |
| button button · Request Foto Struk WA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · volume_off · title Bisu Alarm Suara | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi volume_off. |
| button button · open_in_full · title Buka Layar Penuh | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · chat WhatsApp · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · call Direct VoIP · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · sms SMS Ping · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · power_settings_new Kunci Kontak (Immobilizer) · title Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. | Nonaktif | Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi. |
| button button · fact_check Terbitkan Berita Acara (BAP) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · check Tandai False Alarm / Terverifikasi Legal | Terbatas | Belum ada insiden untuk diproses. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Refresh Feed
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Tidak tersedia: physical ignition wiring NOT_INSTALLED; kontrol mesin tidak dikonfigurasi.
- Bisu Alarm Suara
- Buka Layar Penuh
- GPS Phone Online
- Senin: 4 Kasus
- Selasa: 5 Kasus
- Rabu: 3 Kasus
- Kamis: 7 Kasus
- Jumat: 4 Kasus
- Sabtu: 2 Kasus
- Hari Ini: 5 Kasus
- Ahmad Fauzi (Lead)
- Rian Gunawan
- Dewi Lestari
- Siti Rahma

### geofence-assignments

29 kelompok kontrol. Aksi belum tersedia 14, Terbatas 3, Respons UI teramati 4, Input bisa diubah 1, Belum terverifikasi 2, Tidak ada efek teramati 3, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 4. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 91.
Ukuran font desktop (px) · h1 18–18, button 11–16, h3 15–15, p 12–12, th 11–11, td 12–12.

[Desktop](screenshots/geofence-assignments-desktop.png) · [Mobile](screenshots/geofence-assignments-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · upload_file Import KML / GeoJSON | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Import KML / GeoJSON. |
| button button · download Ekspor Log Kunjungan | Terbatas | Belum ada data untuk diekspor. |
| button button · add_circle + Buat Geofence Baru | Respons UI teramati | Dialog terbuka: Tambah Geofence Assignments; penyimpanan belum diklaim. · Isi dialog INPUT / geofence_id / geofence id, INPUT / vehicle_id / vehicle id, INPUT / starts_at / starts at, INPUT / ends_at / ends at, BUTTON / Batal, BUTTON / Simpan |
| button button · Semua | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Pool & Depo | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pool & Depo. |
| button button · Customer Warehouse | Respons UI teramati | Dialog terbuka: Rentang waktu · Asia/Jakarta; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, INPUT / from / Dari, INPUT / to / Sampai, BUTTON / Terapkan |
| button button · SPBU / Refuel Point | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi SPBU / Refuel Point. |
| button button · Restricted / Blackzone | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Restricted / Blackzone. |
| input text · Cari geofence, kode, atau POI... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · splitscreen · title Split Editor Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi splitscreen. |
| button button · map · title Full Map Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi map. |
| button button · table_rows · title List Registry Mode | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi table_rows. |
| button button · Filter Aturan tune | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · polyline Poligon (Aktif) · title Gambar Poligon Kustom | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Poligon (Aktif). |
| button button · radio_button_unchecked Radius · title Gambar Radius Lingkaran | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Radius. |
| button button · route Koridor · title Koridor Buffer Rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Koridor. |
| button button · edit · title Edit Titik Vertex | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi edit. |
| button button · straighten · title Penggaris / Ukur Jarak | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi straighten. |
| button button · delete · title Hapus Gambar | Belum terverifikasi | Aksi tulis/otorisasi/perintah fisik tidak dieksekusi di live. |
| button button · Vector Fleet | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Satelit | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| button button · Lalu Lintas | Terbatas | Layer ini memerlukan sumber peta/traffic yang belum dikonfigurasi. Peta GPS aktual tetap tersedia. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button button · edit_note Ubah Batas Vertex | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Ubah Batas Vertex. |
| button button · notifications_active Konfigurasi Notifikasi | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Konfigurasi Notifikasi. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Split Editor Mode
- Full Map Mode
- List Registry Mode
- Gambar Poligon Kustom
- Gambar Radius Lingkaran
- Koridor Buffer Rute
- Edit Titik Vertex
- Penggaris / Ukur Jarak
- Hapus Gambar
- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### sensor-telemetry

13 kelompok kontrol. Respons UI teramati 3, Aksi belum tersedia 4, Terbatas 1, Belum terverifikasi 1, Tidak ada efek teramati 1, Input bisa diubah 1, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 51. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 255.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, label 11–11, th 11–11, td 13–13, small 10–10, h2 18–18.

[Desktop](screenshots/sensor-telemetry-desktop.png) · [Mobile](screenshots/sensor-telemetry-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · science Diagnostik Raw Signal | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · upload_file Impor Tabel Kalibrasi (CSV) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Impor Tabel Kalibrasi (CSV). |
| button button · add_circle + Pasang Sensor Baru | Terbatas | Tidak memiliki akses membuat record pada halaman ini. |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · download Unduh Rekap Sensor | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · flaky Pengujian Deviasi Massal | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pengujian Deviasi Massal. |
| input text · Filter ID Sensor / Plat Kendaraan... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · sync · title Reload Live Telemetry | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi sync. |
| button · Detail ×50 | Respons UI teramati | Dialog terbuka: sensor-telemetry #3185; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · download Sertifikat Metrologi (PDF) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Edit Formula Tangki chevron_right | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Edit Formula Tangki. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- -85 dBm
- -78 dBm
- -72 dBm
- -68 dBm
- -65 dBm
- -61 dBm
- Reload Live Telemetry
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.

### events

12 kelompok kontrol. Respons UI teramati 2, Aksi belum tersedia 4, Terbatas 1, Belum terverifikasi 1, Tidak ada efek teramati 1, Input bisa diubah 1, Nonaktif 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 51. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 55.
Ukuran font desktop (px) · h1 24–24, p 11–13, button 11–16, label 11–11, th 11–11, td 13–13, small 10–10, h2 18–18.

[Desktop](screenshots/events-desktop.png) · [Mobile](screenshots/events-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · science Diagnostik Raw Signal | Respons UI teramati | Navigasi ke #raw-protocol-logs |
| button button · upload_file Impor Tabel Kalibrasi (CSV) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Impor Tabel Kalibrasi (CSV). |
| button button · add_circle + Pasang Sensor Baru | Terbatas | Tidak memiliki akses membuat record pada halaman ini. |
| select · select ×4 · opsi All / belum dikonfigurasi | Belum terverifikasi | Tidak ada opsi alternatif dengan data saat audit. |
| button button · download Unduh Rekap Sensor | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · flaky Pengujian Deviasi Massal | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Pengujian Deviasi Massal. |
| input text · Filter ID Sensor / Plat Kendaraan... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · sync · title Reload Live Telemetry | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi sync. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · download Sertifikat Metrologi (PDF) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Edit Formula Tangki chevron_right | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Edit Formula Tangki. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- -85 dBm
- -78 dBm
- -72 dBm
- -68 dBm
- -65 dBm
- -61 dBm
- Reload Live Telemetry
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.

### vehicles/3/overview

10 kelompok kontrol. Respons UI teramati 7, Tidak ada efek teramati 2, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 22.
Ukuran font desktop (px) · h1 24–24, p 13–13, h2 15–15.

[Desktop](screenshots/vehicles_3_overview-desktop.png) · [Mobile](screenshots/vehicles_3_overview-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · ← Daftar kendaraan | Respons UI teramati | Navigasi ke #vehicles |
| select · Pilih kendaraan · opsi Pilih kendaraan, B 9102 PTR, B 9101 PTR, B 1307 KNQ | Respons UI teramati | Navigasi ke #vehicles/2/overview |
| a · Telematics Overview | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Trip History | Respons UI teramati | Navigasi ke #vehicles/3/trips |
| a · Fuel & BLE | Respons UI teramati | Navigasi ke #vehicles/3/fuel |
| a · Kalibrasi | Respons UI teramati | Navigasi ke #vehicles/3/calibration |
| a · Device Info | Respons UI teramati | Navigasi ke #vehicles/3/device |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### vehicles/3/trips

14 kelompok kontrol. Respons UI teramati 3, Input bisa diubah 3, Tersembunyi 1, Aksi belum tersedia 4, Tidak ada efek teramati 2, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 22.
Ukuran font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, button 12–13, h2 16–16, th 12–12.

[Desktop](screenshots/vehicles_3_trips-desktop.png) · [Mobile](screenshots/vehicles_3_trips-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan · opsi Pilih kendaraan, B 9102 PTR · Hino 500 Jakarta–Cirebon, B 9101 PTR · Hino 500 Jakarta–Cirebon, B 1307 KNQ · DT AXOR | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input date · Dari (WIB) | Respons UI teramati | Perubahan input memicu permintaan data. |
| input date · Sampai (WIB) | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button submit · Tampilkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Lihat tanggal data terakhir | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| button button · Lihat seluruh rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Lihat seluruh rute. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ❚❚ Pause. |
| button · ↺ Ulang | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ↺ Ulang. |
| select · Kecepatan · opsi 60×, 300×, 600×, 1800× | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input range · Waktu playback GPS | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · ▶ Putar ×2 | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ▶ Putar. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### vehicles/3/fuel

28 kelompok kontrol. Respons UI teramati 13, Input bisa diubah 1, Aksi belum tersedia 4, Unduhan terverifikasi 1, Terbatas 1, Nonaktif 5, Tidak ada efek teramati 1, Belum terverifikasi 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 3. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 401.
Ukuran font desktop (px) · h1 18–18, p 11–13, button 11–16, h3 15–15, th 10–11, td 13–13, small 10–10.4, h2 15–15.

[Desktop](screenshots/vehicles_3_fuel-desktop.png) · [Mobile](screenshots/vehicles_3_fuel-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Pilih kendaraan · opsi Pilih kendaraan, B 9102 PTR, B 9101 PTR, B 1307 KNQ | Respons UI teramati | Navigasi ke #vehicles/2/fuel |
| input text · Filter Plate / Chassis / Driver... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · local_shipping All Chassis | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi All Chassis. |
| button button · Wingbox | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Wingbox. |
| button button · Tanker | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Tanker. |
| button button · All Status | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Moving | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · file_download Export Log | Unduhan terverifikasi | File diunduh; metadata dan ukuran dicatat. |
| button button · add_circle Register Unit | Terbatas | Tidak memiliki akses membuat record pada halaman ini. |
| button button · send_time_extension Send Poll Command · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · tune Calibrate BLE · title Re-sync BLE CFS300B raw capacitance profile | Respons UI teramati | Navigasi ke #vehicles/3/calibration |
| button button · share Share Dispatch Link | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Share Dispatch Link. |
| button button · more_vert | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · space_dashboard Telematics Overview | Respons UI teramati | Navigasi ke #vehicles/3/overview |
| button button · history Trip History km • | Respons UI teramati | Navigasi ke #vehicles/3/trips |
| button button · local_gas_station Fuel & BLE CFS300B L | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · sensors Sensor Calibration Kalman K= · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · notifications_active Telemetry Events N/A | Respons UI teramati | Navigasi ke #vehicles/3/events |
| button button · developer_board Queclink Device Info | Respons UI teramati | Navigasi ke #vehicles/3/device |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · send Trigger Instant BLE Poll · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · file_download Export CSV | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · tune Calibration Curve | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · chevron_left | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · N/A ×3 | Respons UI teramati | Navigasi ke #vehicles/3/events |
| button button · chevron_right | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Inspect ×3 | Respons UI teramati | Navigasi ke #vehicles/3/overview |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Re-sync BLE CFS300B raw capacitance profile
- Wiring harness is intentionally unwired in this hardware batch
- MAC: F3:1A:8B:20:91:04
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya

### vehicles/3/device

26 kelompok kontrol. Nonaktif 6, Respons UI teramati 12, Aksi belum tersedia 4, Tidak ada efek teramati 2, Input bisa diubah 2.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 1. Tombol aktif di bawah 24 px: 3. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 70.
Ukuran font desktop (px) · button 11–16, h3 15–15, p 13–13, th 11–11, td 13–13, small 10–10.

[Desktop](screenshots/vehicles_3_device-desktop.png) · [Mobile](screenshots/vehicles_3_device-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button button · sensors Send Poll · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · tune Calibrate BLE | Respons UI teramati | Navigasi ke #vehicles/3/calibration |
| button button · restart_alt Reboot OTA · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · share · title Share Dispatch Link | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi share. |
| button button · more_vert · title Advanced Diagnostic Options | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| a · Telematics Overview | Respons UI teramati | Navigasi ke |
| a · Trip History N/A | Respons UI teramati | Navigasi ke |
| a · Fuel & BLE CFS300B N/A | Respons UI teramati | Navigasi ke |
| a · Sensor Calibration N/A | Respons UI teramati | Navigasi ke |
| a · Device Info GV30CEU + CFS300B | Respons UI teramati | Navigasi ke |
| a · Telemetry Events N/A | Respons UI teramati | Navigasi ke |
| button button · Clear · title Clear Output | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · Poll (GTRTO) | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Poll (GTRTO). |
| button button · Buffer Status | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Buffer Status. |
| button button · BLE Health | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi BLE Health. |
| button button · Dump Config | Respons UI teramati | Dialog terbuka: Konfigurasi Alert Rules; penyimpanan belum diklaim. · Isi dialog BUTTON / ×, BUTTON / Tambah rule |
| button button · Soft Reset · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input text · Enter raw Queclink AT command... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · send Execute · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| input text · Filter plate or tracker... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · download Export CSV | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Detail | Respons UI teramati | Dialog terbuka: vehicles/3/devices #10; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button · Edit | Respons UI teramati | Dialog terbuka: Edit Device Management; penyimpanan belum diklaim. · Isi dialog INPUT / imei / imei, INPUT / model_id / model id, INPUT / firmware_version / firmware version, INPUT / protocol_version / protocol version, SELECT / is_active / true false, BUTTON / Batal, BUTTON / Simpan |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Pilih kendaraan · opsi Pilih kendaraan, B 9102 PTR, B 9101 PTR, B 1307 KNQ | Respons UI teramati | Navigasi ke #vehicles/2/device |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- GPS Doppler + 3-Axis MEMS Active (Physically unwired for alternator & battery safety)
- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Share Dispatch Link
- Advanced Diagnostic Options
- Clear Output
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya

### vehicles/3/calibration

30 kelompok kontrol. Respons UI teramati 13, Input bisa diubah 1, Aksi belum tersedia 4, Tidak ada efek teramati 3, Nonaktif 8, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 3. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 174.
Ukuran font desktop (px) · h1 18–18, p 11–13, button 11–16, h3 15–15, th 10–11, td 13–13, small 10–10.4, h2 15–15.

[Desktop](screenshots/vehicles_3_calibration-desktop.png) · [Mobile](screenshots/vehicles_3_calibration-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Pilih kendaraan · opsi Pilih kendaraan, B 9102 PTR, B 9101 PTR, B 1307 KNQ | Respons UI teramati | Navigasi ke #vehicles/2/calibration |
| input text · Filter Plate / Chassis / Driver... | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button button · local_shipping All Chassis | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi All Chassis. |
| button button · Wingbox | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Wingbox. |
| button button · Tanker | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Tanker. |
| button button · All Status | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Moving | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · file_download Export Log | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · add_circle Register Unit | Respons UI teramati | Navigasi ke #device-management |
| button button · send_time_extension Send Poll Command · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · tune Calibrate BLE · title Re-sync BLE CFS300B raw capacitance profile | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button button · share Share Dispatch Link | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Share Dispatch Link. |
| button button · more_vert | Respons UI teramati | Dialog terbuka: Informasi tampilan; penyimpanan belum diklaim. · Isi dialog BUTTON / × |
| button button · space_dashboard Telematics Overview | Respons UI teramati | Navigasi ke #vehicles/3/overview |
| button button · history Trip History km • | Respons UI teramati | Navigasi ke #vehicles/3/trips |
| button button · local_gas_station Fuel & BLE CFS300B L | Respons UI teramati | Navigasi ke #vehicles/3/fuel |
| button button · sensors Sensor Calibration -Point Table • Kalman K= · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · notifications_active Telemetry Events N/A | Respons UI teramati | Navigasi ke #vehicles/3/events |
| button button · developer_board Queclink Device Info | Respons UI teramati | Navigasi ke #vehicles/3/device |
| button button · file_download Export CSV | Belum terverifikasi | Kontrol berubah setelah render ulang; belum diuji. |
| button button · picture_as_pdf Sertifikat PDF · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · sync Re-sync OTA Calibration Table · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · restart_alt Simulate Zero Frequency (Empty Reset) · title Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. | Nonaktif | Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi. |
| button button · chevron_left | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button button · N/A ×3 | Respons UI teramati | Navigasi ke #vehicles/3/events |
| button button · chevron_right | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Inspect ×3 | Respons UI teramati | Navigasi ke #vehicles/3/overview |
| button · Edit ×3 | Respons UI teramati | Dialog terbuka: Edit Sensor Calibration; penyimpanan belum diklaim. · Isi dialog INPUT / fuel_sensor_id / fuel sensor id, INPUT / tank_profile_id / tank profile id, INPUT / revision / revision, SELECT / media / fuel_tank water_test, SELECT / input_quantity / fuel_height_raw volume_raw, INPUT / input_unit / input unit, SELECT / output_unit / liters percent, BUTTON / Batal, BUTTON / Simpan |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Tidak tersedia: aksi hardware/layanan ini belum memiliki konfigurasi dan dukungan protokol tervalidasi.
- Re-sync BLE CFS300B raw capacitance profile
- Wiring harness is intentionally unwired in this hardware batch
- Laporan belum tersedia · Status terakhir dipertahankan sampai event berikutnya
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya

### playback/3

14 kelompok kontrol. Respons UI teramati 3, Input bisa diubah 3, Tersembunyi 1, Aksi belum tersedia 4, Tidak ada efek teramati 2, Belum terverifikasi 1.
Mobile 390 px pada viewport 390 px. Tombol tanpa nama selain ikon yang terdeteksi: 0. Tombol aktif di bawah 24 px: 0. Elemen teks yang dihitung di bawah 12 px pada inventaris desktop: 22.
Ukuran font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, button 12–13, h2 16–16, th 12–12, td 12–12.

[Desktop](screenshots/playback_3-desktop.png) · [Mobile](screenshots/playback_3-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan · opsi Pilih kendaraan, B 9102 PTR · Hino 500 Jakarta–Cirebon, B 9101 PTR · Hino 500 Jakarta–Cirebon, B 1307 KNQ · DT AXOR | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input date · Dari (WIB) | Respons UI teramati | Perubahan input memicu permintaan data. |
| input date · Sampai (WIB) | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button submit · Tampilkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button button · Lihat tanggal data terakhir | Tersembunyi | Kontrol tidak terlihat pada keadaan awal; dicatat untuk cakupan kondisi. |
| button button · Lihat seluruh rute | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi Lihat seluruh rute. |
| a · + · title Zoom in | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − · title Zoom out | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet · title A JavaScript library for interactive maps | Belum terverifikasi | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ❚❚ Pause. |
| button · ↺ Ulang | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ↺ Ulang. |
| select · Kecepatan · opsi 60×, 300×, 600×, 1800× | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input range · Waktu playback GPS | Input bisa diubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · ▶ Putar ×2 | Aksi belum tersedia | Belum ada data atau konfigurasi tervalidasi untuk aksi ▶ Putar. |

Tooltip berbasis atribut title yang terlihat saat inventaris. Kemunculan setiap tooltip lewat hover belum terverifikasi.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

## Bukti audit

[Inventaris kontrol dan hasil browser](pages.json) · [Metrik visual](visual.json) · [Skenario terarah](deep-checks.json) · [Pemeriksaan role lokal](roles.json) · [Tes Go](go-tests.log) · [Tes DOM](dom-tests.log)
