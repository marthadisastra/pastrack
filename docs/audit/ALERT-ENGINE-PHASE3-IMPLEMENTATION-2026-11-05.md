# Phase 3 — Vehicle Identity Resolver & Normalized Telemetry Foundation

Tanggal: **5 November 2026 (WIB)**

## Outcome

Phase 3 mengimplementasikan package read-only `internal/telemetry`. Package ini memetakan GPS melalui historical `vehicle_assignments`, BLE melalui direct historical `ble_sensor_bindings`, memvalidasi GPS receiver hanya sebagai context, dan menghasilkan normalized GPS/fuel telemetry tanpa mengubah ingestion, parser, API, dashboard, atau data source.

Alert evaluator, lifecycle, rule FUEL_LOW/FUEL_DROP/OVER_SPEED/CRASH, notification, worker, dan dashboard alert tidak dibuat.

## 1–3. Files and database migration

File baru:

- `internal/telemetry/types.go`: kontrak resolution, timestamp quality, source facts, dan normalized telemetry.
- `internal/telemetry/resolver.go`: resolver GPS/BLE dan MAC normalization.
- `internal/telemetry/mysql.go`: repository MySQL read-only dengan bounded indexed queries.
- `internal/telemetry/normalizer.go`: GPS/fuel adapters.
- `internal/telemetry/resolver_test.go`: unit tests untuk status resolution, receiver validation, concurrency, dan MAC.
- `internal/telemetry/normalizer_test.go`: unit tests temporal integrity dan normalized values.
- `internal/telemetry/integration_test.go`: real MySQL integration tests pada isolated `pastrack_test_local`.
- Dokumen ini.

Tidak ada file existing yang diubah. Tidak ada migration baru karena schema existing sudah cukup:

- IMEI dan MAC memiliki unique lookup.
- `ix_vehicle_assignments_history(device_id, starts_at, ends_at)` tersedia.
- `ix_ble_sensor_bindings_history(sensor_id, starts_at, ends_at)` tersedia.
- GPS correlation memakai unique/index prefix `gps_positions(raw_message_id, sample_index)`.

Tidak ada evaluation-job table karena Phase 3 belum memiliki producer/worker dan scope memperbolehkannya hanya jika dibutuhkan fondasi integrasi. Queue/outbox tetap ditunda ke Phase 4.

## 4–6. Resolver architecture and flows

Repository memisahkan SQL dari decision logic sehingga resolver dapat diuji tanpa mock SQL. Normalizer hanya membaca persisted facts lalu meminta resolver menentukan identity.

GPS flow:

1. Cocokkan internal device ID dan/atau IMEI dengan `gps_devices`.
2. Tolak event time nil/zero sebagai `INVALID/EVENT_TIME_INVALID`.
3. Cari maksimal dua assignment pada interval `[starts_at, ends_at)` menggunakan GPS event time.
4. Nol row → `UNASSIGNED`; satu → `RESOLVED`; dua → `AMBIGUOUS`.
5. Device yang kemudian inactive/soft-deleted tetap dapat di-resolve secara historis.

BLE flow:

1. Normalisasi MAC menjadi 12 uppercase hex dan cocokkan dengan `ble_sensors`.
2. Tanpa event time yang layak, hasil `UNASSIGNED/EVENT_TIME_UNAVAILABLE`.
3. Cari direct `ble_sensor_bindings` pada event time.
4. Direct binding menyediakan vehicle. GPS receiver tidak pernah menyediakan vehicle pengganti.
5. Receiver harus ada; bila binding menentukan device, device harus sama.
6. Jika receiver mempunyai assignment pada event time, vehicle context harus sama dengan direct binding.
7. Binding/receiver overlap atau mismatch menghasilkan `AMBIGUOUS`.
8. Sensor/device yang kemudian soft-deleted tetap mempertahankan historical resolution.

## 7. Timestamp quality

Empat nilai sesuai requirement:

| Quality | Implementasi |
|---|---|
| `ORIGINAL_DEVICE_TIME` | `gps_positions.gps_time` untuk GPS atau `sensor_telemetry.sensor_time` untuk BLE |
| `CORRELATED_TIME` | Tepat satu distinct GPS time dari raw message GTERI yang sama |
| `RECEIVED_TIME_ONLY` | Hanya received time tersedia; `event_time` sengaja tetap nil |
| `UNKNOWN` | Tidak ada bukti waktu yang dapat dipakai |

GPS time GTERI tidak dipromosikan menjadi timestamp asli CFS300B. Normalizer mengeksposnya sebagai correlated event time. Ia tidak menulis `fuel_readings.reading_at`.

## 8. Historical integrity

