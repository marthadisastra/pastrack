package alertengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"pastrack/internal/telemetry"
)

type testEvaluator struct {
	id     int64
	action Action
	scope  string
}

func (e testEvaluator) RuleID() int64        { return e.id }
func (e testEvaluator) RuleCode() string     { return "TEST_INFRA_ONLY" }
func (e testEvaluator) RuleVersion() int64   { return 1 }
func (e testEvaluator) Accepts(Event) bool   { return true }
func (e testEvaluator) Validate(Event) error { return nil }
func (e testEvaluator) Evaluate(Event, json.RawMessage) (Evaluation, error) {
	return Evaluation{Action: e.action, ScopeKey: e.scope, Severity: "test", Title: "Infrastructure test", State: json.RawMessage(`{"ok":true}`)}, nil
}

type fixture struct {
	db                  *sql.DB
	vehicle, rule, user int64
	prefix              string
}

func setupFixture(t *testing.T) fixture {
	t.Helper()
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("PASTRACK_TEST_DSN not set")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_") || !strings.Contains(dsn, "_test_local") {
		t.Fatal("integration test requires an isolated loopback pastrack_*_test_local schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	prefix := fmt.Sprintf("a4_%d", time.Now().UnixNano())
	r, err := db.Exec(`INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,?,1)`, prefix, "Alert test")
	if err != nil {
		t.Fatal(err)
	}
	vehicle, _ := r.LastInsertId()
	_, err = db.Exec("INSERT INTO event_severities(code,`rank`) VALUES(?,1)", prefix)
	if err != nil {
		t.Fatal(err)
	}
	var severity int64
	if err = db.QueryRow(`SELECT id FROM event_severities WHERE code=?`, prefix).Scan(&severity); err != nil {
		t.Fatal(err)
	}
	r, err = db.Exec(`INSERT INTO alert_rules(name,severity_id,configuration,version,enabled) VALUES(?,?,JSON_OBJECT(),1,1)`, prefix, severity)
	if err != nil {
		t.Fatal(err)
	}
	rule, _ := r.LastInsertId()
	r, err = db.Exec(`INSERT INTO users(username,password_hash,password_algo,auth_version,is_active,full_name,row_version) VALUES(?,'x','test',1,1,'Alert Test',1)`, prefix)
	if err != nil {
		t.Fatal(err)
	}
	user, _ := r.LastInsertId()
	f := fixture{db: db, vehicle: vehicle, rule: rule, user: user, prefix: prefix}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM alert_audit_trail WHERE alert_id IN (SELECT id FROM alert_events WHERE rule_id=?)`, rule)
		db.Exec(`DELETE FROM alert_event_sources WHERE alert_id IN (SELECT id FROM alert_events WHERE rule_id=?)`, rule)
		db.Exec(`DELETE FROM alert_rule_states WHERE rule_id=?`, rule)
		db.Exec(`DELETE FROM alert_events WHERE rule_id=?`, rule)
		db.Exec(`DELETE h FROM alert_job_history h JOIN alert_evaluation_jobs j ON j.id=h.job_id WHERE j.vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_evaluation_jobs WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_input_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_intake_rejections WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_rules WHERE id=?`, rule)
		db.Exec(`DELETE FROM event_severities WHERE code=?`, prefix)
		db.Exec(`DELETE FROM users WHERE id=?`, user)
		db.Exec(`DELETE FROM vehicles WHERE id=?`, vehicle)
	})
	return f
}

func fixtureEvent(f fixture, key string, quality telemetry.TimestampQuality) Event {
	now := time.Now().UTC().Truncate(time.Millisecond)
	var eventTime *time.Time
	if quality == telemetry.OriginalDeviceTime || quality == telemetry.CorrelatedTime {
		eventTime = &now
	}
	return Event{Key: f.prefix + key, Type: "GPS_POSITION", VehicleID: f.vehicle, SourceType: "GPS", SourceIdentifier: "fixture", EventTime: eventTime, ReceivedAt: now.Add(time.Second), TimestampQuality: quality, ResolutionStatus: telemetry.Resolved, Payload: json.RawMessage(`{"fixture":true}`)}
}

func TestIntakeQueueWorkerNoEvaluatorAndRetry(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	intake := Intake{DB: f.db, MaxAttempts: 2}
	event, job, err := intake.Enqueue(ctx, fixtureEvent(f, "no-rule", telemetry.ReceivedTimeOnly))
	if err != nil {
		t.Fatal(err)
	}
	duplicate, duplicateJob, err := intake.Enqueue(ctx, fixtureEvent(f, "no-rule", telemetry.ReceivedTimeOnly))
	if err != nil {
		t.Fatal(err)
	}
	if event.ID != duplicate.ID || job.ID != duplicateJob.ID {
		t.Fatal("duplicate enqueue created new records")
	}
	queue := Queue{DB: f.db, LockTimeout: time.Millisecond}
	deadline, err := queue.NextWake(ctx)
	if err != nil || deadline == nil || deadline.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("pending job deadline=%v err=%v", deadline, err)
	}
	worker := Worker{Queue: queue, DB: f.db, Registry: NewRegistry(), Episodes: EpisodeManager{DB: f.db}, ID: "worker-no-rule"}
	worked, err := worker.RunOnce(ctx)
	if err != nil || !worked {
		t.Fatalf("RunOnce=%v,%v", worked, err)
	}
	var state JobState
	var reason sql.NullString
	if err = f.db.QueryRow(`SELECT state,last_error_code FROM alert_evaluation_jobs WHERE id=?`, job.ID).Scan(&state, &reason); err != nil {
		t.Fatal(err)
	}
	if state != JobSuccess {
		t.Fatalf("state=%s", state)
	}
	var alerts int
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=?`, f.vehicle).Scan(&alerts); err != nil || alerts != 0 {
		t.Fatalf("alerts=%d err=%v", alerts, err)
	}

	_, retryJob, err := intake.Enqueue(ctx, fixtureEvent(f, "retry", telemetry.OriginalDeviceTime))
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := queue.Claim(ctx, "worker-retry")
	if err != nil || claimed.ID != retryJob.ID {
		t.Fatalf("claim=%v err=%v", claimed, err)
	}
	if err = queue.Fail(ctx, *claimed, "TEMPORARY", "safe failure", true); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT state FROM alert_evaluation_jobs WHERE id=?`, retryJob.ID).Scan(&state); err != nil || state != JobRetry {
		t.Fatalf("retry state=%s err=%v", state, err)
	}
	f.db.Exec(`UPDATE alert_evaluation_jobs SET available_at=UTC_TIMESTAMP(3) WHERE id=?`, retryJob.ID)
	claimed, err = queue.Claim(ctx, "worker-final")
	if err != nil {
		t.Fatal(err)
	}
	if err = queue.Fail(ctx, *claimed, "PERMANENT", "safe failure", true); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT state FROM alert_evaluation_jobs WHERE id=?`, retryJob.ID).Scan(&state); err != nil || state != JobFailed {
		t.Fatalf("failed state=%s err=%v", state, err)
	}
}

