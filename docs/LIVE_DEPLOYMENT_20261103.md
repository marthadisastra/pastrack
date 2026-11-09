# PasTrack v2 live — 3 November 2026

Status: deployed for user acceptance. This is not a declaration that every rebuild feature or hardware capability is complete.

- Public application: https://toko.azlaundry.my.id/
- Live database: `pastrack_v2_20261103_r2` on server MariaDB port 3306.
- Web: aaPanel project `pastrack_web`, existing executable `/www/wwwroot/pastrack/pastrack-web`, existing loopback port 18089.
- Raw receiver: aaPanel project `pastrack`, existing executable `/opt/pastrack/pastrack`, existing public TCP port 5001; private health port 18087.
- Both managed projects now use the same new v2 database through `/etc/pastrack/v2-live.env`. Original environment files are preserved; the new file contains private credentials and must not be exposed.
- Existing bcrypt account hashes are preserved. User must sign in again because the v2 application uses its own session cookie. No password reset was performed.
- Inactive dashboard text/icon color fix is included in the served release.

## Migration and verification

An initial activation attempt stopped at count validation and automatically restored healthy legacy services. The source held multiple historical samples linked to the same raw receipt; assigning every imported sample index 0 collided with the new unique constraint. The importer now preserves an ordered legacy sample ordinal per raw receipt. The unsuccessful target remains isolated; no source data was dropped or truncated.

The second cutover froze both old writers, took a fresh consistent source snapshot, and imported to a separate empty target. Before activation, all 428 source position IDs, coordinates and timestamps and all 2971 raw receipts were checked for parity. Both services were then replaced atomically at their existing managed paths and restarted through aaPanel. Nginx upstream/address did not change.

Public authenticated smoke checks returned 200 for the shell, identity, vehicles, fleet overview/live, device health/raw, sensors and the corrected stylesheet. Private web and receiver health report v2/status ok, with empty receiver spool. Actual fresh physical tracker transmission after cutover has not been claimed; the previous staging TCP test used a replay of an actual payload.

## Backup / recovery

Private backup: `/root/pastrack-live-cutover-20261103-r2/backup/`, plus original executable and launch script copies in the parent directory. Existing source database `pastrack` remains intact and is not the current writer target. No DROP/TRUNCATE was executed.

Do not blindly run the initial cutover tool again: it rejects an existing target/success marker. Rollback after new data has been accepted must preserve v2-only receipts and master changes first. The activation tool requeues newly accepted raw receipts if an activation-time failure occurs; it is not a complete long-term reverse migration tool. Avoid unreviewed database overwrites.

## Known boundaries for this review release

- Push delivery is not ported; the source push credential remains preserved privately rather than misrepresented as encrypted in the new table.
- 99 historical positions lack sufficient assignment evidence; they remain in history with unresolved vehicle association. No old association was fabricated.
- Hardware/OTA/immobilizer and unconfigured third-party controls remain disabled with reasons.
- Fuel liters remain unavailable without an approved tank calibration. Physical ignition remains NOT_INSTALLED.
- All-page visual/function acceptance is still in progress; public GET smoke checks do not prove every action is complete.
