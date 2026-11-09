# PasTrack Alert Engine Phase 4 Implementation

Date: 2026-11-05  
Scope: vehicle-centric alert processing infrastructure only

## 1. Implemented Architecture

Phase 4 adds a separate `internal/alertengine` service layer. It consumes the normalized Phase 3 GPS/BLE contract, persists an immutable normalized input, creates a durable MySQL evaluation job in the same transaction, and lets independent workers evaluate registered rule modules.

No rule implementation is registered by this phase. No receiver, parser, BLE ingestion, HTTP API, frontend, authentication, or notification path was changed.

```text
Queclink / BLE persisted telemetry
              |
              v
Phase 3 normalized telemetry + historical vehicle resolver
              |
              v
       Alert Intake validation
       | invalid          | valid (one transaction)
       v                  +--------------------------+
 quarantine        alert_input_events               |
                    + alert_evaluation_jobs <--------+
                              |
                    SELECT ... FOR UPDATE SKIP LOCKED
                              |
                              v
                     concurrent worker(s)
                       | no evaluator
                       +----------> SUCCESS / NO_EVALUATOR
                       |
                       v
                    rule registry
                       |
                       v
        per rule + vehicle + scope transactional lock
                       |
             +---------+----------+
             v                    v
      alert_events episode   alert_rule_states
             |
       +-----+-------------------+
       v                         v
 alert_event_sources       alert_audit_trail
```

## 2. Event Intake Implementation

- `EventFromGPS` and `EventFromFuel` adapt the actual Phase 3 normalized structures.
- `vehicle_id > 0`, an existing vehicle row, and `resolution_status=RESOLVED` are mandatory.
- UNASSIGNED, AMBIGUOUS, and INVALID inputs are quarantined rather than converted into vehicle alerts.
- ORIGINAL_DEVICE_TIME and CORRELATED_TIME require `event_time`.
- RECEIVED_TIME_ONLY and UNKNOWN require `event_time=NULL`; `received_at` is never written into the input event-time column.
- The original device/sensor identity and telemetry row references are retained.
- A SHA-256 event key unique constraint makes event/job creation idempotent.

## 3. MySQL Evaluation Job Queue

The job states are PENDING, PROCESSING, SUCCESS, RETRY, and FAILED. Claiming uses a short READ COMMITTED transaction with `FOR UPDATE SKIP LOCKED`, verified on isolated MySQL 8.0.43 and checked against the live MariaDB 10.11 capability without applying production changes.

Lease ownership is checked again when completing or failing a job. Expired leases can be recovered, exhausted expired jobs become traceable failures, and retry delay uses bounded exponential backoff (maximum 300 seconds). Every claim/retry/failure/success transition is retained in `alert_job_history`.

## 4. Worker Architecture and Engine Core

`Worker.RunOnce` claims one job, reloads and revalidates the immutable input, finds matching `RuleEvaluator` modules, processes them, then completes the job. Runtime wiring is deliberately absent in Phase 4, so TCP ingestion cannot block on rule evaluation.

When there is no registered evaluator, the worker marks the job SUCCESS with reason `NO_EVALUATOR`. It does not insert an alert event. This is the normal Phase 4 behavior, not an error or a synthetic test alert.

The generic evaluator exposes rule identity/version, event acceptance, validation, evaluation, action, scope, and durable evaluator state. No FUEL_LOW, FUEL_DROP, OVER_SPEED, CRASH, or ignition logic exists.

## 5. Alert Episode Management

`alert_rule_states` serializes evaluation for `(rule_id, vehicle_id, scope_key)`. Trigger observations update an existing episode; after resolution a later trigger creates a new episode. The episode key includes rule, vehicle, scope, and first input event and is protected by a unique index.

Events with an original/correlated event time older than the durable watermark do not mutate a newer episode. Inputs without trustworthy event time retain NULL event time and do not advance that event-time watermark.

The alert row freezes `vehicle_id`; later device assignment changes do not rewrite historical alerts.

## 6. Deduplication and Concurrency

- Input: unique binary SHA-256 event key.
- Job: unique `(event_id, job_type)`.
- Rule state: unique `(rule_id, vehicle_id, scope_key)` plus row lock.
- Episode: unique episode key.
- Provenance: unique `(alert_id, input_event_id)`.
- Job finish: worker lease owner check.
- Lifecycle: alert row lock and transition validation.

