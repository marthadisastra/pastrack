# Running the local rebuild

Existing application: web/ remains separate. New v2 uses cmd/pastrack-local and PASTRACK_SCHEMA=v2 for receiver persistence. No deployment script is run.

## Start / stop

From repository root:

```sh
./scripts/local-start.sh
```

The script builds binaries, applies only local target migrations and supervises both services in the foreground. Requires Go 1.25+ (verified with 1.27.3), Python 3 and AMPPS MySQL binaries (override MYSQL_BASE if needed). Private MySQL instance uses .local/mysql and loopback port 33307. Existing MySQL at 3306 is not reused. Script checks actual @@datadir before applying migrations.

- Web: http://127.0.0.1:18090
- Web health: http://127.0.0.1:18090/healthz
- Tracker TCP: 127.0.0.1:15001 (local test only)
- Receiver health: http://127.0.0.1:18091/healthz
- Admin username: admin. Generated password remains only in `.local/bootstrap-password` (mode 0600); read locally, do not put it in logs or source. Bootstrap does not overwrite existing accounts.
- Stop application services: `./scripts/local-stop.sh`. MySQL remains available for inspection. Do not delete .local/mysql or spool to reset the application.

## Configuration

DB_HOST/PORT/NAME/USER/PASSWORD configure SQL; new commands reject non-loopback hosts and names not matching pastrack_*_local. Dev schema pastrack_dev_local, tests pastrack_test_local. PASTRACK_SCHEMA=v2 is required for new receiver storage; default persistence remains legacy. GPS_HOST is loopback, GPS_PORT/HEALTH_ADDR/SPOOL_DIR/REALTIME_SOCKET_PATH set by local script. LOCAL_REPORT_DIR optionally selects private CSV storage. LOCAL_BOOTSTRAP_USER/PASSWORD_FILE configure first local account only. RAW_TEXT_LOG defaults off to avoid identifier-bearing payload logs.

No migration runs during application startup. To apply new reviewed migrations explicitly:

```sh
DB_HOST=127.0.0.1 DB_PORT=33307 DB_USER=root DB_NAME=pastrack_dev_local go run ./cmd/pastrack-db
```

Never point this command at production. Current isolated developer MySQL root is confined to the private loopback instance; this is not a deployment configuration.

## Backup / restore

A local development snapshot was created at `.local/backups/pastrack_dev_local.sql` with private permissions. It contains local schema/account state, not a production backup. Preserve it privately.

Use mysqldump transactional backup with routines/triggers/events and set-gtid-purged=OFF. Restore only into a **new separate local schema**; do not overwrite an active/source database. Source production backup, historical count parity and rollback rehearsal are still pending access to a restored local snapshot. Detailed mapping/rollback procedure remains docs/rebuild/DATABASE_MIGRATION.md.

## Validation

```sh
PASTRACK_TEST_DSN='root@tcp(127.0.0.1:33307)/pastrack_test_local?parseTime=true' go test -race ./...
go vet ./...
node --test tests/platform.test.cjs
(cd web && go test ./... && npm test)
```

Create test schema with pastrack-db first. Test fixtures write only to isolated test schema; never use dev/prod schema as the test target. Constructed protocol fixtures are not physical device verification.
