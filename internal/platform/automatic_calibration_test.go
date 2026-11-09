package platform

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCalibrationStablePhysicalWindow(t *testing.T) {
	now := time.Now().UTC()
	phase := now.Add(-20 * time.Second)
	sample := func(id int64, raw float64, seconds int) stableCalibrationSample {
		at := now.Add(time.Duration(-seconds) * time.Second)
		return stableCalibrationSample{ID: id, Raw: raw, Received: at, Measured: at}
	}
	normal := []stableCalibrationSample{sample(3, 100, 1), sample(2, 101, 6), sample(1, 100, 11)}
	if result := calibrationStability(normal, now, phase); !result.Ready || result.Count != 3 {
		t.Fatal(result)
	}
	unstable := append([]stableCalibrationSample(nil), normal...)
	unstable[1].Raw = 110
	if calibrationStability(unstable, now, phase).Ready {
		t.Fatal("changing signal accepted")
	}
	buffered := append([]stableCalibrationSample(nil), normal...)
	for i := range buffered {
		buffered[i].Received = now
	}
	if calibrationStability(buffered, now, phase).Ready {
		t.Fatal("buffer batch accepted as elapsed stability")
	}
	duplicate := append([]stableCalibrationSample(nil), normal...)
	for i := range duplicate {
		duplicate[i].Measured = now.Add(-time.Second)
	}
	if calibrationStability(duplicate, now, phase).Ready {
		t.Fatal("duplicate measurements counted")
	}
	if calibrationStability(normal, now, now).Ready {
		t.Fatal("pre-start data accepted")
	}
}

func TestAutomaticCalibrationSensorIsolationAndAtomicCompletion(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database required")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insert := func(query string, args ...any) int64 {
		t.Helper()
		r, e := db.Exec(query, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	imei := suffix[len(suffix)-15:]
	mac := fmt.Sprintf("%012X", time.Now().UnixNano()&0xffffffffffff)
	actor := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'QA','unused','bcrypt',1,1,1)", "auto-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'Queclink')", suffix)
	gps := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, model)
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'QA calibration',1)", "AUTO-"+suffix)
	insert("INSERT INTO vehicle_inventory(vehicle_id,tank_capacity_l) VALUES(?,200)", vehicle)
	insert("INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 DAY,'qa')", gps, vehicle)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1)", mac, mac)
	fs := insert("INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'raw_units_unverified')", sensor)
	sample := func(device int64, deviceIMEI string, raw float64, seconds int) {
		at := time.Now().UTC().Add(-time.Duration(seconds) * time.Second)
		payload := fmt.Sprintf("+RESP:GTERI,80201E0306,%s,,qa-%s-%d$", deviceIMEI, suffix, time.Now().UnixNano())
		receipt := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at,device_time) VALUES(?,?,UNHEX(SHA2(?,256)),?,?)", device, payload, payload, at, at)
		insert("INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,sensor_time,fuel_height_raw,validation_state,decoder_version) VALUES(?,?,?,0,?,?,?,'valid_raw','qa')", sensor, device, receipt, at, at, raw)
	}
	sample(gps, imei, 100, 0)
	s := New(db)
	var session int64
	call := func(action string) map[string]any {
		t.Helper()
		payload := fmt.Sprintf(`{"action":%q,"session_id":%d}`, action, session)
		w := httptest.NewRecorder()
		s.api(w, httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sensors/%d/calibration-session", sensor), strings.NewReader(payload)), &Identity{ID: actor, Role: "admin"})
		if w.Code != 200 {
			t.Fatal(action, w.Code, w.Body)
		}
		var response struct{ Data map[string]any }
		if e := json.Unmarshal(w.Body.Bytes(), &response); e != nil {
			t.Fatal(e)
		}
		session = int64(response.Data["session_id"].(float64))
		return response.Data
	}
	if call("start")["state"] != "empty" {
		t.Fatal("did not start")
	}
	db.Exec("UPDATE sensor_calibration_sessions SET phase_started_at=UTC_TIMESTAMP(3)-INTERVAL 20 SECOND WHERE id=?", session)
	for _, seconds := range []int{15, 10, 5} {
		sample(gps, imei, 100, seconds)
	}
	if call("progress")["state"] != "fill" {
		t.Fatal("empty not captured")
	}
	if call("progress")["state"] != "fill" {
		t.Fatal("full captured without confirmation")
	}
	if call("confirm_full")["state"] != "full" {
		t.Fatal("full confirmation not retained")
	}
	db.Exec("UPDATE sensor_calibration_sessions SET phase_started_at=UTC_TIMESTAMP(3)-INTERVAL 20 SECOND,phase_telemetry_id=(SELECT MAX(id) FROM sensor_telemetry WHERE sensor_id=?) WHERE id=?", sensor, session)
	for _, seconds := range []int{15, 10, 5} {
		sample(gps, imei, 1000, seconds)
	}
	completed := call("progress")
	if completed["state"] != "complete" || completed["capacity_l"] != float64(200) {
		t.Fatal(completed)
	}
	call("progress")
	var count int
	db.QueryRow("SELECT COUNT(*) FROM fuel_calibrations WHERE fuel_sensor_id=? AND approval_state='approved'", fs).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate completion", count)
	}
	var full float64
	db.QueryRow("SELECT MAX(p.output_value) FROM fuel_calibration_points p JOIN fuel_calibrations c ON c.id=p.calibration_id WHERE c.fuel_sensor_id=?", fs).Scan(&full)
	if full != 200 {
		t.Fatal("wrong vehicle capacity", full)
	}
	if call("start")["state"] != "empty" {
		t.Fatal("restart failed")
	}
	otherIMEI := "1" + imei[1:]
	if otherIMEI == imei {
		otherIMEI = "2" + imei[1:]
	}
	other := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", otherIMEI, model)
	sample(other, otherIMEI, 300, 0)
	if call("progress")["state"] != "failed" {
		t.Fatal("changed GPS was accepted")
	}
}
