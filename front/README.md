# PASTRACK Frontend

Template Vue 3 berdasarkan delapan mockup HTML di `mookup/`. Menggunakan Vite, Vue Router, dan Tailwind CSS lokal (tanpa Tailwind CDN). Bahasa antarmuka mengikuti mockup: English, dengan teks master layout Indonesia.

## Menjalankan

```sh
cd /Applications/AMPPS/www/pastrack/front
npm install
cp .env.example .env
npm run dev
```

Buka `http://127.0.0.1:5173/#/login` dan gunakan akun Go yang sudah ada. Autentikasi terhubung ke Go/live melalui proxy HTTPS. Halaman All Assets menggunakan registry Go; dashboard dan modul lainnya tetap mengikuti mode demo/API yang sudah ada. Versi live: https://toko.azlaundry.my.id/app/. Lihat [AUTH_INTEGRATION.md](AUTH_INTEGRATION.md) untuk API, session, SMTP, dan pengujian. Untuk preview sepenuhnya offline, set `VITE_AUTH_MODE=demo`; hanya pada mode tersebut password bebas digunakan dan `fail` menampilkan contoh error.

```sh
npm run build
npm run preview
```

Hasil produksi berada di `dist/`. Routing memakai hash agar dapat di-refresh pada static hosting/AMPPS tanpa rewrite server. Jika diakses melalui subfolder AMPPS, atur `VITE_BASE_PATH=/pastrack/front/dist/` sebelum build, kemudian buka `http://localhost/pastrack/front/dist/` (sesuaikan port AMPPS).

## Struktur

```text
src/
  assets/             CSS global dan gambar peta dari mockup
  components/
    ui/               Icon, badge, kartu statistik, modal, toast
    fleet/            Peta, grafik BBM, tabel dan inspeksi kendaraan
  composables/        State fleet bersama dan notifikasi
  layouts/            AppLayout dan AuthLayout (login/recovery/reset)
  mocks/              Seed data kendaraan dan work order
  modules/            Registry, manifest, lifecycle dan view/service milik modul
  router/             Daftar route, menu sesuai mockup dan lazy-loaded views
  services/           HTTP client, auth, fleet, ekspor CSV
  views/
    auth/             Login, forgot password, reset password
    dashboard/        Dashboard operasional
    vehicles/         Registry aset
    system/           Settings, alert center, master layout dan halaman 404
mookup/               Referensi asli; tidak diubah
tests/                Pengujian alur pengguna melalui Playwright
```

## Halaman dan interaksi

| Route                         | Halaman                                                                                                                  |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `/#/dashboard`                | Statistik fleet, peta, grafik BBM, filter state, inspeksi dan ekspor CSV                                                 |
| `/#/tracking`                 | Pilih unit dari roster/peta, pencarian, filter, tampil/sembunyikan roster dan HUD, zoom, layer, trail dan simulasi relay |
| `/#/vehicles`                 | Registry live, pencarian/filter fleet, list/grid, pagination, tambah/edit aset, dan assignment gateway terdaftar         |
| `/#/maintenance`              | Filter work order, jadwalkan maintenance, mulai servis, selesaikan WO, history dan ekspor CSV                            |
| `/#/template`                 | Contoh area konten modular dengan shell bersama                                                                          |
| `/#/login`                    | Login Go dan pemulihan session                                                                                           |
| `/#/forgot-password`          | Permintaan email reset melalui Go/SMTP                                                                                   |
| `/#/reset-password?token=...` | Validasi kekuatan password dan kecocokan konfirmasi                                                                      |

Demo registry berisi enam aset. Statistik jumlah aset dan fuel dihitung dari data ini, sehingga tidak mengklaim 48 aset seperti ilustrasi mockup. Data berubah dalam memori, dibagikan antarhalaman, dan kembali ke seed setelah reload. Telemetri demo diperbarui setiap 4 detik dan dibagikan antara dashboard, tracking, tabel, grafik sesi dan modal. Nilai identitas tetap stabil; geofence dan dasar peta masih ilustrasi. Login Go menggunakan kontrol akses session server. Email reset dikirim hanya setelah admin mengaktifkan SMTP; perintah kendaraan masih simulasi. Diagram arsitektur yang hanya berupa gambar tetap menjadi referensi, bukan halaman aplikasi.

## Sidebar dan kepadatan tampilan

