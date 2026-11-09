# AT+GTDOS — implementasi lokal

Acuan: GV30CEU v0306 halaman 134–137 (GTDOS), 187–188 (ACK), dan 268–269 (laporan output).

## Fungsi switch

Switch mengatur output digital 1 menjadi aktif atau tidak aktif. Efek terhadap relay mengikuti rangkaian yang dipasang. Ini bukan remote starter mandiri dan tidak mengubah status ignition berdasarkan hasil command.

Frame yang dibentuk:

`AT+GTDOS=<password>,0,1,1,<0|1>,0,0,,,1,1,5,,0,,<serial>$`

Konfigurasi awal menggunakan output steady, duration/toggle 0, pelaporan DOS aktif, batas kecepatan 1 km/jam, debounce 5 detik, dan pembatasan saat ignition ON. Output 1 bersifat latched menurut protokol.

## Alur yang diimplementasikan

1. Kontrol ON/OFF pada baris kendaraan membuka dialog status relay.
2. Dialog mengambil kemampuan receiver melalui Unix socket privat.
3. Status output dibaca dari GTDOS. Tanpa laporan, status ditulis **Belum diketahui**, bukan OFF.
4. Admin memilih ON/OFF dan mengonfirmasi kendaraan yang dipilih.
5. Receiver memeriksa profil aktif, koneksi TCP, assignment kendaraan, akun admin, GPS valid maksimal 30 detik, kecepatan maksimal 1 km/jam, dan command sebelumnya.
6. Ledger dan audit di-commit sebelum pengiriman TCP. Password tidak disimpan dalam ledger, respons API, atau audit.
7. Request key menjaga idempotensi. Serial unik per perangkat. Pengiriman yang tidak pasti tidak diulang otomatis.
8. ACK dicocokkan dengan perangkat, serial, dan raw receipt. ACK hanya berarti acknowledged, bukan mesin/output berhasil berubah.
9. Laporan GTDOS menyediakan status output yang teramati. Mode gradual tidak dianggap boolean ON/OFF.
10. Perubahan memicu jalur event SSE yang sudah ada.

## Pengujian yang lulus

- Format command ON/OFF dan penolakan input password yang dapat menyisipkan frame.
- Decoder ACK, laporan output, dan penolakan mode gradual sebagai boolean.
- Pengiriman pada koneksi simulasi menggunakan net.Pipe dan database terisolasi.
- Pengulangan request tidak mengirim command dua kali.
- ACK tidak mengubah status output terkonfirmasi.
- Raw laporan output terbaca sebagai status output.
- Perintah saat kendaraan bergerak ditolak.
- Dialog tanpa konfigurasi menampilkan status tidak diketahui dan menonaktifkan pengiriman.
- `go test -race ./...`, `go vet ./...`, serta 32 pengujian DOM.

## Konfigurasi runtime yang diperlukan

Web dan receiver memerlukan `REMOTE_OUTPUT_SOCKET` pada direktori privat yang sama. Receiver memerlukan `REMOTE_OUTPUT_PROFILES`, yaitu path file JSON dengan permission 0600. Setiap profil berisi `imei`, `password`, `output_id: 1`, dan `enabled: true`.

Password harus kredensial aktual perangkat. Nilai default dari contoh protokol tidak diasumsikan benar. File profil tidak boleh diletakkan dalam assets atau repository, dan isinya tidak dicetak dalam log.

Migration `005_remote_output.sql` menambah ledger command. Migration ini baru dijalankan dalam database pengujian, belum dalam database live. Tidak ada DROP/TRUNCATE.

## Status saat ini

- [x] Builder, decoder, ledger, dispatch, ACK, status report, dan UI lokal.
- [x] Pengujian otomatis dengan database terisolasi dan koneksi simulasi.
- [ ] File profil/kredensial perangkat aktual.
- [ ] Verifikasi fungsi relay pada wiring kendaraan.
- [ ] Pengujian perangkat fisik.
- [ ] QA visual browser; akses Browser Use sebelumnya dinonaktifkan.
- [ ] Deployment setelah checklist penerimaan 100%.

Receiver live dan database live belum diperbarui untuk fitur ini. Layanan command belum dikonfigurasi pada aplikasi lokal biasa, sehingga dialog akan menonaktifkan pengiriman. **Tidak ada perintah fisik yang dikirim. Fitur belum dinyatakan 100% secara operasional.**

