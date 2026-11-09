# PasTrack rebuild — implementation progress

Updated 3 November 2026 (Asia/Jakarta). **Full rebuild is not complete.** New implementation is isolated from the existing application and production. Local execution authorized by latest master prompt; deployment remains prohibited until explicitly requested.

| Phase | Status | Implementasi | Test | Catatan |
|---|---|---|---|---|
| Audit | Partial verified | Existing Go/Beego/vanilla/Leaflet/SSE, migration/query inventory, 21 exported design screens, official protocol inspected | Source and asset inventory | No Git metadata; current legacy DDL/live snapshot not available through an authorized working local connection |
| Database | Schema implemented locally; historical migration pending | 65 domain tables, 109 FKs, checks/indexes, checksum-based explicit migration runner, separate dev/test schemas | MySQL 8.0.43: migration apply/reapply, FK presence, SQL integration tests | No old records changed; no automatic startup migration; historical backfill/rollback rehearsal not yet implemented |
| Backend | Partial implemented | Separate local v2 entrypoint, bcrypt sessions, roles, JSON APIs, bounded cursor lists, master writes, assignments, calibration approval, CSV report jobs, SSE | Authenticated HTTP tests against MySQL; bounded input and permissions tests | Old application remains intact. New v2 does not yet provide parity for every legacy capability |
| GPS | Partial implemented | STT UTC/mask/satellites, 0306 GNSS validity+HDOP, supported ERI GPS groups, immutable raw, byte-safe spool, retry identity, duplicate normalized facts, device/quality/network records | Observed STT fixtures; constructed ERI fixtures; raw/position SQL tests; framing tests | Other ERI section masks, full ACK correlation, heartbeat and alarm decoders remain unsupported |
| BLE/Fuel | Partial implemented | Five-byte CFS300B words, supported ERI mask walking, optional identity/status, telemetry, binding lookup, uncalibrated NULL liters, approved tank curve, last-valid reference, rule-gated sustained fuel events | Observed five-byte sample; document-derived full frame; calibration/drop/disconnect tests | Units not assumed; current physical tank curve and full actual frame variants not validated |
| Frontend | Partial implemented | New shell inspired by local exports, inventory/live/history/fuel/diagnostic views, vehicle subroutes, Leaflet map, valid-liter chart, forms, filters, cursor, CSV export, raw redaction | Two v2 DOM tests; JS syntax; login visually inspected; 26 existing UI tests pass after fixes | Shared data views are not completed reproductions of all 21 Stitch compositions. Animated v2 playback, polished detail layouts and remaining settings/report scheduling UI still pending |
| Integration | Partial verified | MySQL→API/auth/report download tested; receiver raw→parse→SQL tested; Unix socket→SSE wired | Local dev/test DBs separated; /healthz web+receiver HTTP 200 | No physical device E2E claim; complete TCP→authenticated browser proof still pending |
| Testing | Passing implemented coverage | Go unit/integration/race, vet, existing web tests, DOM tests, syntax/build | See TESTING_REPORT.md | Passing tests do not imply every required feature is implemented |
| Documentation | Updated incrementally | Architecture/ERD/schema/mapping retained; current changes/limits/local startup/API documented | Migration generated from schema.json; links reviewed earlier | Final production documentation pending actual backfill, complete UI and physical validation |

## Next executable work

1. Complete v2-specific screen compositions and playback interactions from all local exports; do not count shared table views as completed Stitch screens.
2. Implement incremental, audited legacy backfill and reverse-delta tooling after a local restored snapshot is available; never infer historic vehicle assignments.
3. Expand actual protocol fixture inventory, supported masks and command correlation only with documented evidence.
4. Finish event/notification delivery, report schedules, geofence editing/assignment UX and complete RBAC policy parity.
5. Run full TCP→SQL→API→browser test, local migration restore/rollback rehearsal and visual desktop/mobile checks.

## Stitch correction update

The generic served frontend has been replaced by actual HTML imported from all 21 local design exports, including the original 17-entry menu. API bindings, selected-vehicle tabs, playback, geometry editing and report exports were expanded. See STITCH_IMPLEMENTATION.md. This supersedes the earlier statement that v2 still uses generic table compositions. Exact visual acceptance for every screen and complete hardware/feature parity remain unfinished; do not mark full rebuild complete.

## Live user acceptance deployment

User explicitly requested deployment for function review. Web and raw receiver are now active at the existing managed production paths, using `pastrack_v2_20261103_r2`. Snapshot/raw/position parity and public authenticated smoke checks passed. The old database/config/binaries remain preserved. This supersedes the earlier “no live cutover” status; full rebuild completion is still not claimed. See LIVE_DEPLOYMENT_20261103.md.