Sidebar mengikuti 15 menu dalam empat kelompok mockup: Overview, Fleet Management, Intelligence, dan Hardware & System. Lebar desktop 216px dan dapat diciutkan menjadi 56px melalui tombol header. Menu tanpa desain halaman memiliki template kosong tersendiri, kecuali Alert Center yang kini memuat riwayat notifikasi. Master Layout tetap dapat dibuka dari footer.

Header 48px, footer 24px, kartu statistik dua kolom pada mobile, tabel dan modal compact. Peta tracking memenuhi ruang viewport desktop dengan roster/HUD yang dapat diminimalkan. Login, forgot password dan reset password memakai komposisi berbeda sesuai referensi. SSO masih menampilkan penjelasan konfigurasi, belum menghubungi provider.

## Integrasi backend Go

1. Halaman All Assets memakai `GET /vehicles`, `POST /vehicles/inventory`, `PUT /vehicles/{id}/inventory`, `GET /devices`, dan `POST /vehicles/{id}/assignments`. Kode aset disimpan sebagai `vehicle_code`; gateway dipilih dari perangkat aktif terdaftar menggunakan empat digit terakhir IMEI karena API hanya mengirim IMEI tersamarkan. Perubahan registry memakai `row_version` agar konflik edit terdeteksi.
2. Untuk mengaktifkan API pada modul lain, set `VITE_USE_MOCK=false` dan `VITE_API_BASE_URL` ke alamat API Go, lalu restart Vite/build ulang. Variabel `VITE_*` bersifat publik; jangan simpan secret di sini.
3. Integrasi halaman lain masih bertahap; periksa endpoint yang tersedia sebelum mengaktifkan API untuk modul tersebut.
4. HTTP client `src/services/http.js` mengirim cookie melalui `credentials: 'include'` dan menangani JSON/204 serta error response `{ "message": "..." }`.
5. Dashboard memiliki transport WebSocket tersendiri. Halaman Tracking dan modul demo lainnya masih memakai adapter masing-masing; sesuaikan kontrak data sebelum mengaktifkannya ke API.

| Method | Endpoint usulan         | Payload / hasil                                  |
| ------ | ----------------------- | ------------------------------------------------ |
| POST   | `/auth/login`           | `{ email, password, remember }` → session server |
| POST   | `/auth/forgot-password` | `{ email }`                                      |
| POST   | `/auth/reset-password`  | `{ token, password, terminateSessions }`         |
| GET    | `/vehicles`             | Paged registry dengan envelope `data` dan `page` |
| POST   | `/vehicles/inventory`   | Kode aset, nama, plat dan kapasitas → `{ data: { id } }` |
| PUT    | `/vehicles/:id/inventory` | Pembaruan inventory dengan `row_version`         |
| POST   | `/vehicles/:id/assignments` | `{ device_id, starts_at }` → assignment         |
| GET    | `/maintenance`          | Array `WorkOrder` langsung                       |
| POST   | `/maintenance`          | Field jadwal → `WorkOrder` tersimpan             |
| PATCH  | `/maintenance/:id`      | `{ status }` → `WorkOrder` yang diperbarui       |

Contoh lengkap field ada di `src/mocks/fleet.js`. `Vehicle` menggunakan `id`, `plate`, `model`, `type`, `imei`, `state`, `status`, `tone`, `speed`, `fuel`, `capacity`, `battery`, `probeBattery`, `rssi`, `ignition`, `location`, `odo`, `lat`, `lng`, `x`, `y`. Angka harus bertipe number. `WorkOrder` menggunakan `id`, `vehicleId`, `scope`, `title`, `description`, `trigger`, `current`, `unit`, `facility`, `bay`, `budget`, `status`, `date`. Jika backend memakai envelope `{ data: [...] }` atau penamaan berbeda, normalisasikan di services sebelum mengembalikan nilai ke state/view.

Simulasi relay dan poll packet tetap lokal meskipun API mode diaktifkan. Implementasikan endpoint perintah, otorisasi dan konfirmasi operasional secara eksplisit ketika backend siap.

## Verifikasi

```sh
npx playwright install chromium
npm test
npm run test:realtime
npm run build
```

Playwright memeriksa inspeksi dashboard, ekspor CSV, registry/pairing, transisi maintenance, tracking, auth/reset password, seluruh route dan navigasi mobile. Tidak diperlukan backend untuk pengujian demo.

Font dan ikon mengambil Google Fonts; stylesheet Tailwind serta gambar peta dibundel lokal. Untuk penggunaan offline penuh, host font/ikon secara lokal. Peta juga menyediakan fallback vektor bila gambar gagal dimuat.

