# Riset UX dan keputusan implementasi

Riset referensi untuk pola notifikasi, loading, telemetri realtime dan interaksi operator. Ini merupakan desk research dan evaluasi implementasi, bukan hasil wawancara atau usability testing dengan pengguna Pastrack. Sumber diakses 7 November 2026.

## Sumber dan penerapan

| Sumber primer                                                                                                     | Temuan relevan                                                                                                                    | Penerapan di Pastrack                                                                                                                                                                       |
| ----------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Carbon Design System: Notifications](https://carbondesignsystem.com/patterns/notification-pattern/)              | Pesan perlu relevan, tepat waktu, memberi konteks dan langkah berikutnya. Interupsi berlebihan mengurangi efektivitas notifikasi. | Panel lonceng, severity, jumlah unread, riwayat, tautan unit/halaman, deduplikasi ID. Tidak ada popup pada setiap packet telemetri. Critical event baru dapat menghasilkan toast persisten. |
| [Carbon Design System: Loading](https://carbondesignsystem.com/patterns/loading-pattern/)                         | Skeleton cocok untuk struktur data yang sudah diketahui; indikator proses harus ditempatkan sesuai area yang menunggu.            | Skeleton kartu/tabel saat bootstrap; progress bar navigasi lazy route; spinner hanya pada aksi; skeleton/failure/retry untuk gambar peta.                                                   |
| [W3C: Understanding SC 4.1.3 Status Messages](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)   | Perubahan status perlu dikenali teknologi bantu tanpa memindahkan fokus.                                                          | Status koneksi dan toast memakai status/alert yang sesuai. Data tabel tidak diumumkan setiap packet; node dan fokus dipertahankan.                                                          |
| [W3C: Understanding SC 2.2.2 Pause, Stop, Hide](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html) | Konten yang memperbarui otomatis perlu dapat dikendalikan ketika pembaruan berjalan bersamaan dengan aktivitas lain.              | Tombol Pause/Resume, status eksplisit saat nilai dibekukan, stream berhenti ketika tab disembunyikan, reduced motion.                                                                       |

Durasi progress bar tertunda 150 ms, highlight nilai 700 ms, toast biasa 6 detik, dan interval demo 4 detik adalah keputusan desain implementasi ini, bukan angka yang diwajibkan sumber. Jalur API produksi tidak diberi penundaan buatan. Demo request memiliki latency contoh 280 ms agar state loading dapat diperiksa.

## Konteks pengguna yang digunakan

Hipotesis pekerjaan utama: dispatcher memantau banyak unit, harus mengenali anomali, membuka unit yang relevan, dan melakukan pencatatan aset/maintenance tanpa kehilangan konteks. Karena itu:

- Tabel tidak disortir ulang hanya karena speed/fuel berubah. Record yang sudah ada diperbarui per field.
- Grafik mengikuti sampel yang diterima sejak sesi dibuka; bukan berpura-pura menampilkan history 12/24 jam dari backend.
- Data identitas/capacity tidak dibuat berubah hanya demi animasi. Nilai telemetri dan turunannya berubah ketika menerima packet.
- Saat offline, nilai terakhir tetap terlihat dengan keterangan koneksi; tidak diganti angka nol.
- Status `Live` menyatakan transport terhubung. Waktu `Updated` menyatakan penerimaan/sinkronisasi data fleet, bukan jaminan GPS/fuel measurement baru untuk setiap sensor.

## Lonceng dan riwayat

Lonceng membuka panel nonmodal yang tidak menutupi seluruh layar. Filter All/Unread, mark read per pesan, mark all read, severity, waktu, empty state dan riwayat tersedia. Membuka panel tidak otomatis menandai seluruh pesan. Escape menutup panel dan mengembalikan fokus ke lonceng. Notifikasi unit membawa pengguna ke tracking beserta pilihan unit.

`Read` adalah status pesan. Itu tidak mengakui atau menyelesaikan alert kendaraan. Badge Alert Center di sidebar tetap menghitung anomali fleet; badge lonceng menghitung pesan belum dibaca.

Riwayat dibatasi 200 pesan di memori. ID yang sudah dibaca dalam mode demo disimpan lokal di browser. Mode API belum menyinkronkan aksi mark read ke server; backend perlu menyediakan kontrak read/ack terpisah. Target/isi dan hak akses insiden tetap menjadi tanggung jawab integrasi backend.

## Loading dan motion

Bootstrap menampilkan skeleton, sedangkan refresh berikutnya mempertahankan konten lama. Gambar peta memiliki skeleton, batas waktu 15 detik, pesan kegagalan dan retry; mode vector tetap dapat digunakan. Lapisan dekoratif tidak menangkap klik. Bar navigasi hanya muncul setelah 150 ms agar route cepat tidak berkedip.

Nilai berubah mendapat highlight singkat, marker peta berpindah dengan transisi, dan pergantian halaman memakai fade pendek. Tidak ada count-up angka yang membuat pembacaan telemetri tertunda. Semua motion nonesensial menghormati `prefers-reduced-motion`.

## Realtime dan pemulihan

Satu stream dibuka untuk satu workspace. Memindah halaman di workspace tidak membuka timer tambahan. Saat logout/menuju halaman autentikasi, stream dan listener ditutup. Tab tersembunyi menghentikan stream; kembali aktif menyambungkan ulang.

Mode demo menggunakan simulator 4 detik. Mode API menggunakan EventSource/SSE dengan cookie credentials dan reconnect 1, 2, 4, 8, 16, hingga 30 detik. Data lama dipertahankan, status offline/reconnecting ditampilkan, dan retry manual tersedia. `VITE_STALE_AFTER_MS` mengontrol peringatan tidak ada pembaruan; heartbeat/notifikasi saja tidak membuat telemetri dianggap baru.

Adapter mendukung event normalized `snapshot`, `vehicle`, `work-order`, `notification`, serta patch `values` yang ditemukan pada backend Go lokal. REST tetap memerlukan normalisasi field/envelope dan identitas unit terhadap backend sebenarnya. Patch Go menargetkan entitas yang sudah dikenal melalui `sourceId` atau `id`; entitas baru harus masuk melalui snapshot REST/normalized. Peta masih ilustrasi, bukan GIS geografis.

## Interaksi aksi pengguna

Form memperlihatkan status menyimpan, mencegah submit ganda, serta tidak dapat ditutup di tengah penyimpanan. Jika ada perubahan yang belum disimpan, Cancel/Escape/backdrop menawarkan Keep editing/Discard changes. Navigasi Back meminta konfirmasi yang dapat dibatalkan tanpa kehilangan isi form; tab/window close memakai peringatan browser standar.

Hasil aksi ditampilkan melalui toast dan riwayat notifikasi. Toast biasa dapat ditutup dan tidak kedaluwarsa ketika sedang difokus/di-hover; error tetap sampai ditutup. Maksimal tiga toast terlihat agar tidak menutupi layar. Menutup toast tidak menghapus notifikasi/insiden.

## Bukti verifikasi dan batasannya

Pengujian browser memeriksa unread persistence, tautan ke unit, pembaruan tanpa refresh, fokus dan row identity, Pause/Resume, offline recovery, gambar gagal/retry/vector, proteksi form dan browser Back. Pengujian transport terpisah memakai server HTTP/SSE lokal terisolasi dengan mode non-demo, termasuk patch Go per field dan reconnect setelah stream diputus. Ini belum membuktikan integrasi dengan kredensial, database, perangkat atau server produksi.

Pemeriksaan visual dilakukan pada layar desktop/laptop dan mobile. Pengujian memakai Chromium; belum dilakukan sesi pembaca layar manual, uji lintas browser lengkap, atau uji tugas bersama dispatcher nyata. Langkah validasi produk berikutnya adalah meminta operator menemukan satu alert, menelusuri unitnya, dan menyelesaikan input maintenance sambil mengukur keberhasilan tugas dan kesalahan baca data.