Retry after persistence is safe because state/provenance uniqueness prevents a duplicate episode or duplicate source attachment.

## 7. Lifecycle and Audit Trail

Allowed transitions are:

- active -> acknowledged, resolved, false_positive
- acknowledged -> resolved, false_positive

Terminal transitions clear the matching current episode pointer. Every transition records previous/new state, actor type, optional real user ID, timestamp, and reason. System actions always use `actor_type=SYSTEM` with NULL actor ID; no user is fabricated.

Database state strings remain lowercase to preserve the existing dashboard/API convention.

## 8. Provenance

`alert_event_sources` is a many-to-one history from GPS/BLE normalized inputs to an alert episode. It stores source role (TRIGGER, OBSERVATION, RECOVERY), device/sensor IDs, source type, event time, received time, and timestamp quality. Foreign keys do not depend on active assignment/binding status, so inactive or soft-deleted device history remains traceable.

## 9. Database Changes

Migration `010_alert_engine_foundation.sql` additively extends existing `alert_events`; it does not create a second alert master table. It adds normalized input/quarantine, queue/history, rule state, provenance, and audit tables plus foreign keys and indexes. A development-only rollback is available under `database/rollbacks` and is intentionally outside the automatic forward migration embed.

The migration was applied only to `127.0.0.1:33307/pastrack_test_local`. It was not run against production.

## 10. Files Changed

- `database/migrations/010_alert_engine_foundation.sql`: additive Phase 4 schema.
- `database/rollbacks/010_alert_engine_foundation.sql`: explicit development rollback.
- `internal/alertengine/types.go`: normalized event, queue, evaluator, lifecycle types.
- `internal/alertengine/intake.go`: validation, quarantine, atomic event/job enqueue.
- `internal/alertengine/queue.go`: durable concurrent claim, lease recovery, retry, history.
- `internal/alertengine/engine.go`: worker, evaluator registry use, episode persistence, provenance, lifecycle, audit.
- `internal/alertengine/helpers.go`: internal key/request helpers.
- `internal/alertengine/intake_test.go`: identity and timestamp unit coverage.
- `internal/alertengine/integration_test.go`: isolated MySQL queue/worker/episode/lifecycle/concurrency coverage.

## 11. Test and Regression Results

Executed with `PASTRACK_TEST_DSN=root@tcp(127.0.0.1:33307)/pastrack_test_local?parseTime=true`:

- `go test ./...`: PASS.
- `go test -race ./...`: PASS.
- `go vet ./...`: PASS.
- Phase 4 integration coverage includes duplicate enqueue, missing vehicle quarantine, no-evaluator success, retry/permanent failure, expired lease recovery, parallel distinct job claims, concurrent same-episode processing, multi-input provenance, new episode after resolution, valid/invalid lifecycle transitions, and audit preservation.

No race was reported. Existing platform, protocol, telemetry, fuel, trip, remote-output, and root regression packages remained green.

## 12. Performance Findings

No production benchmark was run. The implementation minimizes ingestion work to validation plus two indexed inserts in one transaction. Worker evaluation is asynchronous, claims one indexed job with a short lock, and serializes only the same rule/vehicle/scope. Parallel test workers claimed distinct jobs without duplicate ownership.

Queue retention/archival policy is intentionally not automated yet; completed job history remains available for audit.

## 13. Known Limitations and Deferred Scope

- The worker is a library component and is not started by the production binaries in Phase 4.
- No business evaluator is implemented or registered.
- There is no dispatcher, notification delivery, endpoint, dashboard, or frontend change.
- CRASH remains NOT_READY and physical ignition remains NOT_INSTALLED/UNKNOWN where hardware data is unavailable.
- RECEIVED_TIME_ONLY and UNKNOWN inputs can be accepted by the infrastructure, but a future rule must explicitly decide whether that timestamp quality is eligible.
- Queue archival, operational metrics, and worker process supervision are deferred to a later authorized phase.
- No production migration, deployment, or data mutation was performed.

## 14. Existing Components Explicitly Left Untouched

- Queclink GV30CEU TCP receiver and protocol parser.
- BLE CFS300B ingestion and receiver-context validation.
- GPS/fuel persistence and calibration.
- Historical GPS assignment and BLE binding resolvers.
- Trip history, tracking, dashboard, existing alert API contract, authentication, and notification tables/workflows.
- Live server database and deployed binaries.

Phase 4 stops here. Phase 5 business rules require separate instruction and authorization.
