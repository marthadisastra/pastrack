# Uji batang CFS300B dalam galon air — 2 November 2026

Sumber: 56 laporan `GTERI` yang benar-benar tersimpan di database produksi dari 15:02:41–15:07:16 WIB; [CSV seluruh sampel](cfs300b-water-immersion-run-2026-11-02.csv). IMEI `861252080093110`, sensor `E04E7A648774`.

| Kondisi uji | Waktu WIB | Kandidat pembacaan 16-bit |
|---|---|---:|
| Batang kering, sebelum bergerak | 15:02:41–15:03:11 | 258–260 |
| Awal batang dimasukkan | 15:03:16–15:03:31 | 262 → 442 |
| Mendekati batas akhir | 15:04:21–15:04:56 | 4457 → 4883 |
| Batas akhir, mulai stabil | 15:05:16–15:06:21 | 4950 → 4992 |
| Batas akhir, stabil | 15:06:41–15:07:16 | 4991–4992 |

Pemeriksaan lanjutan pada 15:09:16 WIB menunjukkan `4996`, perubahan kecil sekitar 0,08% dari `4992` saat batang tetap pada batas akhir.

Kandidat pembacaan adalah dua byte pada field 35 `GTERI`, misalnya `0x0104` = 260 saat kering dan `0x1380` = 4992 saat terendam pada batas akhir uji. Dua byte berikutnya bergerak hampir sebanding (`26` → `499`). Byte pertama field tersebut berubah saat posisi cairan tetap dan belum diidentifikasi. Ini **inferensi dari pengamatan**, bukan definisi field resmi Queclink.

Uji ini membuktikan respons yang kuat dan stabil terhadap perubahan panjang batang terendam dalam **air**. Ia tidak membuktikan nilai 0–100% volume tangki kendaraan, satuan panjang, atau liter BBM. Dua titik lama yang diperkirakan sebagai 15% dan 75% tinggi air berasal dari percobaan sebelumnya dan tidak dipakai untuk membuat kurva pada run ini. Untuk kalibrasi, kumpulkan kedalaman batang yang diukur dalam cm pada beberapa titik tetap, lalu ulangi pada tangki/BBM sebenarnya atau peroleh spesifikasi field resmi CFS300B/GV30CEU.
