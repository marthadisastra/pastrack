# Rebuild issues and limitations

## Evidence/access blockers

- Current legacy DB DDL/data are not available in an authorized working local connection. Existing MySQL root without credentials was rejected; it was not modified. A separate MySQL instance at port 33307 was initialized instead. Do not claim legacy migration/count parity or restore rehearsal completed.
- Full current actual-device ERI/ACK/heartbeat/alarm frame inventory is unavailable in this audit. Existing STT fixtures and CFS raw words are available; full ERI integration fixtures are constructed from the official manual and labeled as such.
- Physical GV30CEU/CFS300B pipeline and tank volume calibration were not exercised. Water immersion evidence cannot approve a fuel tank curve or scale raw volume to liters.

## Remaining implementation work (not external blockers)

- Exact compositions/interactions for all 21 Stitch exports; current v2 shared views cover real data access but do not meet full visual fidelity/feature parity. V2 playback currently draws bounded route points, not a complete animation control surface.
- Historical migration adapter, source checksums/count reconciliation and reverse-delta rollback tools remain to be built and rehearsed against a restored local snapshot.
- Notifications/push delivery, report schedules, command delivery/correlation, organizations (if truly required) and complete permission policy are not fully ported to v2. Old implementations remain untouched.
- Supported ERI path currently requires ERI mask 0x100; CAN/RAT sections and extended BLE masks are preserved as unsupported. STT optional mask support is bounded to verified flags; older 0100 fix semantics remain unknown rather than borrowed from v0306.
- Trip engine persists on explicit bounded recalculate request; incremental ongoing-trip worker/segments/summaries still pending. Fuel rule processing and geometry are implemented; broader fleet-scale integration/load coverage remains pending.
- A migration process crash exactly between MySQL DDL success and step-ledger commit can require manual reconciliation. Runner fails closed; do not edit an applied migration or automatically drop/recreate schema. Back up and compare information_schema before recording an already-applied step.
- Blocked overlapping assignments preserve raw and a .blocked spool copy. Repair assignment history and explicitly requeue that file; never invent a historical assignment or silently discard it.
- SQL engine constraints were validated on local MySQL 8.0.43 only, not the unknown production MySQL/MariaDB version.
- Local dev database intentionally has no fleet/telemetry seed. Empty map/fuel charts are expected until actual validated records are imported/received. QA records exist only in isolated test schemas.

- Fuel event windows currently use server received_at to describe sustained observed changes; absent sensor timestamps remain NULL in readings and must not be labeled physical event time. Event-time late-arrival reconciliation and dedicated source-time metadata remain pending.
