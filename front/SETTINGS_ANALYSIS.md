# Analisis Settings dan arsitektur modul

## Kesimpulan

PASTRACK dapat dibuat modular: satu modul memiliki halaman Vue, menu, konfigurasi, service data, daftar permission, event topic, aset, serta lifecycle backend-nya. Settings menjadi tempat meninjau, memasang, mengaktifkan, mengonfigurasi dan melepas modul. Instalasi seluruh bagian tersebut membutuhkan kontrak installer pada backend; browser sendiri tidak dapat mengaktifkan service Go atau menjalankan migrasi database.

Implementasi saat ini menambahkan halaman Settings dan fondasi modul frontend. Instalasi yang tersedia adalah aktivasi paket yang sudah disertakan dalam build frontend. Tidak ada binary Go atau endpoint live yang diubah.

## Mockup yang diperiksa

Referensi: `mookup/Settings/code.html`, `screen.png`, dan `DESIGN.md` (nama folder menggunakan huruf besar).

| Bagian                | Isi mockup                                                                                         | Hasil implementasi                                                                                                                         |
| --------------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Header                | Settings & Extension Engine, Core version, Reset Defaults, Save Changes, Cmd+S                     | Struktur sama, padding compact, draft perubahan, shortcut Ctrl/Cmd+S dan proteksi unsaved changes                                          |
| Module Manager        | Enam runtime packages, pencarian/filter, core terkunci, toggle extension, config, upload .ptm/.zip | Katalog enam manifest, dependency check, config schema, enable/disable/uninstall/install bawaan, menu dan route mengikuti status tersimpan |
| Branding & Web Assets | Metadata aplikasi, logo, favicon, Apple touch icon, manifest                                       | Editor metadata, preview gambar, favicon/touch icon, perubahan shell/title/accent setelah Save, unduh manifest JSON                        |
| Sound & Audio Alerts  | Master switch/volume, enam jenis insiden, tone/gain/test, quiet hours                              | Browser Web Audio preview, preferensi tersimpan, mute window Asia/Jakarta, notifikasi visual tetap berjalan                                |
| Access & RBAC         | Operator, undangan, empat role, delapan kelompok akses                                             | Draft operator, pilihan role, editor policy/2FA, duplicate role, export JSON, admin policy dilindungi dari pengubahan                      |

Mockup HTML belum menyediakan installer sungguhan: tombol Verify & Install langsung menampilkan toast bahwa hash terverifikasi tanpa memeriksa paket. Save, undangan operator dan pemutaran audio juga sebagian besar berupa simulasi toast. Implementasi tidak menggunakan klaim tersebut sebagai bukti backend berfungsi. Ukuran disk, worker BLE dan latency parser ditampilkan sebagai metrik yang belum tersambung, bukan angka produksi yang dibuat-buat.

## Katalog bawaan

| ID                          | Jenis                  | Kepemilikan frontend                                                                                    |
| --------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------- |
| `core.gis.tracker`          | Native Core / terkunci | Tracking view, route `/tracking`, deklarasi tracking permission/topic                                   |
| `telemetry.cfs300b.ble`     | Native Core / terkunci | Route fuel monitoring/device fuel, konfigurasi threshold BBM; detail engine masih memerlukan backend    |
| `ext.queclink.kinematics`   | Extension              | Route Driving Behavior, konfigurasi threshold impact; halaman detail masih scaffold                     |
| `ext.fleet.maintenance`     | Extension              | View maintenance, service data milik modul, route/menu, work-order consumer dan konfigurasi lead window |
| `core.geofence.spatial`     | Native Core / terkunci | Route Geofence; halaman dan engine geografis masih scaffold/backend                                     |
| `ext.socket.queclink.gtfri` | Extension              | Route Raw Telemetry; decoder runtime masih memerlukan backend                                           |

`Active` dalam preview menyatakan registry frontend aktif. Itu bukan laporan bahwa daemon, decoder atau parser server sudah dipasang. Tiga core dipertahankan aktif dan tidak dapat di-uninstall dari editor ini.

## Struktur yang sudah dibuat

