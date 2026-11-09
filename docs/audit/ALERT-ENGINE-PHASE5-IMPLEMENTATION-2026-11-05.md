# PasTrack Alert Engine Phase 5 Implementation

Date: 2026-11-05  
Scope: FUEL_LOW, FUEL_DROP readiness, and OVER_SPEED only

## 1. Implemented Rules

| Rule | Operational status | Input | Scope |
|---|---|---|---|
| FUEL_LOW | READY | Valid calibrated CFS300B fuel percentage | vehicle + BLE sensor |
| FUEL_DROP | NOT_READY | Requires CFS300B `ORIGINAL_DEVICE_TIME` | vehicle + BLE sensor |
| OVER_SPEED | READY | Valid Queclink GPS fix and GPS device time | vehicle + GPS device |

CRASH remains NOT_READY. Ignition remains NOT_INSTALLED / UNKNOWN and is not evaluated.

## 2. Registration and Configuration

Migration 011 adds a unique rule code to the existing `alert_rules` table and seeds centralized JSON configuration. It does not create another engine, queue, lifecycle, or alert table.

- FUEL_LOW: trigger `<10%`, recovery `>=15%`, two trigger samples, two recovery samples, five-minute receipt freshness.
- FUEL_DROP: drop `>=8` percentage points within five minutes, minimum three samples, three stable recovery observations. It is stored disabled with readiness `NOT_READY_REQUIRES_ORIGINAL_SENSOR_TIME`.
- OVER_SPEED: speed `>80 km/h` continuously for at least ten seconds, recovery `<=75 km/h`, maximum sample gap 15 seconds, maximum credible speed 250 km/h.

`alertrules.LoadEnabled` loads enabled definitions and returns the existing Phase 4 registry. FUEL_DROP has a complete evaluator and unit coverage, but is intentionally excluded from the operational registry until genuine sensor measurement time exists.

## 3. Timestamp and WIB Policy

When CFS300B has no `sensor_time`, the existing Phase 3 normalizer uses `gps_positions.gps_time` from the same raw Queclink message. The result is explicitly marked `CORRELATED_TIME`.

The instant is stored and compared in UTC. For operators it is represented in `Asia/Jakarta` / WIB (UTC+7). No seven-hour offset is written into the database, preventing a double timezone shift.

Example:

```text
Queclink GPS instant: 2026-11-05 01:00:00 UTC
Operator display:     2026-11-05 08:00:00 WIB
timestamp_quality:    CORRELATED_TIME
```

This correlated GPS time is eligible for historical BLE identity resolution and FUEL_LOW freshness/ordering. It is not the CFS300B measurement clock and therefore is never eligible for FUEL_DROP interval calculation. RECEIVED_TIME_ONLY and UNKNOWN also remain ineligible for FUEL_DROP. No historical timestamp was backfilled.

## 4. Trigger, Recovery, and Noise Controls

### FUEL_LOW

- Requires valid sensor identity, resolved vehicle, calibrated percentage in `[0,100]`, `valid` reading status, and fresh receipt.
- Requires two consecutive low samples within the same sensor scope.
- Uses hysteresis and requires two consecutive samples at or above 15% to recover.
- Duplicate input events do not increment counters.
- Sensors and vehicles never share evaluator state.

### FUEL_DROP

- Accepts calculations only for `ORIGINAL_DEVICE_TIME`.
- Requires strictly increasing, nonduplicate samples from one sensor scope.
- Maintains a bounded five-minute window and at least three samples.
- Does not label the condition as theft; the alert text states an anomaly.
- Rejects late/out-of-order calculations and requires stable observations for recovery.
- Operationally NOT_READY because current CFS300B readings do not provide original sensor time.

### OVER_SPEED

