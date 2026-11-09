# Rentang tanggal riwayat perjalanan

Halaman Trip History, tab Trip History kendaraan dan playback menyediakan pilihan Dari dan Sampai. Pilih tanggal mulai dan tanggal selesai, lalu klik Terapkan. Rentang awal adalah hari ini menurut Asia/Jakarta, bukan tanggal contoh dari desain.

Kedua tanggal bersifat inklusif. Contoh 1–2 November 2026 berarti 1 November pukul 00.00.00 sampai 2 November pukul 23.59.59.999 WIB. API menerima UTC sehingga awalnya 31 Oktober 2026 pukul 17.00 UTC dan akhirnya 2 November pukul 16.59.59.999 UTC. Hasil tidak bergantung pada zona waktu browser.

Tanggal selesai tidak boleh sebelum tanggal mulai dan rentang maksimal 366 hari. Saat rentang diterapkan, pagination dimulai kembali dan playback dihentikan untuk memuat hasil baru. Pilihan dipertahankan ketika berpindah kendaraan. Jika tidak ada posisi pada rentang tersebut, halaman menampilkan status kosong tanpa membuat rute contoh.

Riwayat yang tampil mengikuti data tersedia, asosiasi kendaraan dan batas pagination API. Pemilihan tanggal tidak membuat fix GPS yang belum ada dan tidak mengubah ignition NOT_INSTALLED.
