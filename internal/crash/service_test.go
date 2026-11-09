package crash_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"pastrack/internal/alertengine"
	"pastrack/internal/alertrules"
	crashsvc "pastrack/internal/crash"
	"pastrack/internal/protocol"
)

func TestCrashDataBeforeAlarmCorrelatesAndDoesNotDuplicate(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("PASTRACK_TEST_DSN not set")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	prefix := fmt.Sprintf("CR-%d", time.Now().UnixNano())
	at := time.Date(2026, 11, 5, 6, 15, 36, 0, time.UTC)
	result, err := db.Exec(`INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Crash test',1)`, prefix)
	if err != nil {
		t.Fatal(err)
	}
	vehicle, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('Queclink',?,'@Track')`, prefix)
	if err != nil {
		t.Fatal(err)
	}
	model, _ := result.LastInsertId()
	imei := fmt.Sprintf("%015d", time.Now().UnixNano()%1000000000000000)
	result, err = db.Exec(`INSERT INTO gps_devices(imei,model_id,firmware_version,protocol_version,is_active) VALUES(?,?,'R01A03V21','80201E0306',1)`, imei, model)
	if err != nil {
		t.Fatal(err)
	}
	device, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'phase5.2-test')`, vehicle, device, at.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	assignment, _ := result.LastInsertId()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM alert_audit_trail WHERE alert_id IN (SELECT id FROM alert_events WHERE vehicle_id=?)`, vehicle)
		db.Exec(`DELETE FROM alert_event_sources WHERE alert_id IN (SELECT id FROM alert_events WHERE vehicle_id=?)`, vehicle)
		db.Exec(`DELETE FROM alert_rule_states WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE h FROM alert_job_history h JOIN alert_evaluation_jobs j ON j.id=h.job_id WHERE j.vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_evaluation_jobs WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_input_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM crash_signal_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM crash_evidence_sets WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM device_raw_messages WHERE device_id=?`, device)
		db.Exec(`DELETE FROM vehicle_assignments WHERE id=?`, assignment)
		db.Exec(`DELETE FROM gps_devices WHERE id=?`, device)
		db.Exec(`DELETE FROM gps_device_models WHERE id=?`, model)
		db.Exec(`DELETE FROM vehicles WHERE id=?`, vehicle)
	})
	insertRaw := func(payload string) int64 {
		r, e := db.Exec(`INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,UNHEX(SHA2(?,256)),?)`, device, payload, payload, at)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	data := strings.Repeat("0001FFFF0052", 3)
	crdPayload := fmt.Sprintf("+RESP:GTCRD,80201E0306,%s,,00,18,1,1,%s,20261105061536,992F$", imei, data)
	crdReport, err := protocol.DecodeCrash(crdPayload)
	if err != nil {
		t.Fatal(err)
	}
	service := crashsvc.Service{DB: db}
	crd, err := service.Process(ctx, insertRaw(crdPayload), device, at, crdReport)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := alertrules.LoadEnabled(ctx, db, func() time.Time { return at })
	if err != nil {
		t.Fatal(err)
	}
	intake := alertengine.Intake{DB: db}
	worker := alertengine.Worker{Queue: alertengine.Queue{DB: db}, DB: db, Registry: registry, Episodes: alertengine.EpisodeManager{DB: db, Now: func() time.Time { return at }}, ID: "crash-test"}
	if _, _, err = intake.Enqueue(ctx, alertengine.EventFromCrash(crd)); err != nil {
		t.Fatal(err)
	}
	if _, err = worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=? AND rule_code='CRASH'`, vehicle).Scan(&count); err != nil || count != 0 {
		t.Fatalf("GTCRD created alert count=%d err=%v", count, err)
	}
	craPayload := fmt.Sprintf("+RESP:GTCRA,80201E0306,%s,,00,0,,,,,,,0510,0001,7D06,0337D716,00,20261105061536,992E$", imei)
	craReport, err := protocol.DecodeCrash(craPayload)
	if err != nil {
		t.Fatal(err)
	}
	craRaw := insertRaw(craPayload)
	cra, err := service.Process(ctx, craRaw, device, at, craReport)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.Process(ctx, craRaw, device, at, craReport)
	if err != nil || duplicate.SignalEventID != cra.SignalEventID || duplicate.SequenceKey != cra.SequenceKey {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	if _, _, err = intake.Enqueue(ctx, alertengine.EventFromCrash(cra)); err != nil {
		t.Fatal(err)
	}
	if _, err = worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var alertID int64
	if err = db.QueryRow(`SELECT id FROM alert_events WHERE vehicle_id=? AND rule_code='CRASH'`, vehicle).Scan(&alertID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_event_sources WHERE alert_id=?`, alertID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("sources=%d err=%v", count, err)
	}
	afterPayload := strings.Replace(crdPayload, ",18,1,1,", ",19,1,1,", 1)
	afterReport, err := protocol.DecodeCrash(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	after, err := service.Process(ctx, insertRaw(afterPayload), device, at.Add(time.Second), afterReport)
	if err != nil {
		t.Fatal(err)
	}
	if after.EvidenceStatus != "COMPLETE" {
		t.Fatalf("evidence=%s", after.EvidenceStatus)
	}
	if _, _, err = intake.Enqueue(ctx, alertengine.EventFromCrash(after)); err != nil {
		t.Fatal(err)
	}
	if _, err = worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_event_sources WHERE alert_id=?`, alertID).Scan(&count); err != nil || count != 3 {
		t.Fatalf("completed sources=%d err=%v", count, err)
	}
	var description string
	if err = db.QueryRow(`SELECT description FROM alert_events WHERE id=?`, alertID).Scan(&description); err != nil || !strings.Contains(description, "lengkap") {
		t.Fatalf("description=%q err=%v", description, err)
	}
	// A second primary with the same counter is deliberately a distinct sequence.
	cra2Payload := strings.Replace(craPayload, "20261105061536,992E", "20261105061736,993E", 1)
	cra2Report, err := protocol.DecodeCrash(cra2Payload)
	if err != nil {
		t.Fatal(err)
	}
	cra2, err := service.Process(ctx, insertRaw(cra2Payload), device, at.Add(2*time.Minute), cra2Report)
	if err != nil {
		t.Fatal(err)
	}
	if cra2.SequenceKey == cra.SequenceKey {
		t.Fatal("two primary crashes merged")
	}
}
