# Audit fungsi PasTrack setelah pembersihan data uji

Chrome desktop 1366×768 dan mobile 390×844. Pemeriksaan live memakai akses admin dengan mutasi API ditahan. Kontrol yang berulang diperiksa per jenis. Respons UI berarti perubahan tampilan, dialog, navigasi atau permintaan data teramati.

**Cakupan** · 33 rute dan 314 kelompok kontrol.

| Halaman | Status | Bukti utama |
|---|---|---|
| fleet-dashboard | Teruji sebagian | Ringkasan, daftar dan kontrol diperiksa. Perubahan status dari perangkat belum diuji end-to-end. |
| vehicles | Teruji sebagian | Inventaris dan detail kendaraan tersedia. Perubahan data dan kontrol relay tidak dieksekusi di live. |
| live-tracking | Teruji sebagian | Peta dan daftar tersedia. Pergeseran marker dari kiriman baru belum teruji. |
| trip-history | Alur utama berjalan | Tanggal 3 November 2026 WIB memuat 642 titik GPS dan 1 perjalanan. Waktu playback bergerak. |
| geofences | Bermasalah | Kesalahan Leaflet `_leaflet_pos` tercatat saat transisi dari halaman sebelumnya. API geofence merespons dengan 0 entri. |
| fuel-monitoring | Teruji sebagian | Register pembacaan tersedia. Fungsi penyimpanan tidak berlaku pada register ini. |
| fuel-analytics | Terbatas data | API merespons, 0 ringkasan konsumsi. Angka konsumsi belum dapat diperiksa. |
| fuel-events | Teruji sebagian | GPS dan grafik BBM tersedia pada 3 November WIB. Ada 1.743 pembacaan, 1.690 pembacaan volume tervalidasi, dan 0 event. Detail, investigasi dan BAP belum teruji ulang dengan data operasional. |
| gps-quality | Teruji sebagian | API mengembalikan 50 baris pada sampel. |
| device-connectivity | Teruji sebagian | API mengembalikan 50 baris log pada sampel. |
| data-activity | Teruji sebagian | API mengembalikan 50 baris pada sampel. |
| raw-protocol-logs | Teruji sebagian | API mengembalikan 50 log pada sampel. |
| alert-center | Terbatas data | API merespons, 0 alert. Acknowledge belum teruji ulang. |
| reports | Teruji sebagian | Form pembuatan diperiksa di live. Daftar job kosong. Submit, proses dan unduh belum teruji ulang di live. |
| device-management | Teruji sebagian | Satu perangkat terdaftar. Perubahan relasi dan perangkat fisik belum diuji. |
| sensor-management | Teruji sebagian | Satu sensor terdaftar. Capture dan simpan kalibrasi belum diuji pada perangkat fisik. |
| user-management | Teruji sebagian | Daftar dan dialog diperiksa. Perubahan akun belum dijalankan di live. |
| notifications | Teruji sebagian | API mengembalikan 1 notifikasi. |
| audit | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| models | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| tanks | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| calibrations | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| event-types | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| event-severities | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| geofence-assignments | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| sensor-telemetry | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| events | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| vehicles/1/overview | Teruji sebagian | Tampilan dapat dibuka. Cakupan fungsi mengikuti data dan kontrol pada rincian. |
| vehicles/1/trips | Teruji sebagian | Rute tersedia. Uji gerak playback terarah dilakukan melalui Trip History. |
| vehicles/1/fuel | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| vehicles/1/device | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| vehicles/1/calibration | Teruji sebagian | Tabel dan kontrol yang tersedia diperiksa. Bukti per kontrol ada pada rincian. |
| playback/1 | Teruji sebagian | Rute tersedia. Uji gerak playback terarah dilakukan melalui Trip History. |

## Rincian kontrol

### fleet-dashboard

