# Analisis mockup

## Inventaris

Referensi awal berisi delapan folder berisi `code.html` + `screen.png`, dan satu folder diagram arsitektur berisi gambar saja. Referensi HTML menggunakan Tailwind melalui CDN dan sebagian memakai event global/DOM langsung.

| Referensi                       | Implementasi Vue                            |
| ------------------------------- | ------------------------------------------- |
| Login split observability       | AuthLayout + AuthBrandPanel + LoginView     |
| Forgot password                 | ForgotPasswordView                          |
| Reset password                  | ResetPasswordView                           |
| Enterprise dashboard + inspect  | DashboardView + VehicleInspectModal         |
| Live tracking + minimize        | TrackingView + FleetMap                     |
| Vehicles all assets + add asset | AssetsView + AppModal                       |
| Vehicles maintenance            | MaintenanceView + AppModal                  |
| Master layout kosong            | AppLayout + TemplateView                    |
| Diagram arsitektur kendaraan    | Referensi konseptual, tidak dijadikan route |

## Sistem visual

- Latar slate muda, panel putih, border tipis, radius 8–16px, shadow ringan.
- Biru sebagai aksi utama; hijau untuk normal, cyan untuk sensor/idle, merah untuk alert, amber untuk belum paired.
- Inter untuk UI dan JetBrains Mono untuk data telemetri. Material Symbols untuk ikon.
- Referensi shell: sidebar 256px dan header 64px. Versi compact: sidebar 216px (56px saat diciutkan), header 48px, footer 24px, konten scroll independen.
- Tabel memuat detail padat, dengan horizontal scrolling pada layar kecil. Statistik memakai dua kolom pada mobile dan empat kolom pada desktop.

## Keputusan implementasi

Sidebar/header/footer disatukan agar semua halaman operasional memakai navigasi yang sama. Seluruh 15 menu mengikuti kelompok, urutan, label dan ikon pada mockup. Menu yang belum memiliki desain halaman diarahkan ke ModuleView, yaitu template kosong bertajuk sesuai modul. Panel ilustrasi login diadaptasi dari markup referensi, sementara form, state dan event diimplementasikan menggunakan Vue. Forgot password memakai grid security di kiri dan form recovery di kanan. Reset password memakai form utama di kiri dan panel compliance/network di kanan, sesuai komposisi referensi. Dev simulator pada mockup diganti dengan interaksi form nyata pada mode demo.

Jumlah record, ringkasan fleet dan pagination mengikuti enam record contoh agar tabel, pencarian dan statistik konsisten. Gambar satellite dashboard dan live tracking disimpan di assets. Peta masih ilustrasi. Grafik menampilkan sampel sesi yang diterima; data demo bergerak setiap 4 detik dan statusnya dijelaskan pada bar koneksi.

Model data, transport HTTP, state bersama dan presentasi dipisah agar integrasi Go tidak memerlukan perubahan besar pada komponen UI. Belum ada deployment karena tidak ada perubahan backend/endpoint.

## Pemeriksaan compact

Sidebar tersedia dalam mode penuh dan ikon. Panel roster/HUD tracking mengambang di atas peta dan bisa diminimalkan; kontrol peta dan pin dibatasi ke area yang bebas panel. Pada mobile, peta ditampilkan lebih dahulu dan panel ditumpuk dengan scroll masing-masing. Kartu, tabel, filter dan modal memakai padding yang lebih kecil. Teks panjang pada tabel maintenance diringkas agar baris tetap padat.

## Tambahan Settings

Mockup `mookup/Settings` dianalisis dan diimplementasikan menjadi empat tab. Detail pemetaan, module registry dan proposal installer Go ada di [SETTINGS_ANALYSIS.md](./SETTINGS_ANALYSIS.md).