## Pembaruan: nilai bawaan protokol

Kebutuhan file profil per perangkat di atas telah disederhanakan: jika tidak ada file override, receiver menggunakan password bawaan protokol `gv30ceu`, output 1, dan fungsi digital output aktif. Pengguna tidak diminta memasukkan password melalui UI. File override tetap opsional untuk perangkat yang password-nya sudah diubah atau untuk allowlist.

Field Password tetap dikirim dalam frame karena diwajibkan protokol; tidak menghilangkannya dari command. Test default tanpa profil dan test dispatch/ACK tetap lulus. Runtime receiver live belum diperbarui atau diaktifkan pada langkah ini.

## Aktivasi jalur command

Receiver dan web telah diperbarui pada 3 November 2026 setelah pengguna meminta agar jalur pengiriman berfungsi tanpa konfigurasi pengguna. Migration ledger additif diterapkan dengan cadangan database terlebih dahulu. Socket privat aktif. Pemeriksaan GET capability dari live dan lokal menunjukkan configured=true dan connected=true untuk device8/vehicle1. Lokal mem-proxy GET/POST remote-output ke server authoritative. Tidak ada POST command atau pengiriman fisik dilakukan selama aktivasi. Pengujian hardware ON/OFF menunggu tindakan pengguna dan laporan perangkat. Status terkonfirmasi masih null karena belum ada laporan GTDOS.


### Direct switch update
Switch now sends the ON/OFF request immediately on click, without a dialog or confirmation. Duplicate clicks are disabled during transmission. The switch shows the last requested state, not proof of engine state. UI checks: 34 passing; Go race tests and vet passing. No physical command issued during verification.


### Direct dispatch backend revision
Removed application GNSS-validity, freshness and speed prerequisite, plus the two-minute prior-command gate, per explicit user request. Authenticated admin, current vehicle assignment, active device connection, request validation, command ledger and idempotency checks remain. GTDOS device-side command parameters are unchanged. Isolated DB race tests passed, including transmitting OFF with invalid/stale GNSS and nonzero raw speed after a prior ACK. No physical commands sent during tests or deployment. Local requests use the live command service.


### GTDOS protocol correction after Python reference
Compared the supplied Python builder with protocol v3.06 section 3.6.3. Earlier frame had device speed threshold 1 and ignition-on output disabled. The Python uses threshold 5 and ignition-on enable 1. User explicitly selected speed threshold 0 (disabled) on clarification. New frame: `AT+GTDOS=<password>,0,1,1,<state>,0,0,,,1,0,5,,1,,<serial>$`. DOS reports remain enabled. RX examples showed seven-field ACK with serial at field 4, not Python handler field 5; existing decoder retained.

Added TX command logging to raw-messages.txt when RAW_TEXT_LOG=on, with password redacted, serial and send state. Entire Go race suite and vet pass. Tests assert exact wire frame, password-free logging, command idempotency and ACK separate from output confirmation. No physical commands issued during verification.


### Serial conflict and switch state correction
Production commands reached 000A. MAX(CONV(...)) compared decimal strings lexically and selected 9 over 10, attempting duplicate 000A. Fixed with explicit UNSIGNED cast before MAX. Regression preloads 0009 and 000A and asserts next transmission 000B in isolated DB.

Switch now prioritizes reported ignition ON/OFF over browser requested state; sends opposite requested state and awaits a new report before showing an ignition change. Unknown ignition falls back to the last session request. This is not output relay confirmation. 37 UI tests pass including known ON sending OFF without optimistic state change. Remoteoutput race tests and vet pass. No physical command sent for verification.


### Relay switch separated from ignition
Supersedes prior ignition-based switch change: switch uses last valid GTDOS output-1 active state; ignition remains an independent indicator. Added remote_output_active and remote_output_reported_at to fleet and vehicle telemetry from assigned-device GTDOS raw reports. No engine state inferred from ACK. Known output state overrides requested session state. Click checks latest output capability before selecting inverse ON/OFF request. Regression covers output ON while ignition OFF, rendering Matikan Mesin and sending active=false. 37 UI tests, platform race tests and vet pass. No physical command issued.
