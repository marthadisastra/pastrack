# Queclink GV30CEU dan CFS300B — mapping berbasis bukti

## Sumber dan batas kepastian

- parser.go/storage.go/fuel.go: perilaku implementasi sekarang.
- Document/GV30CEU_Develop_Suit_R01A03V21_Eng.zip → protocol/GV30CEU@Track_Air_Interface_Protocol_v0306.pdf: 457 halaman; STT §4.5.8 hlm 256–258; ERI §4.2.3 hlm 198–203; tipe/model aksesori hlm 77 dan 455.
- docs/cfs300b-water-immersion-run-2026-11-02.csv: sampel lima-byte aktual, bukan frame GTERI lengkap. Uji air tidak membuktikan liter BBM.
- Dokumen tersedia bukan bukti setiap message sudah diamati atau didukung decoder. Frame lengkap tersanitasi untuk setiap varian/mask masih diperlukan sebelum mengaktifkan parser baru. Jangan mengisi field belum dikenali.

## Envelope dan jalur persistence

TCP byte frame → spool durable dengan receipt UUID stabil → INSERT raw receipt immutable → classifier (prefix/code/version) → parser registry per model/version → validator → normalized facts dalam transaksi + idempotency key → event engine → SSE invalidation → query API. Bila parse gagal: raw tetap tersimpan dan parse error terpisah; tidak memblokir receipt valid berikutnya tanpa batas. Unknown raw tetap dapat diinspeksi oleh pengguna berizin, identifier disamarkan pada log.

| Bagian | Mapping | Aturan |
|---|---|---|
| byte payload | device_raw_messages.payload/hash | Simpan byte asli, termasuk framing; normalisasi parser tidak memutasi raw. |
| server receipt | received_at, spool_receipt_id | UTC server, bukan waktu GPS. |
| +RESP / +BUFF | protocol_parse_logs.fields.envelope | Buffered mempertahankan event time; received_at tetap waktu server. |
| field 0 | protocol_message_types.code | Ambil suffix hanya jika envelope valid. |
| field 1 | raw.protocol_version | Versi lengkap; gunakan registry layout, bukan parser generik posisi tetap. |
| field 2 | raw identifier + gps_devices lookup | Validasi IMEI; jangan tampilkan/log nilai penuh. Unknown device tidak otomatis diassign ke kendaraan. |
| field 3 | claimed device name | Jangan menganggap nama perangkat selalu model hardware; cocokkan inventory/config dan layout terverifikasi. |
| +ACK / +SACK | raw + parse log | Struktur belum diverifikasi terhadap frame lokal lengkap. Jangan memakai indeks RESP atau menganggap ACK command sukses. |
| frame malformed/biner | raw + protocol_parser_errors | unsupported/malformed dibedakan; tanpa posisi/fuel palsu. Receiver sekarang ASCII, dukungan binary tidak diklaim. |

## GTSTT: mapping layout yang sekarang didukung

Indeks berikut **zero-based termasuk header**. Parser lama membatasi 20 field untuk versi 80201E0100 dan 80201E0306; ini tidak berlaku untuk seluruh varian. Manual memperlihatkan optional fields berdasarkan append mask, sehingga parser baru wajib mask-aware. Dukungan versi 0100 saat ini berasal dari implementasi, belum diverifikasi dengan manual versi itu.

| Indeks | Field manual | Target | Validasi/kepastian |
|---|---|---|---|
| 4 | Motion Status | device_status_logs.status_code_raw | Raw dulu. Kode manual 11/12/16/21/22/41/42 tidak membuktikan ignition fisik. |
| 5 | GNSS Accuracy | gps_positions.gnss_accuracy_raw | 0–50 pada manual; jangan dilabel HDOP/satellites. Interpretasi fix mengikuti aturan manual yang diverifikasi. |
| 6 | Speed | gps_positions.speed_kmh | km/h, 0..999.9 sesuai manual, nullable bila tidak ada. |
| 7 | Azimuth | heading_deg | 0..359 derajat. |
| 8 | Altitude | altitude_m | Meter; nullable. |
| 9 / 10 | Longitude / Latitude | longitude / latitude | -180..180 / -90..90; jangan tukar urutan. |
| 11 | GNSS UTC Time | gps_time | YYYYMMDDHHMMSS UTC; parser sekarang menerima location eksternal, perlu diperbaiki/test agar tidak bergeser WIB. |
| 12 / 13 | MCC / MNC | network_status_logs | Simpan sebagai string agar leading zero tidak hilang. |
| 14 / 15 | LAC / Cell ID | network_status_logs | Hex tervalidasi, tidak mengasumsikan RSSI. |
| 16 | Position Append Mask | parse_log.fields | Mengontrol keberadaan optional fields; panjang message tidak selalu 20. |
| setelah mask | Satellites in Use, Device Status optional | gps_quality_logs, device_status_logs | Indeks dihitung dari mask; jangan hardcode 17 sebagai satellites. |
| tail-2 / tail-1 | Send Time / Count Number | device_send_time / processing identity evidence | Setelah validasi struktur tail; count hex dapat wrap, tidak unik sepanjang umur perangkat. |

