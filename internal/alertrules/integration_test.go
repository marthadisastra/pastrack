package alertrules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"pastrack/internal/alertengine"
	"pastrack/internal/telemetry"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("PASTRACK_TEST_DSN not set")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("integration test requires isolated pastrack_test_local")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func cleanupBusiness(t *testing.T, db *sql.DB, vehicle int64) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM alert_audit_trail WHERE alert_id IN (SELECT id FROM alert_events WHERE vehicle_id=?)`, vehicle)
		db.Exec(`DELETE FROM alert_event_sources WHERE alert_id IN (SELECT id FROM alert_events WHERE vehicle_id=?)`, vehicle)
		db.Exec(`DELETE FROM alert_rule_states WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE h FROM alert_job_history h JOIN alert_evaluation_jobs j ON j.id=h.job_id WHERE j.vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_evaluation_jobs WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM alert_input_events WHERE vehicle_id=?`, vehicle)
		db.Exec(`DELETE FROM vehicles WHERE id=?`, vehicle)
	})
}

func TestConfiguredRulesThroughPhase4Worker(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()
	prefix := fmt.Sprintf("P5-%d", time.Now().UnixNano())
	result, err := db.Exec(`INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Phase 5 test',1)`, prefix)
	if err != nil {
		t.Fatal(err)
	}
	vehicle, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'QA',1)`, prefix+"-sensor", fmt.Sprintf("%012X", time.Now().UnixNano()&0xffffffffffff))
	if err != nil {
		t.Fatal(err)
	}
	sensor, _ := result.LastInsertId()
	t.Cleanup(func() { db.Exec(`DELETE FROM ble_sensors WHERE id=?`, sensor) })
	result, err = db.Exec(`INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'QA')`, prefix)
	if err != nil {
		t.Fatal(err)
	}
	model, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)`, fmt.Sprintf("%015d", time.Now().UnixNano()%1000000000000000), model)
	if err != nil {
		t.Fatal(err)
	}
	device, _ := result.LastInsertId()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM gps_devices WHERE id=?`, device)
		db.Exec(`DELETE FROM gps_device_models WHERE id=?`, model)
	})
	result, err = db.Exec(`INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'calibrated')`, sensor)
	if err != nil {
		t.Fatal(err)
	}
	fuelSensor, _ := result.LastInsertId()
	var rawIDs, telemetryIDs, readingIDs []int64
	t.Cleanup(func() {
		for _, id := range readingIDs {
			db.Exec(`DELETE FROM fuel_readings WHERE id=?`, id)
		}
		for _, id := range telemetryIDs {
			db.Exec(`DELETE FROM sensor_telemetry WHERE id=?`, id)
		}
		for _, id := range rawIDs {
			db.Exec(`DELETE FROM device_raw_messages WHERE id=?`, id)
		}
		db.Exec(`DELETE FROM fuel_sensors WHERE id=?`, fuelSensor)
	})
	cleanupBusiness(t, db, vehicle)

	registry, err := LoadEnabled(ctx, db, fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	intake := alertengine.Intake{DB: db}
	worker := alertengine.Worker{Queue: alertengine.Queue{DB: db}, DB: db, Registry: registry, Episodes: alertengine.EpisodeManager{DB: db, Now: fixedClock}, ID: "phase5-test"}
	enqueueFuel := func(sequence int64, percent float64, quality telemetry.TimestampQuality) {
		at := testNow.Add(time.Duration(sequence-4) * time.Minute)
		payloadText := prefix + fmt.Sprintf(":raw:%d", sequence)
		result, err := db.Exec(`INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,UNHEX(SHA2(?,256)),?)`, device, payloadText, payloadText, testNow)
		if err != nil {
			t.Fatal(err)
		}
		rawID, _ := result.LastInsertId()
		rawIDs = append(rawIDs, rawID)
		result, err = db.Exec(`INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,validation_state,decoder_version) VALUES(?,?,?,0,?,'valid_raw','phase5.1-test')`, sensor, device, rawID, testNow)
		if err != nil {
			t.Fatal(err)
		}
		telemetryID, _ := result.LastInsertId()
		telemetryIDs = append(telemetryIDs, telemetryID)
		result, err = db.Exec(`INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,received_at,fuel_percent,validity) VALUES(?,?,?,?,?,'valid')`, fuelSensor, telemetryID, vehicle, testNow, percent)
		if err != nil {
			t.Fatal(err)
		}
		readingID, _ := result.LastInsertId()
		readingIDs = append(readingIDs, readingID)
		event := fuelEvent(readingID, vehicle, sensor, percent, quality, at)
		event.Key = prefix + fmt.Sprintf(":fuel:%d", sequence)
		event.FuelReadingID = ptr(readingID)
		event.SensorTelemetryID = ptr(telemetryID)
		event.RawMessageID = ptr(rawID)
		var value telemetry.NormalizedFuel
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			t.Fatal(err)
		}
		value.SensorID = sensor
		value.FuelReadingID = readingID
		value.TelemetryID = telemetryID
		value.RawMessageID = rawID
		value.ReceiverDeviceID = device
		event.DeviceID = ptr(device)
		event.Payload, _ = json.Marshal(value)
		if _, _, err := intake.Enqueue(ctx, event); err != nil {
			t.Fatal(err)
		}
		worked, err := worker.RunOnce(ctx)
		if err != nil || !worked {
			t.Fatalf("fuel worker=%v err=%v", worked, err)
		}
	}
	enqueueFuel(1, 9, telemetry.CorrelatedTime)
	enqueueFuel(2, 5, telemetry.CorrelatedTime)
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=? AND rule_code='FUEL_LOW' AND state='active'`, vehicle).Scan(&count); err != nil || count != 1 {
		t.Fatalf("fuel low=%d err=%v", count, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=? AND rule_code='FUEL_DROP'`, vehicle).Scan(&count); err != nil || count != 0 {
		t.Fatalf("fuel drop before minimum samples=%d err=%v", count, err)
	}
	enqueueFuel(3, 1, telemetry.CorrelatedTime)
	var description string
	var alertID int64
	if err = db.QueryRow(`SELECT id,description FROM alert_events WHERE vehicle_id=? AND rule_code='FUEL_DROP' AND state='active'`, vehicle).Scan(&alertID, &description); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(description, "Confidence MEDIUM") || !strings.Contains(description, "waktu GPS yang berkorelasi") {
		t.Fatalf("description=%q", description)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_event_sources WHERE alert_id=?`, alertID).Scan(&count); err != nil || count != 3 {
		t.Fatalf("fuel drop provenance=%d err=%v", count, err)
	}

	enqueueGPS := func(id int64, speed float64, offset time.Duration) {
		event := gpsEvent(id, vehicle, device, speed, testNow.Add(offset), true)
		event.Key = prefix + fmt.Sprintf(":gps:%d", id)
		if _, _, err := intake.Enqueue(ctx, event); err != nil {
			t.Fatal(err)
		}
		worked, err := worker.RunOnce(ctx)
		if err != nil || !worked {
			t.Fatalf("GPS worker=%v err=%v", worked, err)
		}
	}
	enqueueGPS(101, 81, -20*time.Second)
	enqueueGPS(102, 85, -10*time.Second)
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=? AND rule_code='OVER_SPEED' AND state='active'`, vehicle).Scan(&count); err != nil || count != 1 {
		t.Fatalf("overspeed=%d err=%v", count, err)
	}
	enqueueGPS(103, 74, -5*time.Second)
	if err = db.QueryRow(`SELECT COUNT(*) FROM alert_events WHERE vehicle_id=? AND rule_code='OVER_SPEED' AND state='resolved'`, vehicle).Scan(&count); err != nil || count != 1 {
		t.Fatalf("resolved overspeed=%d err=%v", count, err)
	}
}
