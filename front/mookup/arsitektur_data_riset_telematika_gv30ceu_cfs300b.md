# RISET & ARSITEKTUR TELEMATIKA KENDARAAN: QUECLINK GV30CEU & SENSOR BAHAN BAKAR CFS300B

## 1. Karakteristik & Fitur Perangkat Keras

### A. GPS Tracker Gateway: Queclink GV30CEU
Perangkat utama yang bertindak sebagai **satu-satunya gerbang telemetri (single ingestion gateway)** ke server:
- **Konektivitas Seluler & Jaringan:** LTE Cat 1 / EGPRS dengan fallback 2G. Mengirim paket ASCII/Hex terenkripsi melalui protokol `@Track` (+RESP:GTFRI, +RESP:GTGEO, +RESP:GTSVR).
- **Positioning GNSS:** Multi-konstelasi (GPS/GLONASS/Galileo/BDS) dengan akurasi sub-meter (HDOP < 1.0) dan pelaporan koordinat WGS84, kecepatan (speed over ground), heading/azimuth, serta timestamp presisi UTC.
- **Wireless Sensor Bridge (BLE 5.0):** Dilengkapi Bluetooth Low Energy yang memindai dan membaca paket broadcast secara langsung dari sensor probe nirkabel (seperti CFS300B) tanpa membutuhkan kabel RS485 atau CAN bus fisik.
- **Power & Backup Battery:** Operasional 8V–32V DC sistem aki kendaraan, dilengkapi cadangan baterai internal LiPo (3.7V - 4.2V) yang memicu alert bila aki utama dilepas.
- **Digital I/O & Relay:**
  - Input Digital 1 (DI1): Deteksi status kontak kunci mesin (**Ignition ACC ON / OFF**).
  - Output Digital (DO1 Relay): Trigger relay pemutus arus starter/pompa bensin jarak jauh (OTA immobilizer / fuel relay cut).
- **Internal 3-Axis MEMS Accelerometer:** Deteksi guncangan, pengereman mendadak (*Harsh Braking*), akselerasi tajam (*Harsh Acceleration*), dan tabrakan (*Crash/Impact*).

---

### B. Sensor Bahan Bakar: CFS300B (Wireless Capacitive Fuel Level Probe)
Sensor level bahan bakar berpresisi tinggi yang mengukur permitivitas dielektrik cairan:
- **Prinsip Pengukuran:** Kapasitansi kontinu (pF). Ketika bahan bakar mengisi batang probe aluminium, nilai kapasitansi berubah secara linier terhadap tinggi cairan.
- **Koneksi Nirkabel (BLE Broadcast):** Memancarkan data secara periodik langsung ke antena receiver GV30CEU melalui BLE Advertising Beacon (UUID, MAC address, Raw pF, Suhu cairan, CRC, dan tegangan baterai sensor internal 3.65V Li-SOCl2).
- **Kompensasi Slosh (Anti-Slosh):** Dilengkapi algoritma Kalman filtering atau dampening internal agar gejolak cairan saat kendaraan melaju tidak memicu false alarm.

---

## 2. Struktur Data Raw Telemetri (Dari GV30CEU ke Dashboard)
Karena semua raw signal hanya bersumber dari GV30CEU, satu paket laporan periodik (`+RESP:GTFRI`) membawa data gabungan:
1. **Identitas Hardware:** Protocol version, IMEI GV30CEU, Device Name/ID.
2. **Lokasi & Dinamika GNSS:** Longitude, Latitude, Altitudo, Kecepatan (km/h), Heading sudut arah (0–360°), GNSS Fix Status (2D/3D), Jumlah Satelit.
3. **Status Kelistrikan & Mesin:** ACC Ignition State (1 = ON, 0 = OFF), Main Battery Voltage (V), Backup LiPo Voltage (V).
4. **Data Payload BLE Sensor CFS300B:** MAC Address Probe, Kapasitansi (pF) / Estimasi Liter, Suhu Bahan Bakar (°C), Indikator Drift Zero-Point, Sinyal RSSI BLE (dBm), Status Baterai Probe.
5. **Mileage & Runtime:** Odometer virtual akumulatif, Engine Running Hours.

