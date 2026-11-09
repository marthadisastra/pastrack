# Rebuild implementation report — interim

**Status: full rebuild not complete; local implementation increment delivered.** This file is updated as work continues and is not a declaration of production readiness.

## Created source files

- `cmd/pastrack-db/main.go`
- `cmd/pastrack-local/main.go`
- `database/embed.go`
- `database/generate_schema.py`
- `database/migrations/001_platform.sql`
- `database/migrations/002_derived_keys.sql`
- `database/schema.json`
- `fuel_events_v2.go`
- `geofence_v2.go`
- `internal/database/migrate.go`
- `internal/database/migrate_test.go`
- `internal/fuel/calibration.go`
- `internal/fuel/calibration_test.go`
- `internal/geofence/geometry.go`
- `internal/geofence/geometry_test.go`
- `internal/platform/api.go`
- `internal/platform/assignments.go`
- `internal/platform/calibration.go`
- `internal/platform/integration_test.go`
- `internal/platform/password_test.go`
- `internal/platform/reports.go`
- `internal/platform/server.go`
- `internal/platform/server_test.go`
- `internal/platform/trip_version.go`
- `internal/platform/trips.go`
- `internal/platform/ui/index.html`
- `internal/platform/ui/leaflet-LICENSE.txt`
- `internal/platform/ui/leaflet.css`
- `internal/platform/ui/leaflet.js`
- `internal/platform/ui/login.html`
- `internal/platform/ui/login.js`
- `internal/platform/ui/platform.css`
- `internal/platform/ui/platform.js`
- `internal/platform/users.go`
- `internal/protocol/cfs300b.go`
- `internal/protocol/cfs300b_test.go`
- `internal/protocol/eri.go`
- `internal/protocol/eri_test.go`
- `internal/trip/engine.go`
- `internal/trip/engine_test.go`
- `scripts/local-start.sh`
- `scripts/local-stop.sh`
- `storage_ble_v2.go`
- `storage_v2.go`
- `storage_v2_test.go`
- `tests/platform.test.cjs`

## Modified existing files

- .gitignore: exclude private .local artifacts.
- go.mod/go.sum: bcrypt dependency for new isolated auth entrypoint.
- main.go/storage.go/parser.go/realtime.go: opt-in v2 persistence, safe spool/UTC/fix handling and nonblocking invalidations.
- web/ui/static/realtime.js and tracking-playback.js: fixes validated by existing tests.
- docs/rebuild review documentation: current implementation and protocol evidence notes.

No source files removed. Existing Beego application and capabilities remain available; the new local application is a parallel development path until feature/data parity is verified.

## Database/migration results

Created pastrack_dev_local and pastrack_test_local on separate MySQL 8.0.43 port 33307. Each has 65 domain tables + schema_migration_steps, 109 foreign keys. 001 creates schema/constraints/indexes; 002 adds derived result uniqueness. DDL reapply passed. Source historical records were not migrated; no historical count/association parity is claimed. Local dev backup created privately.

## API/UI/protocol

See API_DOCUMENTATION.md for actual endpoints and restrictions. Local UI implements real SQL data views and selected master/assignment/calibration/report interactions; not all 21 Stitch page compositions are complete. Protocol supports bounded documented STT/ERI paths and raw retention for unsupported layouts. CFS raw height/volume are not presumed to be liters. NOT_INSTALLED physical ignition is enforced by default and excluded from movement inference.

## Validation and limitations

See TESTING_REPORT.md and REBUILD_ISSUES.md. Physical hardware, actual tank curve, complete actual raw variant inventory and restored legacy snapshot are unavailable/unverified. Remaining UI/backend work is tracked separately from these external blockers. No deployment, production migration or destructive schema operation was run.

## Running

Use scripts/local-start.sh; open http://127.0.0.1:18090. Admin password remains in private .local/bootstrap-password. Full steps/environment/backups: LOCAL_DEVELOPMENT.md.

## Latest UI correction

The served local frontend now imports all 21 source HTML layouts and the exact 17-entry sidebar from Desain/. This supersedes the earlier generic-table UI description. Source importer, compiled theme/fonts, runtime bindings, nine DOM tests and PDF/XLSX report conversion are implemented. Full visual equivalence and unsupported hardware/third-party features are not claimed. See STITCH_IMPLEMENTATION.md for current boundaries.
