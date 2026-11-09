package alertengine

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

type Queue struct {
	DB          *sql.DB
	LockTimeout time.Duration
	Now         func() time.Time
}

func (q Queue) now() time.Time {
	if q.Now != nil {
		return q.Now().UTC()
	}
	return time.Now().UTC()
}
func (q Queue) timeout() time.Duration {
	if q.LockTimeout <= 0 {
		return 2 * time.Minute
	}
	return q.LockTimeout
}

// NextWake returns the next business deadline for a retry or abandoned lease.
// A nil deadline means the worker can sleep until an external commit event.
func (q Queue) NextWake(ctx context.Context) (*time.Time, error) {
	if q.DB == nil {
		return nil, fmt.Errorf("alert queue database is required")
	}
	var retry, locked sql.NullTime
	err := q.DB.QueryRowContext(ctx, `SELECT MIN(CASE WHEN state IN ('PENDING','RETRY') AND attempts<max_attempts THEN available_at END),MIN(CASE WHEN state='PROCESSING' AND attempts<max_attempts THEN locked_at END) FROM alert_evaluation_jobs WHERE (state IN ('PENDING','RETRY') AND attempts<max_attempts) OR (state='PROCESSING' AND attempts<max_attempts)`).Scan(&retry, &locked)
	if err != nil {
		return nil, err
	}
	var deadline *time.Time
	if retry.Valid {
		value := retry.Time.UTC()
		deadline = &value
	}
	if locked.Valid {
		value := locked.Time.UTC().Add(q.timeout())
		if deadline == nil || value.Before(*deadline) {
			deadline = &value
		}
	}
	return deadline, nil
}

func (q Queue) Claim(ctx context.Context, workerID string) (*Job, error) {
	if q.DB == nil {
		return nil, fmt.Errorf("alert queue database is required")
	}
	if workerID == "" || len(workerID) > 80 {
		return nil, fmt.Errorf("valid worker ID is required")
	}
	now := q.now()
	expired := now.Add(-q.timeout())
	tx, err := q.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Exhausted abandoned jobs become traceable failures before another claim.
	rows, err := tx.QueryContext(ctx, `SELECT id,locked_by FROM alert_evaluation_jobs WHERE state='PROCESSING' AND locked_at<? AND attempts>=max_attempts ORDER BY id LIMIT 100 FOR UPDATE SKIP LOCKED`, expired)
	if err != nil {
		return nil, err
	}
	type exhausted struct {
		id     int64
		worker sql.NullString
	}
	items := []exhausted{}
	for rows.Next() {
		var item exhausted
		if err = rows.Scan(&item.id, &item.worker); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if _, err = tx.ExecContext(ctx, `UPDATE alert_evaluation_jobs SET state='FAILED',processed_at=?,locked_at=NULL,locked_by=NULL,last_error_code='LOCK_EXPIRED_MAX_ATTEMPTS',last_error_redacted='worker lease expired after maximum attempts' WHERE id=?`, now, item.id); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO alert_job_history(job_id,previous_state,new_state,worker_id,reason_code,detail_redacted,occurred_at) VALUES(?,'PROCESSING','FAILED',?,'LOCK_EXPIRED_MAX_ATTEMPTS','worker lease expired after maximum attempts',?)`, item.id, item.worker, now); err != nil {
			return nil, err
		}
	}
	var job Job
	err = tx.QueryRowContext(ctx, `SELECT id,event_id,vehicle_id,job_type,state,attempts,max_attempts,available_at FROM alert_evaluation_jobs WHERE attempts<max_attempts AND state IN ('PENDING','RETRY') AND available_at<=? ORDER BY state,available_at,locked_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, now).Scan(&job.ID, &job.EventID, &job.VehicleID, &job.Type, &job.State, &job.Attempts, &job.MaxAttempts, &job.AvailableAt)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `SELECT id,event_id,vehicle_id,job_type,state,attempts,max_attempts,available_at FROM alert_evaluation_jobs WHERE attempts<max_attempts AND state='PROCESSING' AND locked_at<? ORDER BY state,available_at,locked_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, expired).Scan(&job.ID, &job.EventID, &job.VehicleID, &job.Type, &job.State, &job.Attempts, &job.MaxAttempts, &job.AvailableAt)
	}
	if err == sql.ErrNoRows {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	previous := job.State
	job.State = JobProcessing
	job.WorkerID = workerID
	job.Attempts++
	if _, err = tx.ExecContext(ctx, `UPDATE alert_evaluation_jobs SET state='PROCESSING',attempts=?,locked_at=?,locked_by=?,last_error_code=NULL,last_error_redacted=NULL WHERE id=?`, job.Attempts, now, workerID, job.ID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO alert_job_history(job_id,previous_state,new_state,worker_id,reason_code,occurred_at) VALUES(?,?,?,?,?,?)`, job.ID, previous, JobProcessing, workerID, leaseReason(previous), now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &job, nil
}

func leaseReason(previous JobState) string {
	if previous == JobProcessing {
		return "EXPIRED_LOCK_RECOVERED"
	}
	return "CLAIMED"
}

func (q Queue) Complete(ctx context.Context, job Job, reason string) error {
	return q.finish(ctx, job, JobSuccess, reason, "", time.Time{})
}

func (q Queue) Fail(ctx context.Context, job Job, code, detail string, retryable bool) error {
	state := JobFailed
	available := time.Time{}
	if retryable && job.Attempts < job.MaxAttempts {
		state = JobRetry
		seconds := math.Pow(2, float64(job.Attempts-1))
		if seconds > 300 {
			seconds = 300
		}
		available = q.now().Add(time.Duration(seconds) * time.Second)
	}
	return q.finish(ctx, job, state, code, redact(detail), available)
}

func (q Queue) finish(ctx context.Context, job Job, state JobState, code, detail string, available time.Time) error {
	if q.DB == nil {
		return fmt.Errorf("alert queue database is required")
	}
	now := q.now()
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current JobState
	var lockedBy sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT state,locked_by FROM alert_evaluation_jobs WHERE id=? FOR UPDATE", job.ID).Scan(&current, &lockedBy); err != nil {
		return err
	}
	if current != JobProcessing || !lockedBy.Valid || lockedBy.String != job.WorkerID {
		return fmt.Errorf("job lease lost")
	}
	processed := any(nil)
	if state == JobSuccess || state == JobFailed {
		processed = now
	}
	next := any(nil)
	if state == JobRetry {
		next = available
	}
	result, err := tx.ExecContext(ctx, `UPDATE alert_evaluation_jobs SET state=?,available_at=COALESCE(?,available_at),locked_at=NULL,locked_by=NULL,processed_at=?,last_error_code=?,last_error_redacted=? WHERE id=? AND state='PROCESSING' AND locked_by=?`, state, next, processed, nullableString(code), nullableString(detail), job.ID, job.WorkerID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("job lease lost")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO alert_job_history(job_id,previous_state,new_state,worker_id,reason_code,detail_redacted,occurred_at) VALUES(?,'PROCESSING',?,?,?,?,?)`, job.ID, state, job.WorkerID, nullableString(code), nullableString(detail), now); err != nil {
		return err
	}
	return tx.Commit()
}

func redact(value string) string {
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
