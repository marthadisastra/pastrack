# Current local testing report

3 November 2026. This report covers implemented components, not the full definition of done.

- Root Go unit/integration suite: passed against isolated MySQL 8.0.43, pastrack_test_local. Tests include spool/frame fragmentation/coalescing, observed STT, unknown/ACK handling, UTC, append mask/satellites, v0306 zero-fix handling, byte-preserving spool, raw durability after normalization failure, replay idempotency and documented BLE raw/disconnected NULL liters.
- Domain tests: CFS word decoding/invalid payload, approved monotonic tank calibration/missing/out-of-range/water rejection, sustained fuel drop/disconnect suppression, trip jump/gap handling, circle/polygon geometry.
- V2 HTTP integration: authenticated GET resource inventory, vehicle subroutes, master create with NOT_INSTALLED, excessive limit rejection, asynchronous report creation and authorized CSV download.
- Root Go race suite: passed during implementation; final rerun recorded in command output. Root vet/build: passed.
- Existing web Go packages: passed.
- Existing web Node suite: initially 22/26, then **26/26 passed** after correcting realtime fallback and playback labels.
- New v2 DOM suite: **2/2 passed** (navigation/escaping/NULL/role visibility, keyset/search/error state).
- New and changed JS syntax and local shell syntax checks: passed.
- Explicit local migration: 65 domain tables + 1 migration-ledger table and 109 FKs. Reapply checksum check passed. Additional migration 002 adds derived-record unique keys. No telemetry seed was inserted into dev DB.
- HTTP health checks: local web and local receiver returned 200; TCP listener bound to 127.0.0.1:15001.
- Browser: login page opened and visually inspected; complete authenticated screen-by-screen desktop/mobile QA remains pending.

Not performed: physical GV30CEU/CFS300B test, real fuel tank validation, production migration/deployment, restored legacy data reconciliation, rollback rehearsal, full fleet-scale load test, full 21-screen visual acceptance. No such result is claimed.

## Stitch correction verification

- New v2 DOM coverage expanded from 2 to 9 tests: exact sidebar ordering, all 21 packaged source layouts, all 17 primary routes, five vehicle detail layouts, demo removal, escaping/NULL/ignition, search/cursor/error state, actual unit-create request and disabled unsupported hardware controls.
- PDF/XLSX conversion tests verify real artifacts and preserve blank fuel fields. These are export tests, not sensor calibration/physical-device proof.
- Browser overview visual inspected at mobile/desktop sizes; later local browser access was disabled in Work settings. Full 21-screen screenshot acceptance remains pending.
