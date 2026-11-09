# Rebuild changelog

## 3 November 2026 — local implementation increment

- Added database/schema.json, deterministic DDL generator and versioned 001/002 migrations; created isolated MySQL data directory, dev/test schemas and development-only guarded migration command.
- Added cmd/pastrack-local and internal/platform: session auth, role checks, parameterized/cursor query resources, master mutations, assignment history, user administration, immutable approved calibration curves, report job/CSV download and SSE invalidation.
- Added protocol/fuel/trip/geofence domain packages and tests. Official v0306 p193–194 clarifies GNSS Accuracy zero means failed fix/last-known coordinates; corrected v2 validity accordingly.
- Added storage_v2.go, storage_ble_v2.go, fuel_events_v2.go, geofence_v2.go behind PASTRACK_SCHEMA=v2; old persistence remains available. Raw receipt commit precedes derived transaction. RESP/BUFF equivalent facts deduplicate without deleting raw receipts.
- Updated main.go: stable spool receipt ID, base64 byte copy for lossless non-UTF8 retention, invalid spool quarantine, explicit blocked-assignment quarantine, redacted persistence errors and local v2 connection guard. Raw text logging now requires RAW_TEXT_LOG=on. Receiver default bind changed to loopback for local work; real deployment must explicitly configure GPS_HOST later.
- Updated realtime.go to avoid blocking ingestion when invalidation queue is full/unavailable.
- Added local UI assets with read/write interactions and tests, plus start/stop scripts. The new application does not replace web/ or pretend full design parity.
- Fixed legacy web/ui/static/realtime.js fallback invalidation and tracking-playback.js status labels exposed by four failing existing tests.
- Added .local/ ignore rule. Runtime passwords, database files, logs, backups and binaries remain private local artifacts.
- No source file removed. No production connection, production environment update, destructive schema operation or deployment performed.

## Stitch correction increment

- Replaced served v2 index with the actual exported sidebar/header and 21 page compositions. Added deterministic HTML import, local compiled Tailwind theme, local Inter/Material Symbols fonts and runtime bindings.
- Added `stitch.js`, `stitch.css`, `stitch-pages.json`, `stitch-resources.js`, font files/licenses and compiled CSS. Old platform.js/platform.css remain unserved reference files; the new index does not load them.
- Corrected selected-vehicle detail queries and secondary registry sources. Added playback/seek, geofence drawing, measured calibration dialogs and contextual inventory actions.
- Added report PDF/XLSX conversion and tests using the same established libraries already used by the legacy web application. GPS exports use currently loaded real positions.
- Replaced v2 DOM tests with nine tests covering exact menu order, all imported layouts, navigation/detail routes, escaping/NULL values, filters/cursors, real register-unit mutation and unsupported command reasons.
- Browser visual continuation blocked by Work settings disabling local browser use; full screen-by-screen visual acceptance remains open.

## Inactive Fleet Dashboard correction and deployment request

- Fixed the initially-active exported dashboard item losing its text/icon color on deselection. Added uniform inactive classes, scoped active/inactive colors and regression test.
- User authorized live web/database/receiver replacement. Initial SSH check failed; deployment has not occurred. Required server verification/backups/migration/cutover checks are documented in DEPLOYMENT_READINESS.md.

## Server staging and actual-protocol validation

- SSH access restored using the user-identified existing key. Created private verified snapshot/restore and isolated server staging schema; existing live application and source schema remain unchanged.
- Added stage/backfill/private-runtime/verification tools in deploy/rebuild. Server validation reports contain only metadata/counts and endpoint status.
- Added coordinate precision widening migration 003 after rehearsal exposed rounding; source coordinate/time parity checked separately.
- Added explicitly guarded production-runtime target configuration (separate from local migration authorization), disabled local account bootstrap in production mode and enabled secure cookie configuration behind HTTPS proxy.
- Added real sanitized 0x8104 ERI fixture, documented RAT/band handling and opaque exact-match CAN retention. Fixed legacy discovered sensor MAC resolution while preserving administrative assignments.
- Verified 35 authenticated reads and actual-payload TCP replay through new schema with raw receipt duplication preserved and normalized BLE facts deduplicated. Physical device test and final live cutover remain open.

## Live cutover for user acceptance

- Per explicit user instruction, deployed latest web UI and raw receiver together to the existing aaPanel projects and ports, with new separate database `pastrack_v2_20261103_r2`.
- Initial count check exposed imported multi-sample raw references; aborted activation and restored legacy services. Corrected stable legacy sample ordinals, repeated into a fresh target and verified all 428 positions/2971 receipts before activation.
- Preserved original DB, configurations, binaries and a fresh private source snapshot. Kept old login hashes; v2 requires a new login session, without resetting credentials.
- Public authenticated shell/API/style smoke tests and both private service health checks passed. Corrected the login footer to remove a development-only claim in the production UI.

## Atomic Calibrate BLE registration flow

- Added available-sensor/profile query and transactional vehicle BLE registration API. Gateway is resolved from server observations, not user-entered ID.
- Register Unit on vehicle calibration now opens Sensor ID/Tank Profile ID dropdowns, shows gateway/last-received state and can create an unassigned tank profile when needed.
- Added migration 004 to allow unassigned tank profiles; no table/data removal.
- Added concurrent/exclusive/idempotent SQL tests and imported-layout DOM form test. Existing physical ignition and calibration safeguards remain intact.

## Trip History date range

- Replaced the imported static Today date chip with From/To date controls and Apply in Trip History, vehicle Trip History and playback.
- Date-only inputs represent complete Asia/Jakarta calendar days, converted to UTC for API filters; same-day selection is valid and the last day is inclusive through 23:59:59.999 WIB.
- Reversed/invalid/over-366-day ranges are rejected without querying. Applying a range resets pagination/playback and keeps the chosen range when switching vehicles.
- Added DOM tests for WIB boundary conversion, invalid range rejection and selected-vehicle/range persistence.

## Local source linked to live data

- Connected the local application through a loopback SSH tunnel to the same live schema. The user explicitly requested full live parity; the initially proposed read-only mode is not enabled.
- Same live account/roles used locally. Reads/authentication use live SQL; mutations, report files and realtime use the authoritative live service to avoid separate laptop state.
- Added guarded live upstream routing and optional read-only support (default off); credentials stay in private ignored configuration. No local receiver or automatic migration runs in this mode.
- Verified identity, live vehicle data, shared reports/realtime and identical invalid-mutation validation without changing business data.
