# Remote control — status persiapan

Acuan GV30CEU v0306 hal134–137: AT+GTDOS mengendalikan output digital, bukan remote starter mandiri. Output1 latched. Output aktif/inaktif tergantung wiring/polaritas relay, bukan otomatis berarti mesin ON/OFF. ACK penerimaan command bukan bukti mesin benar-benar berubah.

Source lokal:
- Pembaca ignition mencakup GTIGL/GTVGL, GTIGN/GTIGF, GTVGN/GTVGF.
- Status tetap sampai event berikutnya. Query memilih waktu asli Send Time, lalu penerimaan/ID untuk tie; buffer lama tidak mengalahkan laporan baru dengan waktu asli lebih akhir.
- Switch remote pada setiap row vehicle/dashboard disiapkan dalam kondisi disabled dengan label Remote belum dikonfigurasi.
- Tidak ada perintah fisik dikirim. Tidak ada asumsi bahwa checkbox OFF berarti output/mesin OFF.

Informasi wajib sebelum implementasi command aktif: fungsi remote (starter inhibit atau start/stop module), relay terpasang, output ID, polaritas active/inactive, dan konfigurasi/password device yang sesuai. Diminta melalui pertanyaan pengguna, jawaban belum diterima.

Belum implementasi pengiriman command, antrean/ACK/end-to-end hardware. Belum deploy. Jangan menyebut fitur remote selesai100%.