Overflow desktop False · mobile False · error —
Font desktop (px) · span 10–22, button 11–16, h1 24–24, p 12–14, input 12–12, a 12–22, h2 15–15, select 12–12, th 11–11, td 13–13, small 10–10.4

[Desktop](screenshots/fleet-dashboard-desktop.png) · [Mobile](screenshots/fleet-dashboard-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · CONFIG tune | Respons UI teramati | Dialog terbuka: Definisi data Fleet Dashboard; penyimpanan belum diklaim. |
| button · All Vehicles 1 | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Moving 0 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Stopped 0 | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Offline 1 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Active Alerts 0 | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input · Cari kendaraan di peta | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · crop_free Fit All | Respons UI teramati | Belum ada posisi GPS valid. |
| button · add | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · remove | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · + | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · − | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · more_vert | Respons UI teramati | Dialog terbuka: Definisi data Fleet Dashboard; penyimpanan belum diklaim. |
| div · Bergerak (&gt;3 km/jam) 0 · 0.0% | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Berhenti (≤3 km/jam) 0 · 0.0% | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · GPS offline 1 · 100.0% | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| div · Online · GPS lama 0 · 0.0% | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| div · Online · tanpa GPS fix 0 · 0.0% | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Kecepatan tidak tersedia 0 · 0.0% | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| div · Belum terpasang GPS 0 · 0.0% | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.14 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.13 WIB 0 km/h · -6.1334610, 106.8724130 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. |
| button · B 1307 KNQ · Pembacaan fuel CFS300B Diterima 3 Nov 2026, 18.58.14 WIB · Waktu ukur tidak tersedia 2,7 L · 2,25 % · valid | Respons UI teramati | Dialog terbuka: Pembacaan fuel CFS300B; penyimpanan belum diklaim. |
| button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.09 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.07 WIB 0 km/h · -6.1334810, 106.8724130 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. |
| button · B 1307 KNQ · Pembacaan fuel CFS300B Diterima 3 Nov 2026, 18.58.09 WIB · Waktu ukur tidak tersedia 2,7 L · 2,25 % · valid | Respons UI teramati | Dialog terbuka: Pembacaan fuel CFS300B; penyimpanan belum diklaim. |
| button · B 1307 KNQ · Laporan posisi GV30CEU Diterima 3 Nov 2026, 18.58.04 WIB · Waktu GPS/sensor 3 Nov 2026, 18.58.02 WIB 0 km/h · -6.1333210, 106.8723640 · valid | Respons UI teramati | Dialog terbuka: Laporan posisi GV30CEU; penyimpanan belum diklaim. |
| input · Filter plate / fleet... | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · Filter kode fleet | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · download Export CSV | Terbatas | Belum ada data untuk diekspor. |
| button · Reset filter | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Aktifkan Mesin | Respons UI teramati | GPS belum memiliki koneksi TCP aktif |
| button · Detail | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps
- Filter kendaraan berdasarkan status ini
- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · 3 Nov 2026, 18.03.56 WIB; ignition ditampilkan terpisah.

### vehicles

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 18–18, span 10–13, p 11–13, input 13–13, select 12–12, button 11–16, h2 15–15, th 11–11, td 13–13, small 10–10.4

[Desktop](screenshots/vehicles-desktop.png) · [Mobile](screenshots/vehicles-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| select · Filter fleet kendaraan | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · file_download Export Log | Unduhan terverifikasi | File diunduh; metadata dan ukuran dicatat. |
| button · add_circle Register Unit | Respons UI teramati | Dialog terbuka: Tambah Vehicles; penyimpanan belum diklaim. |
| button · Reset filter | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Refresh | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Kelola GPS | Respons UI teramati | Navigasi ke #device-management |
| button · Aktifkan Mesin | Respons UI teramati | GPS belum memiliki koneksi TCP aktif |
| button · Inspect | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| button · Edit | Respons UI teramati | Dialog terbuka: Edit Vehicles; penyimpanan belum diklaim. |
| button · Previous | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Next | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya
- Status remote dari output relay terakhir · 3 Nov 2026, 18.03.56 WIB; ignition ditampilkan terpisah.

### live-tracking

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, button 11–13, p 13–13, a 12–22, span 11–22, th 13–13, td 13–13

[Desktop](screenshots/live-tracking-desktop.png) · [Mobile](screenshots/live-tracking-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · Terapkan | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Muat ulang | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| a · + | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| a · − | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · Detail | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### trip-history

Overflow desktop False · mobile False · error —
Font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, select 11–11, input 11–13, button 12–13, h2 16–16, a 12–22, span 12–22, th 12–12, td 12–12

[Desktop](screenshots/trip-history-desktop.png) · [Mobile](screenshots/trip-history-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Dari (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Sampai (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Tampilkan | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Lihat tanggal data terakhir | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Lihat seluruh rute | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · + | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Waktu playback GPS | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### geofences

Overflow desktop False · mobile False · error [{'message': "Cannot read properties of undefined (reading '_leaflet_pos')", 'url': 'https://toko.azlaundry.my.id/#geofences'}]
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/geofences-desktop.png) · [Mobile](screenshots/geofences-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Geofences; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### fuel-monitoring

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, select 14–14, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/fuel-monitoring-desktop.png) · [Mobile](screenshots/fuel-monitoring-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### fuel-analytics

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, select 14–14, input 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/fuel-analytics-desktop.png) · [Mobile](screenshots/fuel-analytics-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### fuel-events

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 20–20, button 10–11, label 10–11, select 11–11, input 11–13, p 9–12, small 9–10, h2 14–14, a 12–22, span 10–22

[Desktop](screenshots/fuel-events-desktop.png) · [Mobile](screenshots/fuel-events-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · ↓ Unduh BAP Telematika | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ◎ Investigasi Anomali | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Start date (WIB) | Respons UI teramati | Perubahan input memicu permintaan data. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · ↻ Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| input · Geofence | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| a · + | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · −1m | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · +1m | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Waktu playback rute dan BBM | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| summary · Event & detail perjalanan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Semua Event (0) | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Anomali & Drop (0) | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Refueling (0) | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Idle (0) | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| summary · Aturan deteksi & dasar perhitungan | Belum teruji | locator.click: Timeout 1800ms exceeded. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### gps-quality

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/gps-quality-desktop.png) · [Mobile](screenshots/gps-quality-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### device-connectivity

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/device-connectivity-desktop.png) · [Mobile](screenshots/device-connectivity-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### data-activity

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/data-activity-desktop.png) · [Mobile](screenshots/data-activity-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### raw-protocol-logs

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/raw-protocol-logs-desktop.png) · [Mobile](screenshots/raw-protocol-logs-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Respons UI teramati | Dialog terbuka: Raw Message #39076; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### alert-center

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, select 14–14, input 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/alert-center-desktop.png) · [Mobile](screenshots/alert-center-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### reports

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/reports-desktop.png) · [Mobile](screenshots/reports-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Buat laporan | Respons UI teramati | Dialog terbuka: Tambah Reports; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### device-management

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/device-management-desktop.png) · [Mobile](screenshots/device-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Device Management; penyimpanan belum diklaim. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Hubungkan kendaraan | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### sensor-management

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/sensor-management-desktop.png) · [Mobile](screenshots/sensor-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Sensor Management; penyimpanan belum diklaim. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Kalibrasi | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### user-management

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/user-management-desktop.png) · [Mobile](screenshots/user-management-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah User Management; penyimpanan belum diklaim. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### notifications

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/notifications-desktop.png) · [Mobile](screenshots/notifications-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### audit

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/audit-desktop.png) · [Mobile](screenshots/audit-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### models

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/models-desktop.png) · [Mobile](screenshots/models-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Device Models; penyimpanan belum diklaim. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Edit | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### tanks

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, select 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/tanks-desktop.png) · [Mobile](screenshots/tanks-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Fuel Tank Profiles; penyimpanan belum diklaim. |
| button · Detail | Respons UI teramati | Dialog terbuka: Fuel Tank Profiles #3; penyimpanan belum diklaim. |
| button · Edit | Respons UI teramati | Dialog terbuka: Edit Fuel Tank Profiles; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### calibrations

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/calibrations-desktop.png) · [Mobile](screenshots/calibrations-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Sensor Calibration; penyimpanan belum diklaim. |
| button · Detail | Respons UI teramati | Dialog terbuka: Sensor Calibration #5; penyimpanan belum diklaim. |
| button · Edit | Respons UI teramati | Dialog terbuka: Edit Sensor Calibration; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### event-types

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/event-types-desktop.png) · [Mobile](screenshots/event-types-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Event Types; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### event-severities

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/event-severities-desktop.png) · [Mobile](screenshots/event-severities-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Event Severities; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### geofence-assignments

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, select 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/geofence-assignments-desktop.png) · [Mobile](screenshots/geofence-assignments-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Tambah | Respons UI teramati | Dialog terbuka: Tambah Geofence Assignments; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### sensor-telemetry

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/sensor-telemetry-desktop.png) · [Mobile](screenshots/sensor-telemetry-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### events

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 12–12, span 11–11

[Desktop](screenshots/events-desktop.png) · [Mobile](screenshots/events-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### vehicles/1/overview

Overflow desktop False · mobile False · error —
Font desktop (px) · a 12–22, h1 24–24, p 13–13, select 11–11, h2 15–15, span 11–22

[Desktop](screenshots/vehicles_1_overview-desktop.png) · [Mobile](screenshots/vehicles_1_overview-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · ← Daftar kendaraan | Respons UI teramati | Navigasi ke #vehicles |
| select · Pilih kendaraan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| a · Telematics Overview | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Trip History | Respons UI teramati | Navigasi ke #vehicles/1/trips |
| a · Fuel & BLE | Respons UI teramati | Navigasi ke #vehicles/1/fuel |
| a · Kalibrasi | Respons UI teramati | Navigasi ke #vehicles/1/calibration |
| a · Device Info | Respons UI teramati | Navigasi ke #vehicles/1/device |
| a · + | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · − | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Laporan GTIGL · 3 Nov 2026, 18.01.09 WIB · Status terakhir dipertahankan sampai event berikutnya
- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### vehicles/1/trips

Overflow desktop False · mobile False · error —
Font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, select 11–11, input 11–13, button 12–13, h2 16–16, a 12–22, span 12–22, th 12–12, td 12–12

[Desktop](screenshots/vehicles_1_trips-desktop.png) · [Mobile](screenshots/vehicles_1_trips-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Dari (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Sampai (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Tampilkan | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Lihat tanggal data terakhir | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Lihat seluruh rute | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · + | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Waktu playback GPS | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

### vehicles/1/fuel

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, a 14–14, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/vehicles_1_fuel-desktop.png) · [Mobile](screenshots/vehicles_1_fuel-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · Telematika | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| a · Perjalanan | Respons UI teramati | Navigasi ke #vehicles/1/trips |
| a · BBM | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Kalibrasi | Respons UI teramati | Navigasi ke #vehicles/1/calibration |
| a · Perangkat | Respons UI teramati | Navigasi ke #vehicles/1/device |
| input · Start date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · End date (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Belum teruji | locator.click: Timeout 1800ms exceeded. |

### vehicles/1/device

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, a 14–14, label 13–13, input 14–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/vehicles_1_device-desktop.png) · [Mobile](screenshots/vehicles_1_device-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · Telematika | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| a · Perjalanan | Respons UI teramati | Navigasi ke #vehicles/1/trips |
| a · BBM | Respons UI teramati | Navigasi ke #vehicles/1/fuel |
| a · Kalibrasi | Respons UI teramati | Navigasi ke #vehicles/1/calibration |
| a · Perangkat | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| input · Cari | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Belum teruji | Kontrol berubah setelah render ulang; belum diuji. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### vehicles/1/calibration

Overflow desktop False · mobile False · error —
Font desktop (px) · h1 24–24, a 13–14, button 11–13, p 13–13, th 13–13, td 13–13, span 11–11

[Desktop](screenshots/vehicles_1_calibration-desktop.png) · [Mobile](screenshots/vehicles_1_calibration-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| a · Telematika | Respons UI teramati | Navigasi ke #vehicles/1/overview |
| a · Perjalanan | Respons UI teramati | Navigasi ke #vehicles/1/trips |
| a · BBM | Respons UI teramati | Navigasi ke #vehicles/1/fuel |
| a · Kalibrasi | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · Perangkat | Respons UI teramati | Navigasi ke #vehicles/1/device |
| a · Kelola perangkat GPS | Respons UI teramati | Navigasi ke #device-management |
| button · Terapkan | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Muat ulang | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| button · Detail | Respons UI teramati | Dialog terbuka: Sensor Calibration #5; penyimpanan belum diklaim. |
| button · Sebelumnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · Berikutnya | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

### playback/1

Overflow desktop False · mobile False · error —
Font desktop (px) · small 10–11, h1 24–24, p 11–13, label 11–11, select 11–11, input 11–13, button 12–13, h2 16–16, a 12–22, span 12–22, th 12–12, td 12–12

[Desktop](screenshots/playback_1-desktop.png) · [Mobile](screenshots/playback_1-mobile.png)

| Kontrol | Status | Bukti |
|---|---|---|
| select · Kendaraan | Respons UI teramati | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Dari (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Sampai (WIB) | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| button · Tampilkan | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| button · Lihat tanggal data terakhir | Belum teruji | locator.click: Timeout 1800ms exceeded. |
| button · Lihat seluruh rute | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · + | Tidak ada efek teramati | Klik selesai tanpa perubahan tampilan, dialog, download atau request yang teramati. |
| a · − | Respons UI teramati | Keadaan tampilan atau permintaan data berubah. |
| a · Leaflet | Belum teruji | Link eksternal teridentifikasi; tujuan tidak diuji. |
| button · ▶ Play | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| button · ↺ Ulang | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |
| select · Kecepatan | Input berubah | Nilai input dapat diubah; efek data setelah Apply diperiksa terpisah. |
| input · Waktu playback GPS | Nonaktif | Disabled tanpa alasan yang dapat dibaca. |

Atribut tooltip yang terlihat pada inventaris. Kemunculan melalui hover belum diuji untuk seluruh kontrol.

- Zoom in
- Zoom out
- A JavaScript library for interactive maps

## Uji terarah tambahan

- **device-management Hubungkan kendaraan** · {'fields': 1}
- **sensor-management Kalibrasi** · {'fields': 2}
- **user-management Edit** · {'fields': 6}
- **reports Buat laporan** · {'fields': 5}
- **raw-protocol-logs pagination** · Berikutnya mengubah baris dan Sebelumnya mengembalikan baris awal
- **gps-quality pagination** · Berikutnya mengubah baris dan Sebelumnya mengembalikan baris awal
- **data-activity pagination** · Berikutnya mengubah baris dan Sebelumnya mengembalikan baris awal
- **device-connectivity pagination** · Berikutnya mengubah baris dan Sebelumnya mengembalikan baris awal
- **fuel-monitoring pagination** · Berikutnya mengubah baris dan Sebelumnya mengembalikan baris awal
- **Vehicles CSV** · {'name': 'vehicles-2026-11-04.csv', 'bytes': 507}
- **Fuel Events tabs** · {'selected': True}

## Bukti

[Inventaris dan hasil browser](pages.json) · [Uji data operasional](actual.json) · [Log browser](browser.log) · [Uji kontrol terarah](controls.json)
