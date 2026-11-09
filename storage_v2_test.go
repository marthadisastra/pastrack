package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"pastrack/internal/database"
	"strings"
	"testing"
	"time"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("set PASTRACK_TEST_DSN for isolated local integration database")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	var name string
	if e = db.QueryRow("SELECT DATABASE()").Scan(&name); e != nil {
		t.Fatal(e)
	}
	if e = database.LocalTarget("127.0.0.1", name); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(dsn, "127.0.0.1:33307") {
		t.Fatal("integration test requires isolated loopback port 33307")
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func TestV2RawAndPositionIdempotency(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()
	suffix := fmt.Sprint(time.Now().UnixNano())
	model, e := db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", suffix)
	if e != nil {
		t.Fatal(e)
	}
	modelID, _ := model.LastInsertId()
	imei := "000" + suffix[len(suffix)-12:]
	device, e := db.Exec("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, modelID)
	if e != nil {
		t.Fatal(e)
	}
	deviceID, _ := device.LastInsertId()
	payload := strings.Replace(sample, "861252080093813", imei, 1)
	receipt := "qa-" + suffix
	r := Record{ReceiptID: receipt, Payload: payload, Received: time.Now().UTC()}
	for i := 0; i < 2; i++ {
		if e = saveV2(ctx, db, r); e != nil {
			t.Fatal(e)
		}
	}
	r.ReceiptID += "-network-repeat"
	if e = saveV2(ctx, db, r); e != nil {
		t.Fatal(e)
	}
	var rawCount, positionCount int
	db.QueryRow("SELECT COUNT(*) FROM device_raw_messages WHERE device_id=?", deviceID).Scan(&rawCount)
	db.QueryRow("SELECT COUNT(*) FROM gps_positions WHERE device_id=?", deviceID).Scan(&positionCount)
	if rawCount != 2 || positionCount != 1 {
		t.Fatalf("raw=%d positions=%d", rawCount, positionCount)
	}
	var outboxCount int
	if e = db.QueryRow("SELECT COUNT(*) FROM dashboard_change_outbox o JOIN device_raw_messages r ON r.id=o.raw_message_id WHERE r.device_id=?", deviceID).Scan(&outboxCount); e != nil || outboxCount != 2 {
		t.Fatalf("transactional dashboard events=%d error=%v", outboxCount, e)
	}

	r.ReceiptID += "-buffered"
	r.Payload = strings.Replace(payload, "+RESP:", "+BUFF:", 1)
	if e = saveV2(ctx, db, r); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow("SELECT COUNT(*) FROM gps_positions WHERE device_id=?", deviceID).Scan(&positionCount); e != nil || positionCount != 1 {
		t.Fatal("buffered duplicate normalized again", positionCount, e)
	}
	r.ReceiptID += "-malformed"
	r.Payload = "unidentified$"
	if e = saveV2(ctx, db, r); e != nil {
		t.Fatal(e)
	}
}

func TestDashboardOutboxDrainsOnPublisherStartup(t *testing.T) {
	db := integrationDB(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	model, err := db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", "outbox-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	modelID, _ := model.LastInsertId()
	imei := "009" + suffix[len(suffix)-12:]
	device, err := db.Exec("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, modelID)
	if err != nil {
		t.Fatal(err)
	}
	deviceID, _ := device.LastInsertId()
	receipt := "outbox-" + suffix
	if err = saveV2(context.Background(), db, Record{ReceiptID: receipt, Payload: strings.Replace(sample, "861252080093813", imei, 1), Received: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	dir, err := os.MkdirTemp("/tmp", "pt-outbox-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := dir + "/live.sock"
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	old := realtimeChanges
	realtimeChanges = make(chan []byte, 2048)
	defer func() { realtimeChanges = old }()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		publishChanges(ctx, db, path)
		close(done)
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("outbox publisher failed to stop")
		}
	}()
	listener.SetReadDeadline(time.Now().Add(5 * time.Second))
	matched := false
	for !matched {
		buffer := make([]byte, 512)
		n, _, readErr := listener.ReadFromUnix(buffer)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var change entityChange
		if json.Unmarshal(buffer[:n], &change) == nil && change.IMEI == imei {
			matched = true
		}
	}
	var delivered sql.NullTime
	if err = db.QueryRow("SELECT o.delivered_at FROM dashboard_change_outbox o JOIN device_raw_messages r ON r.id=o.raw_message_id WHERE r.device_id=? AND r.spool_receipt_id=?", deviceID, receipt).Scan(&delivered); err != nil || !delivered.Valid {
		t.Fatalf("durable event was not acknowledged: %v %+v", err, delivered)
	}
}
func TestSTTUTCAndSatelliteMask(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	m, e := parseMessage(sample, loc)
	if e != nil || m.Position.Satellites == nil || *m.Position.Satellites != 7 || m.Position.GPSTime.Hour() != 6 {
		t.Fatal(m, e)
	}
	raw := strings.Replace(sample, ",01,7,", ",00,", 1)
	m, e = parseMessage(raw, time.UTC)
	if e != nil || m.Position.Satellites != nil {
		t.Fatal(m, e)
	}
}

func TestV2DocumentedBLEWithoutInventedFuel(t *testing.T) {
	db := integrationDB(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	imei := "001" + suffix[len(suffix)-12:]
	mac := fmt.Sprintf("%012X", time.Now().UnixNano()&0xffffffffffff)
	model, e := db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", "ble-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	modelID, _ := model.LastInsertId()
	device, e := db.Exec("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, modelID)
	if e != nil {
		t.Fatal(e)
	}
	deviceID, _ := device.LastInsertId()
	// Document-derived fixture, not a recorded physical device frame.
	raw := "+RESP:GTERI,80201E0306," + imei + ",GV30CEU,00000100,,10,1,0,0.0,0,0.0,,,,0510,0001,7D06,0337D73E,01,0,0.0,0000000:00:00,,,,100,000000,,1,0,27,0,560104001A,0007,CFS300B," + mac + ",1,20261103000000,0001$"
	r := Record{ReceiptID: "ble-" + suffix, Payload: raw, Received: time.Now().UTC()}
	if e = saveV2(context.Background(), db, r); e != nil {
		t.Fatal(e)
	}
	var height, volume int
	var liters sql.NullFloat64
	var validity string
	e = db.QueryRow(`SELECT t.fuel_height_raw,t.volume_raw,f.fuel_liters,f.validity FROM sensor_telemetry t JOIN fuel_readings f ON f.telemetry_id=t.id WHERE t.device_id=?`, deviceID).Scan(&height, &volume, &liters, &validity)
	if e != nil {
		t.Fatal(e)
	}
	if height != 260 || volume != 26 || liters.Valid || validity != "uncalibrated" {
		t.Fatal("unverified raw became fuel", height, volume, liters, validity)
	}

	vehicle, e := db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'QA ERI',1)", "ERI-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	vehicleID, _ := vehicle.LastInsertId()
	if _, e = db.Exec("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,'2020-01-01','qa')", vehicleID, deviceID); e != nil {
		t.Fatal(e)
	}
	valid := strings.Replace(raw, "0,0.0,0,0.0,,,,", "2,20.0,90,10.0,106.0,-6.0,20261103000000,", 1)
	if e = saveV2(context.Background(), db, Record{ReceiptID: "gps-" + suffix, Payload: valid, Received: time.Now().UTC()}); e != nil {
		t.Fatal("ERI GPS persistence", e)
	}
	var inherited int
	if err := db.QueryRow("SELECT COUNT(*) FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id WHERE t.device_id=? AND f.vehicle_id=?", deviceID, vehicleID).Scan(&inherited); err != nil || inherited == 0 {
		t.Fatal("BLE did not inherit gateway vehicle", err, inherited)
	}
	var latestCount int
	if e = db.QueryRow("SELECT COUNT(*) FROM latest_positions WHERE vehicle_id=?", vehicleID).Scan(&latestCount); e != nil || latestCount != 1 {
		t.Fatal("valid ERI unavailable live", latestCount, e)
	}
	r.ReceiptID += "-disconnect"
	r.Payload = strings.Replace(raw, ",1,20261103000000", ",0,20261103000000", 1)
	if e = saveV2(context.Background(), db, r); e != nil {
		t.Fatal(e)
	}
	e = db.QueryRow(`SELECT f.validity,f.fuel_liters FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id WHERE t.device_id=? ORDER BY f.id DESC LIMIT 1`, deviceID).Scan(&validity, &liters)
	if e != nil || validity != "disconnected" || liters.Valid {
		t.Fatal(validity, liters, e)
	}
	// A reported name may change; the MAC remains the same identity. Delayed
	// packets must not restore an older name.
	renamed := strings.ReplaceAll(strings.Replace(raw, ",CFS300B,", ",BBM_SATU,", 1), "20261103000000", "20261103000100")
	for i, payload := range []string{renamed, raw} {
		if err := saveV2(context.Background(), db, Record{ReceiptID: fmt.Sprintf("rename-%s-%d", suffix, i), Payload: payload, Received: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	var name string
	var identities int
	if err := db.QueryRow("SELECT n.reported_name FROM sensor_reported_names n JOIN ble_sensors s ON s.id=n.sensor_id WHERE s.mac=?", mac).Scan(&name); err != nil || name != "BBM_SATU" {
		t.Fatal("reported name ordering", name, err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM ble_sensors WHERE mac=?", mac).Scan(&identities); err != nil || identities != 1 {
		t.Fatal("rename changed identity", identities, err)
	}
}

func TestRawSurvivesNormalizationFailure(t *testing.T) {
	db := integrationDB(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	imei := "002" + suffix[len(suffix)-12:]
	model, e := db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", "failure-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	modelID, _ := model.LastInsertId()
	device, e := db.Exec("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, modelID)
	if e != nil {
		t.Fatal(e)
	}
	deviceID, _ := device.LastInsertId()
	vehicle, e := db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?, 'QA',1)", "ERR-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	vehicleID, _ := vehicle.LastInsertId()
	for i := 0; i < 2; i++ {
		if _, e = db.Exec("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,'2020-01-01','qa_overlap')", vehicleID, deviceID); e != nil {
			t.Fatal(e)
		}
	}
	receipt := "failed-" + suffix
	e = saveV2(context.Background(), db, Record{ReceiptID: receipt, Payload: strings.Replace(sample, "861252080093813", imei, 1), Received: time.Now().UTC()})
	if e == nil {
		t.Fatal("invalid assignment accepted")
	}
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM device_raw_messages WHERE spool_receipt_id=?", receipt).Scan(&count); e != nil || count != 1 {
		t.Fatal("raw was lost", count, e)
	}
}
func TestSpoolPreservesInvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	payload := string([]byte{0xff, 0xfe, '$'})
	if e := enqueue(dir, Record{Payload: payload, Received: time.Now().UTC()}); e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(dir)
	if e != nil || len(entries) != 1 {
		t.Fatal(e)
	}
	b, e := os.ReadFile(dir + "/" + entries[0].Name())
	if e != nil {
		t.Fatal(e)
	}
	var r Record
	if e = json.Unmarshal(b, &r); e != nil {
		t.Fatal(e)
	}
	if string(r.PayloadBytes) != payload {
		t.Fatal("wire bytes mutated")
	}
}

func TestProtocol0306FixIsNotSatelliteCount(t *testing.T) {
	raw := strings.Replace(sample, "80201E0100", "80201E0306", 1)
	m, e := parseMessage(raw, time.UTC)
	if e != nil || m.Position.FixStatus != "no_fix" || m.Position.HDOP != nil {
		t.Fatal("zero accuracy was promoted", m, e)
	}
	raw = strings.Replace(raw, ",41,0,0.6", ",41,2,0.6", 1)
	m, e = parseMessage(raw, time.UTC)
	if e != nil || m.Position.FixStatus != "valid" || m.Position.HDOP == nil || *m.Position.HDOP != 2 {
		t.Fatal(m, e)
	}
}
func TestACKEnvelopeDoesNotInventDeviceFields(t *testing.T) {
	for _, raw := range []string{"+SACK:0001$", "+ACK:GTFOO,unidentified,fields$"} {
		m, e := parseMessage(raw, time.UTC)
		if e != nil || m.IMEI != "" || m.Position != nil {
			t.Fatal(m, e)
		}
	}
}

func TestV2CrashSignalCreatesNormalizedJob(t *testing.T) {
	db := integrationDB(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	imei := "004" + suffix[len(suffix)-12:]
	modelResult, e := db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('Queclink','GV30CEU','@Track') ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)")
	if e != nil {
		t.Fatal(e)
	}
	model, _ := modelResult.LastInsertId()
	deviceResult, e := db.Exec("INSERT INTO gps_devices(imei,model_id,firmware_version,protocol_version,is_active) VALUES(?,?,'R01A03V21','80201E0306',1)", imei, model)
	if e != nil {
		t.Fatal(e)
	}
	device, _ := deviceResult.LastInsertId()
	vehicleResult, e := db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Crash receiver QA',1)", "CRA-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	vehicle, _ := vehicleResult.LastInsertId()
	if _, e = db.Exec("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,'2026-11-05 00:00:00','phase5.2-test')", vehicle, device); e != nil {
		t.Fatal(e)
	}
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
		db.Exec(`DELETE FROM vehicle_assignments WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM gps_devices WHERE id=?`, device)
		db.Exec(`DELETE FROM vehicles WHERE id=?`, vehicle)
	})
	payload := strings.Replace(observedCrashFixture, "000000000000000", imei, 1)
	if e = saveV2(context.Background(), db, Record{ReceiptID: "crash-" + suffix, Payload: payload, Received: time.Date(2026, 11, 5, 6, 15, 37, 0, time.UTC)}); e != nil {
		t.Fatal(e)
	}
	var signals, jobs int
	if e = db.QueryRow("SELECT COUNT(*) FROM crash_signal_events WHERE device_id=? AND vehicle_id=? AND message_type='GTCRA'", device, vehicle).Scan(&signals); e != nil || signals != 1 {
		t.Fatalf("signals=%d err=%v", signals, e)
	}
	if e = db.QueryRow("SELECT COUNT(*) FROM alert_evaluation_jobs WHERE vehicle_id=?", vehicle).Scan(&jobs); e != nil || jobs != 1 {
		t.Fatalf("jobs=%d err=%v", jobs, e)
	}
	if _, e = db.Exec("UPDATE alert_evaluation_jobs SET state='SUCCESS',processed_at=UTC_TIMESTAMP(3) WHERE vehicle_id=?", vehicle); e != nil {
		t.Fatal(e)
	}
}

const observedCrashFixture = "+RESP:GTCRA,80201E0306,000000000000000,,00,0,,,,,,,0510,0001,7D06,0337D716,00,20261105061536,992E$"
