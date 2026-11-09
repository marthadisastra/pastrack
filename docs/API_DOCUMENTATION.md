# V2 API implemented locally

Base `/api/v1`; authenticated session cookie required except POST /login. Existing web/ API remains separate. These routes query real target SQL tables; empty database yields empty data, not mock telemetry.

## Collection/detail queries

| Collection | Read | Write |
|---|---|---|
| /vehicles | GET list and GET /{id} | POST, PUT /{id} |
| /devices | GET list and GET /{id} | POST, PUT /{id} |
| /device-models | GET list and GET /{id} | POST, PUT /{id} |
| /sensors | GET list and GET /{id} | POST, PUT /{id} |
| /fuel/sensors | GET list and GET /{id} | POST, PUT /{id} |
| /fuel/readings | GET list and GET /{id} | — |
| /fuel/events | GET list and GET /{id} | — |
| /fuel/analytics | GET list and GET /{id} | — |
| /gps/positions | GET list and GET /{id} | — |
| /gps/quality | GET list and GET /{id} | — |
| /trips | GET list and GET /{id} | — |
| /devices/health | GET list and GET /{id} | — |
| /devices/connectivity | GET list and GET /{id} | — |
| /devices/data-activity | GET list and GET /{id} | — |
| /devices/raw-messages | GET list and GET /{id} | — |
| /devices/parse-errors | GET list and GET /{id} | — |
| /events | GET list and GET /{id} | — |
| /geofences | GET list and GET /{id} | POST, PUT /{id} |
| /geofence-events | GET list and GET /{id} | — |
| /alerts/rules | GET list and GET /{id} | POST, PUT /{id} |
| /alerts/events | GET list and GET /{id} | — |
| /sensor-telemetry | GET list and GET /{id} | — |
| /calibrations | GET list and GET /{id} | POST, PUT /{id} |
| /tank-profiles | GET list and GET /{id} | POST, PUT /{id} |
| /users | GET list and GET /{id} | POST, PUT /{id} (admin; bcrypt, roles, auth_version) |
| /notifications | GET list and GET /{id} | — |
| /audit | GET list and GET /{id} | — |
| /reports | GET list and GET /{id} | POST (asynchronous bounded CSV job) |
| /event-types | GET list and GET /{id} | POST, PUT /{id} |
| /event-severities | GET list and GET /{id} | POST, PUT /{id} |
| /geofence-assignments | GET list and GET /{id} | POST, PUT /{id} |

## Specialized routes

- GET /me; POST /logout; POST /login (username/password JSON; rate limited).
- GET /fleet/overview, /fleet/statistics, /fleet/live, /fleet/activity; /fuel/overview.
- GET /gps/history aliases positions; /devices/gps-quality aliases quality.
- GET /vehicles/{id}/telemetry, /positions, /trips, /events, /fuel, /devices, /calibrations.
- GET/POST /vehicles/{id}/assignments; GET/POST /sensors/{id}/assignments. POST requires device_id, starts_at RFC3339, optional ends_at; sensor binding also vehicle_id. Overlaps return 409.
- GET/POST /calibrations/{id}/points: input_value, output_value (liters), evidence. POST /calibrations/{id}/approve admin only, requires valid approved tank-volume curve; approved curves are immutable. Water/unsupported quantity cannot be approved.
- GET /trips/summary; POST /trips/recalculate: vehicle_id/from/to, optional explicit speed/duration/distance/gap thresholds. Maximum 31 days/10000 validated points, versioned idempotent trip persistence. Open unfinished trip is not emitted as completed.
- POST /alerts/events/{id}/acknowledge; roles admin/operator. Rule configuration for fuel events is explicit JSON, never automatically seeded.
- POST /reports with report_type positions/trips/fuel/events, from/to RFC3339, optional vehicle_id. Maximum 31 days/50000 exported rows; two concurrent jobs; GET /reports/{id}/download requires owner/admin and completed unexpired artifact.
- GET /realtime uses SSE ready/change + keepalive, reconnect; Unix datagram ingestion invalidations reused. No application-wide periodic data polling.

## Query/response contract

List query: limit 1..200 (default 50), cursor (descending ID keyset), search only on declared searchable resources, vehicle_id where resource supports it, from/to RFC3339 where timestamp is declared. Sorting currently fixed to descending ID; arbitrary sorting is not implemented. Live fleet endpoint is bounded at 1000 vehicles and has no complete paging yet.

List response `{data:[],page:{has_more,next_cursor},meta:{as_of,timezone}}`; detail `{data:{...}}`; error `{error:{code,message}}`. Mutation JSON must match allowlisted fields. SQL values use placeholders. Invalid query=400, no session=401, forbidden=403, missing=404, conflict=409, invalid mutation=422, SQL unavailable=503.

Device identifiers are masked on list/detail. Raw inspector returns payload_redacted only to admin. Never use the inspector as a public data export. API does not return passwords, session tokens, report storage paths or secrets.

## Not yet provided

Full CRUD/soft-deactivation parity, report schedules, notification delivery/push ports, per-permission RBAC matrix, arbitrary list sorting, complete frontend-specific aggregate contracts and automatic incremental trip worker remain unfinished. No placeholder API is advertised as a working implementation.
