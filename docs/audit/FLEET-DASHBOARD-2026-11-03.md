# Fleet Dashboard — hasil penyelesaian 3 November 2026

Backend dan UI sudah diterapkan ke live. Source lokal memakai DB dan stream authoritative yang sama. Tidak ada migrasi destruktif atau perubahan payload GPS/sensor.

## Checklist fungsi

- [x] Total armada dihitung dari kendaraan aktif, bukan angka mockup.
- [x] Offline dihitung dari heartbeat gateway; kendaraan tanpa GPS ditampilkan sebagai UNASSIGNED.
- [x] Status GPS dibedakan dari konektivitas gateway: STALE, GPS_NO_FIX, UNKNOWN_SPEED, MOVING, STOPPED.
- [x] Distribusi kategori bersifat eksklusif; jumlah semua kategori sama dengan total kendaraan.
- [x] Ambang MOVING >3 km/jam; STOPPED ≤3 km/jam. Tidak menyimpulkan parkir/loading dock atau kondisi mesin dari kecepatan.
- [x] KPI dan tabel memakai fungsi fleetRows yang sama, dengan relasi GPS–kendaraan aktif.
- [x] Baris kendaraan berisi model/identitas GPS, speed, fuel liter/persen, validity, status sensor, dan jumlah alert.
- [x] Fuel historis tetap ditampilkan dengan waktu penerimaan dan label data lama; data kosong tidak dibuat 0 liter.
- [x] Status BLE laporan lama menjadi STALE; last reported status tetap tersedia di API sebagai sensor_last_status.
- [x] Waktu GPS dan heartbeat gateway ditampilkan terpisah.
- [x] Ignition tetap NOT_INSTALLED.
- [x] Donut dan legenda status diperbarui per event, termasuk total dan persentase setiap kategori.
- [x] Aktivitas perangkat diambil dari event tersimpan, dengan raw code dan label unverified; tidak mengklaim raw code sebagai kejadian theft/refuel terverifikasi.
- [x] Activity feed dipatch per record melalui SSE, bukan reload seluruh panel.
- [x] Filter All/Moving/Stopped/Offline/Active Alerts diterapkan ke tabel.
- [x] Search plat/nama/model/identitas GPS bekerja; hasil kosong ditampilkan dengan benar.
- [x] Filter lokal tidak membuat ulang peta atau mengambil ulang data dari API.
- [x] Pencarian kendaraan di peta via Enter, fit all, zoom dan marker tersedia di source. Marker offline/STALE abu-abu.
- [x] Detail pada baris membuka #vehicles/{id}/overview; diuji lewat browser.
- [x] Ekspor CSV memakai data dashboard yang sedang difilter; file browser diunduh dan diperiksa isinya.
- [x] Event deadline satu kali memperbarui status freshness setelah batas 5 menit tanpa polling interval. Heartbeat SSE tetap hanya untuk koneksi/sesi.
- [x] Pemeriksaan browser pada viewport default menampilkan KPI 1 kendaraan / 1 offline, fuel 112,62 L / 93,85%, dan distribusi 100% offline.
- [x] Browser menguji filter offline, pencarian kosong, reset pencarian, CSV, dan navigasi detail.
- [x] Perapihan status bar untuk viewport sempit.

## Batas yang tetap ditampilkan secara jujur

- [ ] Kiriman GPS/sensor baru dari perangkat fisik belum terlihat selama QA. UI tidak mengubah tanggal data lama agar tampak real time.
- [ ] Traffic/satellite/corridor serta grup/wilayah belum memiliki sumber konfigurasi. Tombol dinonaktifkan dengan alasan; bukan dianggap fitur selesai.
- [ ] Uji visual seluruh ukuran layar, akses semua peran, uji beban banyak armada/subscriber, dan uji animasi dengan perangkat fisik bergerak masih perlu dilakukan.
- [ ] Aktivitas raw lama belum memiliki pemetaan makna semua event; tetap diberi label unverified.

## Skor audit

**90% (9 dari 10 kriteria inti audit)**: A–I terpenuhi; J belum lulus menyeluruh. Browser sudah menguji fungsi utama, tetapi belum seluruh ukuran layar/peran/aksi. Ini bukan 90% dari seluruh fitur layanan pihak ketiga atau mockup. Dashboard meningkat dari 80% pada audit sebelumnya menjadi 90%; tidak diberi 100% hanya karena endpoint merespons.

## Bukti

- `go test -race ./...` pada DB pengujian terisolasi: lulus.
- `go vet ./...`: lulus.
- 20 pengujian DOM: lulus, termasuk filter dashboard, nilai fuel, distribusi, dan feed SSE tanpa fetch tambahan.
- Unit test kategori fleet mencakup offline, GPS lama, tanpa fix, speed null, batas 3 km/jam, clock future, serta jumlah distribusi.
- CSV hasil browser: 1 kendaraan, fuel_liters=112.618, fuel_percent=93.85, physical_ignition_state=NOT_INSTALLED.

Laporan otomatis terakhir: `dashboard-tests-20261103.txt`. Audit ini tidak mengubah status halaman lain menjadi selesai.