```text
src/modules/
  registry.js                 manifest catalog dan lazy view entry
  state.js                    state Settings tersimpan dan helper status modul
  lifecycle.js                install, dependency check, disable, uninstall
  branding.js                 shell identity, favicon/touch icon, manifest download
  audio.js                    browser audio dan aturan quiet window
  tracking/
    manifest.json
    views/TrackingView.vue
  maintenance/
    manifest.json
    views/MaintenanceView.vue
    service.js
  fuel/manifest.json
  kinematics/manifest.json
  geofence/manifest.json
  telemetry/manifest.json
src/components/settings/
  ModuleManager.vue
  BrandingSettings.vue
  AudioSettings.vue
  AccessSettings.vue
src/views/system/SettingsView.vue
src/services/settings.js
public/examples/maintenance.module.json
```

Entry view modul dimuat lazy dari katalog build menggunakan `import.meta.glob`. Routes dimiliki manifest dan dibuat dari registry; menu dipadukan dengan urutan master-layout mockup. Detail glob import merujuk [dokumentasi resmi Vite](https://vite.dev/guide/features.html#glob-import). Registry ini adalah katalog pada waktu build: menambah entry kode baru tetap memerlukan build/deploy frontend. JSON upload tidak membuat JavaScript eksternal menjadi dapat dieksekusi.

Metadata manifest saat ini: `schemaVersion`, ID, versi, klasifikasi, minimum core, major core, dependencies, paths, entry, permissions, topics dan config schema. Deklarasi permission/topic sudah tersedia untuk integrasi; backend belum otomatis mendaftarkan endpoint, permission authorization, worker atau audio hardware hanya dari metadata tersebut.

## Perilaku lifecycle frontend

1. Pengguna melakukan perubahan pada draft. Aplikasi yang sedang berjalan belum berubah sampai Save Changes berhasil.
2. Save memeriksa nilai konfigurasi, dependency, core lock dan bentuk policy.
3. Dalam demo, state disimpan di browser; setelah penyimpanan berhasil registry dan shell diperbarui.
4. Extension nonaktif dihilangkan dari menu. URL langsung diarahkan ke halaman Module unavailable sebelum view modul dimuat.
5. Maintenance yang nonaktif tidak memuat data melalui service-nya dan tidak mengonsumsi work-order update. Data cache tetap disimpan, bukan dihapus; re-enable menyinkronkan data kembali.
6. Uninstall mengubah status catalog menjadi belum terpasang. Install kembali mengaktifkan entry yang telah dibundel beserta menu/route-nya. Ini tidak menghapus file bundle dari server atau menambah binary baru.

Upload preview menerima manifest JSON maksimal 128 KB. ID dan versi harus cocok dengan katalog yang disertakan, dan deklarasi dari upload tidak mengganti entry/permission/dependency terpercaya dari build. `.ptm/.zip` diberi penjelasan bahwa installer backend diperlukan. Tidak ada klaim signature/hash verified pada tahap ini.

## Instalasi paket lengkap pada backend Go

Rekomendasi awal saya adalah modular monolith: registry Go yang eksplisit dan kontrak service yang jelas, kemudian proses terpisah untuk extension yang benar-benar perlu didistribusikan secara independen. Ini merupakan pilihan desain untuk kebutuhan Pastrack, bukan satu-satunya arsitektur yang mungkin.

Saya tidak merekomendasikan menjadikan file Go `.so` acak sebagai fondasi installer. Dokumentasi resmi [Go `plugin`](https://pkg.go.dev/plugin) menjelaskan keterbatasan platform, kecocokan toolchain/dependency dan bahwa plugin yang dibuka tidak dapat ditutup. Proses terpisah dengan kontrak HTTP/gRPC dapat memberi lifecycle upgrade/stop yang lebih jelas. Versi package Go untuk build tetap dikelola melalui [Go modules](https://go.dev/doc/modules/managing-dependencies).

Satu distribusi modul produksi sebaiknya memuat:

| Bagian paket                 | Tanggung jawab installer                                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Manifest + signature         | Identity/version, publisher terpercaya, integrity, core/API compatibility, dependency graph                      |
| Frontend bundle + assets     | Sajikan entry/asset yang disetujui; strategi build-release atau remote bundle terisolasi harus dipilih eksplisit |
| Backend registration/service | API namespace, settings schema, lifecycle start/stop dan tenant-module gate                                      |
| Database migrations          | Versioned migration, backup/recovery plan dan kompatibilitas data                                                |
| Permissions                  | Permission IDs, default grants yang ditinjau dan server-side authorization                                       |
| Worker + event subscriptions | Start dengan context, hentikan/unsubscribe saat disable; drain pekerjaan berjalan                                |
| Dashboard/widgets/navigation | Registrasi yang dapat dicabut ketika module disabled                                                             |
| Tests + diagnostics          | Smoke check, health/readiness, logs dan versi yang sedang berjalan                                               |

Proses instalasi: upload ke staging → pemeriksaan signature/integrity → compatibility/dependency validation → tampilkan installation plan → buat job idempotent → migration/registration → start worker → health check → publish installed version. Jika gagal, job mencatat tahap dan mengembalikan versi aplikasi yang aman. Rollback data harus memakai rencana migrasi yang telah diuji; tidak semua migrasi dapat dibalik otomatis.

Disable melepas akses dan konsumen event setelah pekerjaan berjalan ditangani. Uninstall secara default menyimpan data. Penghapusan data harus menjadi tindakan terpisah yang eksplisit. Settings/installer harus tetap dapat diakses administrator meskipun extension gagal.

## Kontrak API usulan — belum dibuat

`src/services/settings.js` disiapkan untuk `GET /settings` dan `PUT /settings` dengan schema yang sama dengan editor. Nama route, envelope, autentikasi dan tenant scope harus disesuaikan ke backend nyata. Save ini bukan pengganti installer runtime.

| Endpoint usulan               | Fungsi                                                |
| ----------------------------- | ----------------------------------------------------- |
| `GET /modules/catalog`        | Catalog, versi, compatibility dan state per tenant    |
| `POST /modules/packages`      | Upload paket ke staging; tidak langsung mengeksekusi  |
| `POST /modules/:id/plan`      | Dry run dependency, migration dan permission changes  |
| `POST /modules/:id/install`   | Mulai job instalasi dengan idempotency key            |
| `GET /module-jobs/:id`        | Progress, log, hasil health check dan recovery action |
| `PATCH /modules/:id/state`    | Enable/disable dengan validasi dependency server      |
| `POST /modules/:id/uninstall` | Job uninstall yang mempertahankan data secara default |
| `GET/PUT /settings/branding`  | Metadata dan asset URL                                |
| `GET/PUT /settings/audio`     | Preferensi audio per tenant/operator                  |
| `GET/PUT /settings/rbac`      | Policy dengan enforcement di setiap endpoint backend  |

Perubahan settings memerlukan revision/ETag untuk mencegah editor saling menimpa. Instalasi perlu audit log, progress event SSE, penanganan kegagalan, autentikasi admin dan tenant isolation. Frontend gating hanya perilaku UI, bukan batas keamanan server.

## Batas yang tetap terlihat pada UI

- Draft undangan tidak mengirim email; checkbox 2FA hanya policy preview, bukan bukti 2FA telah diverifikasi.
- RBAC editor belum mengubah hak akses sesi atau menggantikan pemeriksaan authorization backend.
- Domain field tidak mengubah DNS/TLS, dan export manifest tidak menginstal PWA/service worker.
- Test Tone memutar nada browser yang singkat. External siren/GPIO tidak diakses. Autoplay tetap tunduk pada kebijakan browser dan membutuhkan interaksi pengguna.
- Threshold hardware dan modul yang masih scaffold disimpan sebagai konfigurasi; engine backend belum mengeksekusi kebijakan tersebut.
- State settings demo persisten di browser. Dataset kendaraan/work order demo tetap mempunyai lifecycle memori yang dijelaskan di README; bukan database produksi.

## Verifikasi

Pengujian mencakup empat tab, core lock, staged save, menu gating dan direct URL, persistence setelah reload, uninstall/reinstall, rejection paket tidak dikenal/versi salah/binary tanpa installer, branding, unduhan manifest, audio preferences, role policy, invitation draft dan unsaved changes. Pengujian regresi halaman sebelumnya serta transport SSE dijalankan terpisah. Tidak ada backend atau endpoint live yang diperbarui dalam pekerjaan ini.