Pekerjaan ini hanya menambahkan frontend. Backend Go dan endpoint live tidak diubah.

## UX, loading dan realtime

Hasil riset referensi, keputusan, serta batas verifikasi tersedia di [UX_RESEARCH.md](./UX_RESEARCH.md). Lonceng menampilkan filter All/Unread, mark read dan riwayat. Read berbeda dari penyelesaian insiden. Demo read IDs disimpan di browser; API read state belum ditulis kembali ke server.

Status bar menyediakan Pause/Resume, umur pembaruan dan retry koneksi. Input dan fokus tetap dipertahankan ketika menerima packet. Ketika offline atau tab tersembunyi, stream dihentikan; penyambungan ulang otomatis mempertahankan nilai terakhir. Gambar peta memiliki skeleton/error/retry; route memiliki progress bar tertunda. Error dan data lama tidak disamarkan sebagai keberhasilan.

Event SSE normalized yang didukung:

```text
event: snapshot
data: {"vehicles":[Vehicle],"orders":[WorkOrder],"notifications":[Notification]}

event: vehicle
data: {"id":"HAULER-01","fuel":331.2,"speed":72,"updatedAt":"2026-11-07T05:00:00Z"}

event: work-order
data: {"id":"WO-8821","status":"In Workshop"}

event: notification
data: {"id":"event-123","title":"Fuel warning","message":"Inspect this unit","severity":"critical","vehicleId":"HAULER-01","target":"/tracking?unit=HAULER-01"}
```

`Vehicle`, `WorkOrder`, `Notification` pada contoh snapshot adalah placeholder tipe, bukan literal JSON. Event `vehicle`/`work-order` boleh berisi field parsial untuk entitas yang dikenal. ID wajib stabil, tipe angka harus number, dan snapshot menyediakan seluruh record yang hendak ditampilkan. Notification memiliki `createdAt` ISO opsional, `severity` info/success/warning/critical, serta ID unik untuk deduplikasi.

Adapter juga mendukung `event: values` dari backend Go lokal (`internal/platform/server.go`, `live_delta.go`) dengan upserts/removed/order per source. ID backend dipetakan ke `Vehicle.sourceId` atau `Vehicle.id`. Field contoh `speed_kmh`, `fuel_liters`, `latitude`, `longitude` dipetakan ke model UI; ini belum merupakan normalizer lengkap semua respons API. Snapshot REST harus memuat unit baru dan menormalisasi field serta null/invalid measurements sesuai kontrak UI.

`npm run test:realtime` membuka server fixture dan Vite terisolasi pada port 5183, kemudian menutup keduanya. Tidak menyentuh endpoint live. Pengujian ini meliputi event normalized, patch Go, notifikasi, pause/resume, offline retention dan automatic reconnect.

## Settings dan module manager

Settings tersedia pada `/#/settings`, dengan tab Module Manager, Branding & Web Assets, Sound & Audio Alerts, serta Access & RBAC. Analisis mockup, struktur modul, batas implementasi dan proposal installer Go tersedia di [SETTINGS_ANALYSIS.md](./SETTINGS_ANALYSIS.md).

Enam manifest berada di `src/modules/*/manifest.json`. Maintenance dan Tracking kini memiliki view di folder modulnya sendiri; service maintenance juga dipisah. Registry membuat route modul dan status tersimpan memfilter menu. Extension disabled/uninstalled tidak dapat dibuka melalui URL langsung. Core modules terkunci. Semua perubahan memakai draft sampai Save Changes.

Konfigurasi demo disimpan dengan key `pastrack.demo.settings.v1` di localStorage. Reset Defaults hanya mengisi draft; simpan untuk menerapkannya. Uninstall frontend tidak menghapus data atau bundle dari disk. Install preview memakai modul yang sudah dibundel; tersedia contoh manifest di `public/examples/maintenance.module.json`. Upload `.ptm/.zip` memerlukan installer Go dan tidak dieksekusi dalam browser.

Branding yang disimpan memperbarui nama/tagline/logo shell, title, primary-action accent, favicon dan touch icon. Audio menggunakan Web Audio dengan test tone, gain dan quiet hours. Invitation/2FA/RBAC adalah policy editor preview, bukan fitur identity server. Dalam mode API, service mengusulkan `GET /settings` dan `PUT /settings`; sesuaikan kontraknya sebelum digunakan.