GTSTT tanpa koordinat yang teramati tetap menjadi status/receipt, tidak membuat posisi (0,0). Motion tidak sama dengan engine running. physical_ignition_state=NOT_INSTALLED; virtual ignition disabled sampai konfigurasi dan validasi terpisah.

## GTERI dan BLE CFS300B

GTERI mempunyai ERI mask, jumlah posisi, append mask dan jumlah aksesori; **tidak ada satu indeks global untuk sensor**. Decoder berjalan menurut mask dan jumlah record. Jangan mencari marker string CFS300B lalu otomatis menerima field di dekatnya sebagai data final.

| Sumber | sensor_telemetry | fuel_readings / aturan |
|---|---|---|
| Accessory Type 27 + Model 0 | accessory_type=27, accessory_model=0 | CFS300B teridentifikasi menurut manual hlm 77/455, dengan layout valid. |
| Accessory Index | accessory_index | UNIQUE(raw_message_id,accessory_index); satu raw bisa berisi beberapa sensor. |
| Accessory Name/MAC optional | ble_sensors + binding lookup | Parse hanya jika append mask mengaktifkan; nama bukan identitas unik. Observasi tidak mengubah binding administratif. |
| Raw Data 5 byte | payload_hex | Tepat 10 hex digit untuk CFS300B; invalid length/hex → invalid, bukan nol. |
| byte 0 | tilt_raw | Manual menyebut Tilt Angle; skala/unit tidak dijelaskan dalam tabel, belum konversi derajat. |
| byte 1–2 | fuel_height_raw | Manual menyebut Fuel Height; simpan word raw dengan decoder evidence. Uji byte order terhadap frame/sampel; jangan klaim mm/cm. |
| byte 3–4 | volume_raw | Manual menyebut Volume; skala/unit belum terverifikasi, bukan otomatis liter. |
| battery/status/temperature/RSSI/timestamp | kolom opsional nullable | Hanya berdasarkan append mask dan tipe yang sesuai. Status absent bukan disconnected. Timestamp absent → sensor_time NULL; received_at tetap terisi. |
| telemetry tervalidasi, input belum terkalibrasi | raw_measurement/raw_unit nullable | validity=uncalibrated; liters dan percent NULL. Raw tidak harus dipaksakan menjadi scalar jika quantity belum terpilih. |
| calibration approved + tank association + quantity/unit verified | calibration_id, vehicle_id | Interpolasi piecewise monotonic sesuai input quantity; output liter, percent=100*liter/capacity hanya jika capacity valid. Di luar domain → invalid/out_of_range, jangan clamp diam-diam. |
| sensor stale/disconnected | status log, latest observed | Tidak insert fuel zero, tidak menganggap last valid sebagai current; tampilkan timestamp/stale. |

Contoh bukti lokal `560104001A`: segment raw `56 | 0104 | 001A`; implementasi lama menghasilkan kandidat 260 dari word tengah, field akhir 26. Ini mendukung ekstraksi word, tidak membuktikan satuan atau konversi 10:1. Nilai sensitif sensor/device tidak disalin ke dokumen ini.

## Message inventory dan kemampuan

| Jenis | Bukti lokal | Status desain dan API |
|---|---|---|
| GTSTT RESP/BUFF | parser/tests + audit lama | Posisi/status untuk layout tervalidasi; /api/v1/vehicles/{id}/telemetry dan health. |
| GTERI | storage.go + catatan uji | Sensor discovery eksperimental sekarang; decoder mask-aware baru belum diterapkan. /api/v1/sensors/{id}/telemetry, fuel endpoints setelah validasi. |
| ACK / trafik tanpa identitas | catatan audit terdahulu | Raw preserved, command correlation belum dibuktikan. /api/v1/devices/raw-messages. |
| Heartbeat/status/alarm/GPS reports lain | belum ada inventory frame lengkap pada pekerjaan ini | Registry planned; unknown retained, tidak diklaim supported. |

Fuel drop/refueling hanya dari rangkaian liter valid dengan minimum sample count, waktu gap maksimum, smoothing, hysteresis, threshold liter/rate dan versi rule tersimpan. Disconnect, kalibrasi baru, assignment berubah, outlier dan telemetry terlambat tidak langsung menghasilkan theft/refuel. Alarm hardware hanya dibuat jika code/semantiknya terverifikasi. Trip berbasis pergerakan GPS tervalidasi, bukan engine/ignition; gap dilabel unknown, bukan driving.

## Implementasi terbaru 3 November 2026

Manual resmi hlm 193–194 (definisi GNSS Accuracy) memberi bukti tambahan: 0 berarti fix gagal dan koordinat last-known; 1–50 berarti fix berhasil dan nilai HDOP. Pernyataan review awal yang belum memetakan field ini ke HDOP telah digantikan bukti tersebut. Implementasi v2 menerapkannya untuk versi 0306; versi 0100 tetap unknown. Satellite count tidak digunakan sebagai pengganti fix validity.

Parser ERI kini mendukung mask 0x100, per-position append flags 0/1, kelompok GPS dan aksesori BLE sederhana sesuai manual. Extended accessor mask/CAN/RAT tetap unsupported. Full ERI fixtures pada test adalah constructed/document-derived, bukan rekaman perangkat fisik. ACK/SACK disimpan tanpa menebak identitas/hasil command.