func TestEpisodeDedupLifecycleProvenanceAndHistory(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	intake := Intake{DB: f.db}
	first, _, err := intake.Enqueue(ctx, fixtureEvent(f, "trigger", telemetry.OriginalDeviceTime))
	if err != nil {
		t.Fatal(err)
	}
	manager := EpisodeManager{DB: f.db}
	evaluator := testEvaluator{id: f.rule, action: Trigger, scope: "vehicle"}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- manager.Process(ctx, evaluator, first) }()
	}
	wg.Wait()
	close(errs)
	for err = range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var alertID int64
	var count int
	if err = f.db.QueryRow(`SELECT MIN(id),COUNT(*) FROM alert_events WHERE rule_id=? AND vehicle_id=?`, f.rule, f.vehicle).Scan(&alertID, &count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("episodes=%d", count)
	}
	second, _, err := intake.Enqueue(ctx, fixtureEvent(f, "support", telemetry.ReceivedTimeOnly))
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.Process(ctx, evaluator, second); err != nil {
		t.Fatal(err)
	}
	var sources int
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM alert_event_sources WHERE alert_id=?`, alertID).Scan(&sources); err != nil || sources != 2 {
		t.Fatalf("sources=%d err=%v", sources, err)
	}
	uid := f.user
	if err = (Lifecycle{DB: f.db}).Transition(ctx, alertID, AlertAcknowledged, Actor{Type: "USER", UserID: &uid}, "reviewed"); err != nil {
		t.Fatal(err)
	}
	if err = (Lifecycle{DB: f.db}).Transition(ctx, alertID, AlertResolved, SystemActor, "recovered"); err != nil {
		t.Fatal(err)
	}
	if err = (Lifecycle{DB: f.db}).Transition(ctx, alertID, AlertActive, SystemActor, ""); err == nil {
		t.Fatal("invalid transition accepted")
	}
	var audits int
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM alert_audit_trail WHERE alert_id=?`, alertID).Scan(&audits); err != nil || audits < 4 {
		t.Fatalf("audits=%d err=%v", audits, err)
	}
	third, _, err := intake.Enqueue(ctx, fixtureEvent(f, "new-episode", telemetry.CorrelatedTime))
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.Process(ctx, evaluator, third); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE rule_id=? AND vehicle_id=?`, f.rule, f.vehicle).Scan(&count); err != nil || count != 2 {
		t.Fatalf("episodes after recovery=%d err=%v", count, err)
	}
}

func TestConcurrentWorkersClaimDistinctJobs(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	intake := Intake{DB: f.db}
	for _, key := range []string{"a", "b"} {
		if _, _, err := intake.Enqueue(ctx, fixtureEvent(f, key, telemetry.OriginalDeviceTime)); err != nil {
			t.Fatal(err)
		}
	}
	queue := Queue{DB: f.db}
	jobs := make(chan int64, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []string{"w1", "w2"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			job, err := queue.Claim(ctx, id)
			if err == nil && job != nil {
				jobs <- job.ID
			}
			errs <- err
		}(id)
	}
	wg.Wait()
	close(jobs)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[int64]bool{}
	for id := range jobs {
		seen[id] = true
	}
	if len(seen) != 2 {
		t.Fatalf("claimed distinct jobs=%d", len(seen))
	}
}

func TestIntakeRejectsMissingVehicleAndQueueRecoversExpiredLease(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	intake := Intake{DB: f.db, MaxAttempts: 3}
	missing := fixtureEvent(f, "missing", telemetry.OriginalDeviceTime)
	missing.VehicleID = f.vehicle + 999999999
	t.Cleanup(func() {
		f.db.Exec(`DELETE FROM alert_intake_rejections WHERE event_key=UNHEX(SHA2(?,256))`, missing.Key)
	})
	if _, _, err := intake.Enqueue(ctx, missing); err == nil {
		t.Fatal("missing vehicle accepted")
	}
	var rejected int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM alert_intake_rejections WHERE event_key=UNHEX(SHA2(?,256))`, missing.Key).Scan(&rejected); err != nil || rejected != 1 {
		t.Fatalf("rejections=%d err=%v", rejected, err)
	}

	_, created, err := intake.Enqueue(ctx, fixtureEvent(f, "expired", telemetry.OriginalDeviceTime))
	if err != nil {
		t.Fatal(err)
	}
	queue := Queue{DB: f.db, LockTimeout: time.Second}
	claimed, err := queue.Claim(ctx, "dead-worker")
	if err != nil || claimed == nil || claimed.ID != created.ID {
		t.Fatalf("initial claim=%v err=%v", claimed, err)
	}
	if _, err = f.db.Exec(`UPDATE alert_evaluation_jobs SET locked_at=UTC_TIMESTAMP(3)-INTERVAL 5 SECOND WHERE id=?`, created.ID); err != nil {
		t.Fatal(err)
	}
	deadline, err := queue.NextWake(ctx)
	if err != nil || deadline == nil || deadline.After(time.Now().UTC()) {
		t.Fatalf("expired lease deadline=%v err=%v", deadline, err)
	}
	recovered, err := queue.Claim(ctx, "replacement-worker")
	if err != nil || recovered == nil || recovered.ID != created.ID {
		t.Fatalf("recovered=%v err=%v", recovered, err)
	}
	if recovered.Attempts != 2 {
		t.Fatalf("attempts=%d", recovered.Attempts)
	}
	var reason string
	if err = f.db.QueryRow(`SELECT reason_code FROM alert_job_history WHERE job_id=? ORDER BY id DESC LIMIT 1`, created.ID).Scan(&reason); err != nil || reason != "EXPIRED_LOCK_RECOVERED" {
		t.Fatalf("reason=%q err=%v", reason, err)
	}
}
