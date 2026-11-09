# PasTrack Phase 5.1 — FUEL_DROP Correlated GPS Time Reassessment

Date: 2026-11-05  
Verdict: **READY_WITH_VALIDATED_CORRELATED_TIME** for prospective eligible telemetry only

## 1. Correlation Evidence

The correlation is structurally provable in the current GV30CEU GTERI path:

1. `DecodeERI` parses the GPS position group and CFS300B accessory group from one immutable GTERI payload.
2. The receipt is persisted once in `device_raw_messages`.
3. GPS facts are written to `gps_positions` with that receipt's `raw_message_id`.
4. CFS300B facts are written to `sensor_telemetry` with the same `raw_message_id`.
5. `fuel_readings.telemetry_id` links the calibrated reading to that sensor telemetry row.
6. Phase 3 only returns correlated time when exactly one distinct, non-null, validated GPS time with a valid fix exists on that same raw message.
7. Equal wire payloads are retained as receipts but only one produces derived facts because `message_processing_keys` deduplicates normalized processing.

The normalizer does not query a vehicle's latest GPS position. It queries only positions carrying the fuel telemetry's exact `raw_message_id`.

One GTERI may contain multiple accessory readings. Each accessory has an index and sensor identity but shares the report GPS group. Rule state remains scoped to sensor, so readings from different sensors are never mixed.

## 2. What the Evidence Does Not Prove

The payload proves GPS and BLE values were reported together by the GV30CEU. It does not prove the internal CFS300B sampling instant. A sensor observation may have been held before the tracker sent GTERI, and buffered tracker reports may arrive substantially later.

Therefore:

- GPS correlation remains `CORRELATED_TIME`; it is never promoted to `ORIGINAL_DEVICE_TIME`.
- Correlated samples receive MEDIUM confidence.
- Only genuine sensor time receives HIGH confidence.
- Unproved, stale, delayed, duplicate, or ambiguous evidence receives INSUFFICIENT confidence and cannot trigger.

## 3. Read-only Live Evidence

The production database was queried read-only; no migration or mutation was performed.

Snapshot findings:

| Measurement | Count |
|---|---:|
| All fuel readings | 13,924 |
| Valid calibrated readings | 12,399 |
| Readings with sensor_time | 0 |
| Valid readings with exactly one same-raw GPS time | 11,777 |
| Valid readings with no same-raw GPS time | 622 |
| Valid readings with ambiguous same-raw GPS times | 0 |
| Valid readings with valid GPS fix and lag −60…300 seconds | 3,425 |
| Same-raw readings with lag over 300 seconds | 7,811 |

Lag distribution among valid readings with a same-raw GPS position:

- 0–60 seconds: 3,600
- 61–300 seconds: 368
- 301–3,600 seconds: 677
- over 3,600 seconds: 7,134

This confirms delayed data is a material risk. Eligibility is consequently much narrower than “has a same-raw GPS time.” The 3,425 individually eligible observations were not backfilled, replayed, or treated as alert episodes.

## 4. Timestamp Policy

| Quality | Eligibility | Confidence |
|---|---|---|
| ORIGINAL_DEVICE_TIME with `SENSOR_TIME` evidence | Allowed | HIGH |
| CORRELATED_TIME with validated same-raw evidence | Allowed | MEDIUM |
| RECEIVED_TIME_ONLY | Forbidden | INSUFFICIENT |
| UNKNOWN | Forbidden | INSUFFICIENT |

For correlated samples, the evaluator requires all of the following:

- event time present;
- `timestamp_source=SAME_RAW_GPS_TIME`;
- matching positive `raw_message_id` in normalized payload and intake event;
- matching fuel reading, sensor telemetry, receiver device, sensor, and vehicle identities;
- valid calibrated fuel percentage;
- valid/validated GPS fix selected by the Phase 3 repository;
- receipt freshness;
- GPS-to-receipt lag between −60 and 300 seconds;
- strictly increasing timestamp;
- no repeated event or raw message;
- gap from the preceding accepted sample no greater than 180 seconds.

UTC remains the stored instant. WIB is only the `Asia/Jakarta` presentation of that instant. `received_at` is never promoted to event time.

## 5. Rule Behavior

FUEL_DROP retains:

- minimum net drop: 8 percentage points;
- window: 5 minutes;
- minimum samples: 3;
- severity: WARNING;
- sensor-specific scope;
- stable recovery observations: 3.

A trigger contains initial/final percentage, drop, duration, first/last measurement time, confidence, sensor ID, receiver device ID, and vehicle ID. For MEDIUM confidence the description explicitly states:

> Indikasi penurunan BBM berdasarkan waktu GPS yang berkorelasi dengan pembacaan sensor.

It never states theft as fact.

The evaluator state retains each sample's timestamp quality, event ID, telemetry ID, raw-message ID, time, and percentage. On trigger, Phase 4 now attaches all contributing normalized input events to `alert_event_sources`, not only the final sample.

## 6. Deduplication and Reassignment Protection

- Duplicate normalized event IDs are ignored.
- A second reading referring to an already accepted raw-message ID is ignored.
- Non-monotonic timestamps are ignored.
- Excessive sampling gaps reset the candidate window.
- State scope includes sensor ID, while Phase 4 also keys by rule and frozen vehicle ID.
- Phase 3 resolves the historical BLE binding and receiver assignment at correlated event time.
- Later sensor or vehicle reassignment cannot rewrite an existing alert's vehicle identity or provenance.

## 7. Database and Code Changes

- `database/migrations/012_fuel_drop_correlated_time.sql`: enables FUEL_DROP version 2 with correlated-time guardrails.
- `database/rollbacks/012_fuel_drop_correlated_time.sql`: returns it to the disabled Phase 5 state.
- `internal/telemetry/mysql.go`: same-raw correlation now requires valid, validated GPS fix.
- `internal/telemetry/types.go`: normalized fuel carries raw-message identity and timestamp evidence source.
- `internal/telemetry/normalizer.go`: emits `SENSOR_TIME`, `SAME_RAW_GPS_TIME`, or `RECEIVED_TIME_ONLY` evidence without changing quality.
- `internal/alertengine/types.go`: carries raw-message provenance and related contributing event IDs.
- `internal/alertengine/engine.go`: attaches all related trigger samples transactionally.
- `internal/alertrules/rules.go`: supports validated correlated/mixed samples and HIGH/MEDIUM/INSUFFICIENT confidence.
- Tests cover original, correlated, mixed, forbidden qualities, mismatched raw IDs, delayed observations, duplicate raw/event, out-of-order time, gap reset, thresholds, recovery, new state, and full Phase 4 provenance.

Migration 012 was applied only to `pastrack_test_local`. Production remains unchanged.

## 8. Verification

Executed against isolated MySQL `127.0.0.1:33307/pastrack_test_local`:

- `go test ./...`: PASS
- `go test -race ./...`: PASS
- `go vet ./...`: PASS

FUEL_LOW and OVER_SPEED behavior remained green. No parser, BLE ingestion, historical fuel row, `reading_at`, CRASH, ignition, API, frontend, or production deployment change was made.

## 9. Residual Risks

1. MEDIUM confidence cannot prove the CFS300B's internal sample instant.
2. Large delayed-report population means many readings are correctly rejected.
3. A valid GPS fix and short report lag reduce but cannot eliminate sensor-side caching risk.
4. Only prospective events entering the centralized engine are evaluated; historical readings are not replayed.
5. Operational production activation still requires explicit approval, migration, worker runtime wiring/verification, monitoring, and rollback readiness.

Phase 5.1 stops here pending approval.
