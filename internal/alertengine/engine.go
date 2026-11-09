package alertengine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type EpisodeManager struct {
	DB  *sql.DB
	Now func() time.Time
}

func (m EpisodeManager) now() time.Time {
	if m.Now != nil {
		return m.Now().UTC()
	}
	return time.Now().UTC()
}

// Process serializes a rule/vehicle/scope, evaluates with its durable state,
// and applies the result in the same transaction.
func (m EpisodeManager) Process(ctx context.Context, evaluator RuleEvaluator, event Event) error {
	if m.DB == nil || evaluator == nil {
		return fmt.Errorf("episode manager dependencies are required")
	}
	if err := ValidateEvent(event); err != nil {
		return err
	}
	if err := evaluator.Validate(event); err != nil {
		return ValidationError{"RULE_INPUT_INVALID"}
	}

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// The empty scope is a valid global-per-vehicle scope. Evaluators may return
	// a narrower scope; the initial state is therefore read after evaluation.
	evaluation, err := evaluator.Evaluate(event, nil)
	if err != nil {
		return fmt.Errorf("rule evaluation: %w", err)
	}
	if evaluation.ScopeKey == "" {
		evaluation.ScopeKey = "vehicle"
	}
	lockedScope := evaluation.ScopeKey
	if len(evaluation.ScopeKey) > 120 {
		return ValidationError{"RULE_SCOPE_INVALID"}
	}
	if evaluation.Action == NoAction {
		return tx.Commit()
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO alert_rule_states(rule_id,vehicle_id,scope_key) VALUES(?,?,?) ON DUPLICATE KEY UPDATE id=id`, evaluator.RuleID(), event.VehicleID, evaluation.ScopeKey)
	if err != nil {
		return err
	}
	var stateID int64
	var alertID, lastEvent sql.NullInt64
	var stateBytes []byte
	var watermark sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id,current_alert_id,evaluator_state,event_watermark,last_event_id FROM alert_rule_states WHERE rule_id=? AND vehicle_id=? AND scope_key=? FOR UPDATE`, evaluator.RuleID(), event.VehicleID, evaluation.ScopeKey).Scan(&stateID, &alertID, &stateBytes, &watermark, &lastEvent)
	if err != nil {
		return err
	}

	// Re-evaluate with the durable rule state once the scope lock is held.
	evaluation, err = evaluator.Evaluate(event, json.RawMessage(stateBytes))
	if err != nil {
		return fmt.Errorf("rule evaluation: %w", err)
	}
	if evaluation.ScopeKey == "" {
		evaluation.ScopeKey = "vehicle"
	}
	if evaluation.ScopeKey != lockedScope {
		return ValidationError{"RULE_SCOPE_CHANGED"}
	}

	if event.EventTime != nil && watermark.Valid && event.EventTime.Before(watermark.Time) {
		return tx.Commit() // retain late input/job history without mutating a newer episode
	}

	now := m.now()
	switch evaluation.Action {
	case Trigger:
		if alertID.Valid {
			_, err = tx.ExecContext(ctx, `UPDATE alert_events SET last_detected_at=?,severity=?,title=?,description=?,latitude=?,longitude=?,row_version=row_version+1 WHERE id=?`, detectedAt(event, now), nullableString(evaluation.Severity), nullableString(evaluation.Title), nullableString(evaluation.Description), evaluation.Latitude, evaluation.Longitude, alertID.Int64)
			if err != nil {
				return err
			}
			if err = attachSource(ctx, tx, alertID.Int64, event, "OBSERVATION"); err != nil {
				return err
			}
			if err = attachRelatedSources(ctx, tx, alertID.Int64, event.VehicleID, event.ID, evaluation.RelatedEventIDs); err != nil {
				return err
			}
			if err = audit(ctx, tx, alertID.Int64, "ALERT_UPDATED", AlertActive, AlertActive, SystemActor, "RULE_TRIGGERED", "", &event.ID, now); err != nil {
				return err
			}
		} else {
			episodeKey := episodeKey(evaluator.RuleID(), event.VehicleID, evaluation.ScopeKey, event.ID)
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO alert_events(rule_id,rule_code,rule_version,severity,title,description,vehicle_id,state,episode_key,scope_key,raised_at,last_detected_at,primary_device_id,primary_sensor_id,latitude,longitude) VALUES(?,?,?,?,?,?,?,'active',?,?,?,?,?,?,?,?)`, evaluator.RuleID(), evaluator.RuleCode(), evaluator.RuleVersion(), nullableString(evaluation.Severity), nullableString(evaluation.Title), nullableString(evaluation.Description), event.VehicleID, episodeKey, evaluation.ScopeKey, detectedAt(event, now), detectedAt(event, now), event.DeviceID, event.SensorID, evaluation.Latitude, evaluation.Longitude)
			if insertErr != nil {
				return insertErr
			}
			alertID.Int64, err = result.LastInsertId()
			alertID.Valid = err == nil
			if err != nil {
				return err
			}
			if err = attachSource(ctx, tx, alertID.Int64, event, "TRIGGER"); err != nil {
				return err
			}
			if err = attachRelatedSources(ctx, tx, alertID.Int64, event.VehicleID, event.ID, evaluation.RelatedEventIDs); err != nil {
				return err
			}
			if err = audit(ctx, tx, alertID.Int64, "ALERT_CREATED", "", AlertActive, SystemActor, "RULE_TRIGGERED", "", &event.ID, now); err != nil {
				return err
			}
		}
	case Observe:
		if alertID.Valid {
			if _, err = tx.ExecContext(ctx, `UPDATE alert_events SET last_detected_at=?,severity=COALESCE(?,severity),title=COALESCE(?,title),description=COALESCE(?,description),row_version=row_version+1 WHERE id=?`, detectedAt(event, now), nullableString(evaluation.Severity), nullableString(evaluation.Title), nullableString(evaluation.Description), alertID.Int64); err != nil {
				return err
			}
			if err = attachSource(ctx, tx, alertID.Int64, event, "OBSERVATION"); err != nil {
				return err
			}
		}
	case Recover:
		if alertID.Valid {
			if _, err = tx.ExecContext(ctx, `UPDATE alert_events SET state='resolved',resolved_at=?,last_detected_at=?,row_version=row_version+1 WHERE id=? AND state IN ('active','acknowledged')`, now, detectedAt(event, now), alertID.Int64); err != nil {
				return err
			}
			if err = attachSource(ctx, tx, alertID.Int64, event, "RECOVERY"); err != nil {
				return err
			}
			if err = audit(ctx, tx, alertID.Int64, "ALERT_RESOLVED", "", AlertResolved, SystemActor, "RULE_RECOVERED", "", &event.ID, now); err != nil {
				return err
			}
			alertID = sql.NullInt64{}
		}
	default:
		return ValidationError{"RULE_ACTION_INVALID"}
	}

	var nextWatermark any
	if event.EventTime != nil {
		nextWatermark = event.EventTime.UTC()
	} else if watermark.Valid {
		nextWatermark = watermark.Time
	}
	_, err = tx.ExecContext(ctx, `UPDATE alert_rule_states SET current_alert_id=?,evaluator_state=?,event_watermark=?,last_event_id=?,row_version=row_version+1 WHERE id=?`, nullableNullInt(alertID), nullableJSON(evaluation.State), nextWatermark, event.ID, stateID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func detectedAt(event Event, fallback time.Time) time.Time {
	if event.EventTime != nil {
		return event.EventTime.UTC()
	}
	return fallback // lifecycle time only; event_time remains NULL in the input/provenance
}

func episodeKey(ruleID, vehicleID int64, scope string, eventID int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%d", ruleID, vehicleID, scope, eventID)))
	return hex.EncodeToString(sum[:])
}

func nullableNullInt(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}
func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return []byte(value)
}

func attachSource(ctx context.Context, tx *sql.Tx, alertID int64, event Event, role string) error {
	result, err := tx.ExecContext(ctx, `INSERT IGNORE INTO alert_event_sources(alert_id,input_event_id,source_role,source_type,device_id,sensor_id,event_time,received_at,timestamp_quality) VALUES(?,?,?,?,?,?,?,?,?)`, alertID, event.ID, role, event.SourceType, event.DeviceID, event.SensorID, event.EventTime, event.ReceivedAt, event.TimestampQuality)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO alert_audit_trail(alert_id,action,actor_type,reason_code,input_event_id,request_id,occurred_at) VALUES(?,'SOURCE_ATTACHED','SYSTEM','TELEMETRY_SOURCE',?,?,UTC_TIMESTAMP(3))`, alertID, event.ID, requestID())
	return err
}

func attachRelatedSources(ctx context.Context, tx *sql.Tx, alertID, vehicleID, currentEventID int64, eventIDs []int64) error {
	seen := map[int64]bool{currentEventID: true}
	for _, eventID := range eventIDs {
		if eventID <= 0 || seen[eventID] {
			continue
		}
		seen[eventID] = true
		related, err := loadEvent(ctx, tx, eventID)
		if err != nil {
			return err
		}
		if related.VehicleID != vehicleID {
			return ValidationError{"RELATED_EVENT_VEHICLE_MISMATCH"}
		}
		if err = attachSource(ctx, tx, alertID, related, "OBSERVATION"); err != nil {
			return err
		}
	}
	return nil
}

func audit(ctx context.Context, tx *sql.Tx, alertID int64, action string, previous, next AlertState, actor Actor, code, reason string, inputID *int64, at time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO alert_audit_trail(alert_id,action,previous_state,new_state,actor_id,actor_type,reason_code,reason_text,input_event_id,request_id,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, alertID, action, nullableString(string(previous)), nullableString(string(next)), actor.UserID, actor.Type, nullableString(code), nullableString(reason), inputID, requestID(), at)
	return err
}

func validateActor(actor Actor) error {
	if actor.Type == "SYSTEM" && actor.UserID == nil {
		return nil
	}
	if actor.Type == "USER" && actor.UserID != nil && *actor.UserID > 0 {
		return nil
	}
	return fmt.Errorf("invalid alert actor")
}

type Worker struct {
	Queue    Queue
	DB       *sql.DB
	Registry Registry
	Episodes EpisodeManager
	ID       string
}

func (w Worker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.Queue.Claim(ctx, w.ID)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	event, err := loadEvent(ctx, w.DB, job.EventID)
	if err != nil {
		return true, w.fail(ctx, *job, "EVENT_LOAD_FAILED", err, retryableDBError(err))
	}
	if err = ValidateEvent(event); err != nil {
		return true, w.fail(ctx, *job, "EVENT_VALIDATION_FAILED", err, false)
	}
	evaluators := w.Registry.Matching(event)
	if len(evaluators) == 0 {
		return true, w.Queue.Complete(ctx, *job, "NO_EVALUATOR")
	}
	for _, evaluator := range evaluators {
		if err = w.Episodes.Process(ctx, evaluator, event); err != nil {
			return true, w.fail(ctx, *job, "EVALUATION_FAILED", err, !isPermanent(err))
		}
	}
	return true, w.Queue.Complete(ctx, *job, "EVALUATED")
}

func (w Worker) fail(ctx context.Context, job Job, code string, cause error, retryable bool) error {
	// Persist stable classifications only. Arbitrary evaluator/database errors may
	// contain credentials, identifiers, or payload fragments.
	if err := w.Queue.Fail(ctx, job, code, code, retryable); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func isPermanent(err error) bool { var validation ValidationError; return errors.As(err, &validation) }
func retryableDBError(err error) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && (myErr.Number == 1205 || myErr.Number == 1213)
}

type Lifecycle struct {
	DB  *sql.DB
	Now func() time.Time
}

func (l Lifecycle) Transition(ctx context.Context, alertID int64, next AlertState, actor Actor, reason string) error {
	if l.DB == nil {
		return fmt.Errorf("lifecycle database is required")
	}
	if err := validateActor(actor); err != nil {
		return err
	}
	tx, err := l.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current AlertState
	if err = tx.QueryRowContext(ctx, `SELECT state FROM alert_events WHERE id=? FOR UPDATE`, alertID).Scan(&current); err != nil {
		return err
	}
	if !allowedTransition(current, next) {
		return fmt.Errorf("invalid alert transition %s to %s", current, next)
	}
	now := time.Now().UTC()
	if l.Now != nil {
		now = l.Now().UTC()
	}
	query := `UPDATE alert_events SET state=?,row_version=row_version+1`
	args := []any{next}
	switch next {
	case AlertAcknowledged:
		query += `,acknowledged_by=?,acknowledged_at=?`
		args = append(args, actor.UserID, now)
	case AlertResolved:
		query += `,resolved_at=?`
		args = append(args, now)
	case AlertFalsePositive:
		query += `,false_positive_by=?,false_positive_at=?,false_positive_reason=?`
		args = append(args, actor.UserID, now, nullableString(reason))
	}
	query += ` WHERE id=?`
	args = append(args, alertID)
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	if next == AlertResolved || next == AlertFalsePositive {
		if _, err = tx.ExecContext(ctx, `UPDATE alert_rule_states SET current_alert_id=NULL,row_version=row_version+1 WHERE current_alert_id=?`, alertID); err != nil {
			return err
		}
	}
	action := "ALERT_" + strings.ToUpper(string(next))
	if err = audit(ctx, tx, alertID, action, current, next, actor, "LIFECYCLE_TRANSITION", reason, nil, now); err != nil {
		return err
	}
	return tx.Commit()
}

func allowedTransition(current, next AlertState) bool {
	return (current == AlertActive && (next == AlertAcknowledged || next == AlertResolved || next == AlertFalsePositive)) ||
		(current == AlertAcknowledged && (next == AlertResolved || next == AlertFalsePositive))
}