---

## 3. Desain Kebutuhan Data Dashboard (Sudut Pandang Pengguna)

### A. Pemantauan Lokasi (Vehicle Location & Fleet Tracking)
- **Live Vector Map:** Posisi real-time kendaraan dengan orientasi arah gerak (heading icon), jejak rekam jejak (breadcrumb trail), dan kecepatan saat ini.
- **Status Armada Instan:** Unit dikelompokkan menjadi **Bergerak (Moving)**, **Diam Mesin Nyala (Idling)**, **Parkir/Mesin Mati (Parked/ACC OFF)**, dan **Sinyal Hilang/Offline**.
- **Geofence Spatial Boundary:** Indikator apakah kendaraan berada di dalam zona kerja resmi (Depot, Warehouse, Terminal) atau di luar koridor yang diizinkan.

### B. Pemantauan Bahan Bakar (Fuel Monitoring)
- **Volume & Persentase Bahan Bakar:** Level bbm saat ini dalam liter (misal: 384 L / 400 L) dan persentase gauge (96%).
- **Fuel Run Chart (24 Jam / Real-time):** Grafik kontinu yang memetakan tinggi bbm terhadap waktu dan korelasi status mesin (ACC ON vs ACC OFF).
- **Deteksi Pengisian (Refueling Event):** Peningkatan volume drastis dalam waktu singkat saat kendaraan berhenti di depot/SPBU (+L, waktu, dan lokasi).
- **Deteksi Pengurasan / Pencurian (Fuel Siphon / Leakage Alert):** Penurunan mendadak volume bbm di luar batas normal, terutama saat status kunci kontak mesin mati (ACC OFF).

---

## 4. Matriks Alert yang Sesuai dengan Fitur Resmi GV30CEU

| Kategori Alert | Trigger / Kondisi Hardware GV30CEU & CFS300B | Dampak Operasional / Tindakan | Severity |
| :--- | :--- | :--- | :--- |
| **Fuel Siphon / Theft** | Penurunan level bbm CFS300B > ambang batas (misal: >5L dalam 3 menit) saat ACC = OFF | Indikasi pencurian bbm / kebocoran tangki saat parkir | **P1 Critical** |
| **Power Disconnect** | Tegangan Vin drop dari 24V/12V ke 0V, perangkat beralih ke baterai internal LiPo | Sabotase kabel kelistrikan aki kendaraan | **P1 Critical** |
| **Relay Cut Confirm** | Konfirmasi eksekusi digital output DO1 (Fuel cutoff relay aktif) | Kendaraan dimatikan paksa via remote command | **P1 Critical** |
| **Geofence Breach** | Firmware Queclink mendeteksi koordinat GNSS keluar/masuk zona poligon internal | Pelanggaran perimeter operasional | **P2 High** |
| **Over-Speed Alert** | Kecepatan GNSS melebihi batas batas ambang yang ditentukan (misal: >80 km/h) | Risiko keselamatan armada & pemborosan bahan bakar | **P2 High** |
| **Kinematic Shock** | Accelerometer internal merekam deselerasi > 0.45g (Harsh Braking) atau tabrakan | Potensi insiden kecelakaan di jalan | **P2 High** |
| **BLE Probe Loss / Weak RSSI** | Paket broadcast CFS300B tidak diterima GV30CEU selama > 3 menit atau RSSI < -85 dBm | Gangguan sinyal RF nirkabel / baterai probe habis | **P3 Warning** |
| **Low Probe Battery** | Baterai internal CFS300B drop di bawah 3.0V (Li-SOCl2 depleted) | Perlu penggantian sensor / baterai probe | **P4 Health** |
| **GNSS HDOP Degraded** | HDOP > 2.5 atau satelit terkunci < 5 (Blind spot / GPS Jamming) | Kehilangan akurasi titik lokasi peta | **P4 Health** |