- Requires resolved vehicle, device identity, valid GPS fix, valid coordinates, plausible nonnegative speed, fresh receipt, and ORIGINAL_DEVICE_TIME.
- A single sample cannot trigger an episode.
- Sampling gaps over 15 seconds reset continuity; no interpolation is performed.
- Speeds of 80 km/h or lower do not begin the duration timer.
- An active episode recovers at 75 km/h or lower.

## 5. Episode, Provenance, and Lifecycle

All evaluators implement the existing Phase 4 `RuleEvaluator`. State remains serialized by `(rule_id, vehicle_id, scope_key)`. Phase 4 performs episode deduplication, source attachment, audit, and automatic recovery in the same transaction.

BLE scope includes sensor ID and GPS scope includes device ID. Consequently, different sensors are not mixed even when mounted on one vehicle. Historical alert `vehicle_id` is frozen and does not follow later device reassignment.

## 6. Actual Flows

### FUEL_LOW

```text
CFS300B: 9%, then 8%; sensor_time absent
-> same-message Queclink gps_time, CORRELATED_TIME (displayed WIB)
-> Phase 3 resolves direct BLE binding to vehicle_id
-> Phase 4 intake + queue
-> FUEL_LOW sees two valid low samples
-> one ACTIVE vehicle episode
-> two valid readings >=15%
-> same episode RESOLVED
```

### FUEL_DROP

```text
CFS300B: 75% -> 70% -> 65%; only correlated GPS time
-> normalized events retain CORRELATED_TIME
-> FUEL_DROP is not in enabled registry / evaluator also refuses interval state
-> no alert
```

With three future samples carrying genuine, ordered ORIGINAL_DEVICE_TIME, the tested evaluator can calculate the five-minute window after explicit administrative enablement.

### OVER_SPEED

```text
GPS: 81 km/h at T0, 85 km/h at T+10s; valid fix/time
-> Phase 3 resolves historical GPS assignment to vehicle_id
-> Phase 4 intake + queue
-> OVER_SPEED proves continuous minimum duration
-> one ACTIVE vehicle episode
-> GPS speed 74 km/h
-> same episode RESOLVED
```

## 7. Database and File Changes

- `database/migrations/011_vehicle_alert_rules.sql`: rule codes, definitions, centralized configuration, readiness flag.
- `database/rollbacks/011_vehicle_alert_rules.sql`: development-only rollback.
- `internal/alertrules/rules.go`: configuration loader and three evaluator implementations.
- `internal/alertrules/rules_test.go`: deterministic boundary, quality, stale, noise, duplicate, gap, and recovery tests.
- `internal/alertrules/integration_test.go`: actual Phase 4 intake/queue/worker/episode integration.
- `internal/telemetry/types.go`: exposes existing persisted fuel validity in normalized telemetry.
- `internal/telemetry/normalizer.go`: carries that validity without recalibration.
- `internal/telemetry/normalizer_test.go`: verifies correlated GPS time and its WIB representation.

Migration 011 was applied only to isolated MySQL `127.0.0.1:33307/pastrack_test_local`. No production migration or deployment occurred.

## 8. Test Results

Using the isolated test database:

- `go test ./...`: PASS.
- `go test -race ./...`: PASS.
- `go vet ./...`: PASS.

Coverage includes threshold boundaries, minimum sample counts, hysteresis, stale and invalid readings, timestamp qualities, duplicate/out-of-order samples, sensor scopes, GPS validation, sampling gaps, integration with Phase 4 worker, episode creation, recovery, provenance, and regression packages.

## 9. Known Limitations

- Live CFS300B data remains unsuitable for FUEL_DROP until the sensor supplies a trustworthy measurement timestamp. GPS correlation does not remove this limitation.
- Phase 4/5 provides the durable worker and registry loader but does not deploy or start a new production worker process under this phase's explicit no-deployment boundary.
- Thresholds are global definitions; per-vehicle overrides are not implemented.
- No notification, new API, frontend, dashboard, CRASH, or ignition feature was added.

Phase 5 stops here and awaits Phase 6 instructions.