Integration test membuktikan resolver/normalizer tidak mengubah jumlah `gps_positions`, `fuel_readings`, atau `vehicle_assignments`; `reading_at` tetap NULL dan `fuel_percent` tetap identik. Reassignment baru tidak mengubah hasil resolusi event sebelum cutover. Late-arriving GPS tetap masuk vehicle historical.

Konflik requirement: 857 posisi unassigned telah dihapus dari production atas instruksi pengguna sebelum Phase 3. Phase 3 tidak mengembalikan atau merekayasa posisi tersebut. Baseline live read-only setelah implementasi lokal:

- GPS positions tanpa vehicle: **0**.
- Fuel readings: **13.226**.
- Fuel readings dengan `reading_at NULL`: **13.226**.
- Alert events: **0**.

Angka fuel bertambah karena ingestion live aktif. Package Phase 3 tidak dideploy dan tidak dapat memengaruhi angka production.

## 9–11. Test results

Unit/race:

```text
go test -race ./internal/telemetry
ok pastrack/internal/telemetry
```

Integration memakai MySQL 8.0.43 terisolasi di `127.0.0.1:33307/pastrack_test_local` dan mencakup:

- GPS by IMEI/device ID, unknown device, missing assignment, historical assignment, reassignment, overlap, invalid time, late arrival.
- BLE by MAC, unknown sensor, missing/historical/reassigned/overlapping binding, receiver valid/mismatch, unknown/correlated time.
- Soft-deleted source identity tetap historis.
- Fuel timestamp tidak direkayasa, percentage tetap konsisten.
- Concurrent resolution stabil.
- `EXPLAIN` mengiklankan kedua history indexes.

```text
go test -run TestMySQLHistoricalResolversAndNormalization -v ./internal/telemetry
PASS

go test -cover ./internal/telemetry
coverage: 81.3% of statements (with isolated DB)
```

Regression suite dengan database test terisolasi:

```text
PASTRACK_TEST_DSN=... go test -count=1 ./...
PASS: pastrack, database, fuel, geofence, platform, protocol,
      remoteoutput, telemetry, trip

go vet ./...
PASS
```

## 12. Example results

Resolved historical GPS:

```json
{
  "status": "RESOLVED",
  "reason": "HISTORICAL_ASSIGNMENT",
  "vehicle_id": 40,
  "assignment_id": 30,
  "device_id": 10
}
```

Resolved BLE with correlated time and matching receiver context:

```json
{
  "kind": "FUEL_READING",
  "sensor_id": 20,
  "mac": "AABBCCDDEEFF",
  "receiver_device_id": 10,
  "vehicle_id": 40,
  "binding_id": 50,
  "fuel_percent": 42.5,
  "timestamp_quality": "CORRELATED_TIME",
  "resolution_status": "RESOLVED"
}
```

BLE without source/correlated time:

```json
{
  "event_time": null,
  "timestamp_quality": "RECEIVED_TIME_ONLY",
  "resolution_status": "UNASSIGNED",
  "resolution_reason": "EVENT_TIME_UNAVAILABLE"
}
```

## 13. Known limitations and remaining risks

- Package belum dipanggil ingestion/API; ini disengaja agar Phase 3 tidak mengubah behavior existing. Integrasi asynchronous dilakukan pada Phase 4.
- Current BLE writer masih dapat menaruh vehicle berdasarkan gateway fallback. Normalizer Phase 3 mengabaikan stored association sebagai otoritas dan me-resolve ulang dari direct historical binding.
- Sebagian besar historical fuel tidak mempunyai direct `binding_id`/sensor time; hasil dapat `UNASSIGNED` walaupun stored fuel vehicle terisi.
- `CORRELATED_TIME` tetap bukan waktu ukur asli sensor dan belum boleh dipakai sebagai dasar Fuel Drop production tanpa keputusan policy.
- Assignment/binding overlap masih mungkin ada karena database belum mempunyai exclusion constraint temporal; resolver fail-closed sebagai `AMBIGUOUS`.
- Resolver melakukan jumlah query tetap per fact (bounded, bukan N+1 internal), tetapi batch API belum dibuat. Phase 4 harus melakukan batch/worker claim dengan ukuran terbatas.
- Tidak ada migration rollback karena tidak ada migration. Semua perubahan source adalah additive dan dapat dilepas dengan menghapus package yang belum terintegrasi.

## Deployment status

Tidak ada deployment, migration production, API change, dashboard change, atau write production. Ini mengikuti strict execution rule Phase 3 dan mengesampingkan kebijakan deploy-default project untuk fase ini.

**Phase 3 selesai. Berhenti sebelum Phase 4.**
