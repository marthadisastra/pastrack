# Inspect kendaraan — Telematics Overview

Penyebab N/A: template detail belum mengikat field telemetri pada kartu; sejumlah teks contoh masih tersisa. Ini terpisah dari registry Vehicles.

Perbaikan source lokal:
- [x] Identitas kendaraan/GPS berdasarkan ID yang dipilih.
- [x] Speed dan heading dari posisi tersimpan, termasuk 0 km/jam yang valid.
- [x] Fuel liter/persen dan validity dari kalibrasi tersimpan.
- [x] GPS fix, satelit dan HDOP dari data posisi.
- [x] Heartbeat gateway dan status BLE/freshness.
- [x] Waktu pengukuran/penerimaan dan data lama dibedakan.
- [x] Label tab navigasi tetap jelas, bukan N/A.
- [x] Peta hanya untuk kendaraan yang dipilih.
- [x] ALL PASS, virtual ignition ON, No Theft, depot contoh dan percepatan contoh dihapus.
- [x] Binding kartu menerima SSE tanpa reload.
- [x] Tes DOM data terpilih + update speed melalui SSE lulus.
- [ ] QA visual browser desktop/mobile dan navigasi akhir: belum dilakukan; tool menolak karena Local browser use disabled in Work settings.

Belum deploy. Preview lokal port18092 menggunakan sumber database live yang sama, tanpa perubahan data bisnis dalam pekerjaan ini. Registry Vehicles yang sebelumnya selesai tidak berarti seluruh halaman detail telah selesai.
