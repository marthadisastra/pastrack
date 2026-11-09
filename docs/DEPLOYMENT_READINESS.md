# Deployment request — 3 November 2026

User explicitly authorized replacing the live web application, database and Go raw receiver. Earlier no-deploy instruction is superseded by that authorization. This file records readiness, not a completed deployment.

## Verified change

Inactive Fleet Dashboard lacked the inactive text class because it was the originally selected item in the HTML export. The importer now gives every menu entry the complete inactive classes. A scoped CSS rule sets inactive text to rgba(239,241,243,.8), active text to #eeefff and icons to inherit. A regression test covers the initially active item.

## Access blocker

`ssh -o BatchMode=yes root@76.13.195.48` failed with `Permission denied (publickey,password)`. No live configuration, service, database or binary has been changed. Need a working existing SSH alias/key path or restored authorized SSH access. Do not put private key/password contents in this report or chat.

## Required safe cutover sequence

1. Read actual aaPanel service commands, MySQL/MariaDB version, schema, active database/user grants and receiver config on the server. Inspect full actual raw variants privately; redact identifiers in diagnostics.
2. Make consistent database backup and private configuration/binary/spool backups. Verify backup and restore into an isolated staging schema. Do not copy secrets to public build assets.
3. Apply versioned schema to a distinct target database; do not overwrite/drop the existing source. Test actual server-engine compatibility before application switch.
4. Implement/rehearse source-to-target mapping using current DDL and PK ledger. Preserve raw bytes, passwords, sessions policy, photos and assignment provenance. Validate counts/FKs/timestamps; quarantine unresolved historic assignments instead of inferring current ownership backwards.
5. Resolve the current development-only runtime/schema guards through a separately reviewed production configuration path. The current local command intentionally rejects non-local database names. Do not deploy it with dev credentials or rename a production schema to bypass the safeguard.
6. Start staged web+receiver against target on private non-conflicting ports. Validate authenticated page actions and TCP→durable raw→parser→target SQL→API→SSE. Unsupported protocol/features must remain clearly unavailable; no mock hardware success.
7. Stop writes briefly, drain/copy spool, apply final delta and reconcile. Atomically replace matching executables and switch both web and receiver to the same new schema. Preserve old binaries/config/database for rollback.
8. Verify public web/auth, private health, actual TCP listener and increasing raw/derived target counts. Verify old/source data is untouched. Keep event-time/raw timestamp semantics and NOT_INSTALLED ignition.
9. Rollback must account for any new raw/master changes accepted after cutover; do not switch back and lose target-only data. Use tested reverse delta or roll forward.

## Incomplete acceptance items

Full feature parity cannot be certified from the current tests: several mockup hardware/external-service controls are explicitly unavailable; complete legacy backfill/reverse-delta rehearsal, physical-device validation and all-page visual acceptance remain pending. Access restoration alone does not make these checks pass. See REBUILD_ISSUES.md and STITCH_IMPLEMENTATION.md.

## Access restored and staging executed

SSH access succeeded with the existing `imsa_agent_deploy_ed25519` key identified by the user. The earlier access blocker is resolved.

- Server engine verified: MariaDB 10.11.10. Existing live services remain `/opt/pastrack/pastrack` and `/www/wwwroot/pastrack/pastrack-web` under user www.
- Full source snapshot, original binaries and runtime configs backed up privately under `/root/pastrack-rebuild-20261103/backup/`; compressed SQL was read/verified and restored into `pastrack_legacy_restore_20261103` with exact per-table counts.
- Versioned migrations 001–003 applied to `pastrack_v2_stage_20261103` with 109 FKs. Migration 003 widens GPS coordinate precision to preserve source precision rather than rounding it.
- Backfill rehearsal preserves all 2971 raw receipts byte-for-byte and migrates 428 GPS positions, 96 events and the source master/auth/audit/calibration/inbox records. One push credential remains quarantined with its original retained privately; no plaintext is mislabeled encrypted.
- Two private staging services are running: web 127.0.0.1:18092; receiver 127.0.0.1:15002 / health 18093. No public nginx or live TCP routing was changed.
- Runtime now supports an explicitly selected production v2 target through PASTRACK_RUNTIME=production; production mode cannot bootstrap local accounts. Local migration runner remains local-only. New stage DB user has no UPDATE/DELETE permission on raw receipts.
- All 35 tested authenticated GET API paths returned 200. Replaying a real source payload twice through TCP produced 2 immutable receipts and 1 sensor telemetry record; fuel remained uncalibrated/NULL liters. This is a replay test, not physical device validation.
- Current actual ERI mask 00008104 is supported with exact bounded observed opaque CAN tokens and documented BLE/RAT sections. Unknown CAN contents remain unsupported; no undocumented CAN meaning was invented.

**Live cutover remains incomplete.** The user's requirement to verify every page/function is not yet met: notification push parity, unresolved historical assignments, unsupported hardware/external actions and full visual acceptance still require work. Do not use staging success as a claim of completed production replacement. Reports: `docs/rebuild/server-validation/` (counts/status only, no credentials/payload identifiers).
