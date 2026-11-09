package platform

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"pastrack/internal/fuel"
	"strings"
	"testing"
	"time"
)

func TestSimpleSetupLifecycle(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database required")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	insert := func(q string, a ...any) int64 {
		t.Helper()
		r, e := db.Exec(q, a...)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused','bcrypt',1,1,1)", "quick-"+suffix, "Isolated QA")
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'QA','NOT_INSTALLED',1)", "QS-"+suffix)
	insert("INSERT INTO vehicle_inventory(vehicle_id,tank_capacity_l) VALUES(?,200)", vehicle)
	other := insert("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'QA','NOT_INSTALLED',1)", "QO-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'Queclink')", "Q"+suffix)
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "Q"+suffix, model)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,model,is_active) VALUES(?,'CFS300B',1)", "Q"+suffix)
	fs := insert("INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'raw_units_unverified')", sensor)
	// Sensor is discovered through its gateway; the first physical capture binds it.
	s := New(db)
	admin := &Identity{ID: user, Role: "admin"}
	request := func(path, method, payload string, identity *Identity) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/"+path, strings.NewReader(payload))
		if !s.simpleSetup(w, r, identity, path) {
			t.Fatal("unhandled")
		}
		return w
	}
	cp := fmt.Sprintf("devices/%d/connect-vehicle", device)
	qp := fmt.Sprintf("sensors/%d/quick-calibration", sensor)
	for i := 0; i < 2; i++ {
		w := request(cp, "POST", fmt.Sprintf(`{"vehicle_id":%d}`, vehicle), admin)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := request(cp, "POST", fmt.Sprintf(`{"vehicle_id":%d}`, other), admin); w.Code != 409 {
		t.Fatal("accepted conflicting mapping", w.Code, w.Body.String())
	}
	if w := request(qp, "POST", `{"action":"empty","quantity":"volume_raw"}`, &Identity{ID: user, Role: "operator"}); w.Code != 403 {
		t.Fatal("operator activated calibration", w.Code)
	}
	sample := func(value int, stale bool) {
		t.Helper()
		sum := sha256.Sum256([]byte(fmt.Sprint(value, suffix, stale)))
		raw := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at,device_time) VALUES(?,'qa',?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3))", device, sum[:])
		if stale {
			db.Exec("UPDATE device_raw_messages SET device_time=UTC_TIMESTAMP(3)-INTERVAL 1 DAY WHERE id=?", raw)
		}
		insert("INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,volume_raw,validation_state,decoder_version) VALUES(?,?,?,0,UTC_TIMESTAMP(3),?,'valid_raw','qa')", sensor, device, raw, value)
	}
	sample(100, true)
	if w := request(qp, "POST", `{"action":"empty","quantity":"volume_raw"}`, admin); w.Code != 409 {
		t.Fatal("accepted delayed sample", w.Code, w.Body.String())
	}
	sample(100, false)
	w := request(qp, "POST", `{"action":"empty","quantity":"volume_raw"}`, admin)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Data struct {
			ID int64 `json:"calibration_id"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	cid := result.Data.ID
	if cid == 0 {
		t.Fatal("missing draft")
	}
	payload := func(action string, capacity int) string {
		return fmt.Sprintf(`{"action":%q,"calibration_id":%d,"quantity":"volume_raw","capacity_l":%d}`, action, cid, capacity)
	}
	if w = request(qp, "POST", payload("full", 0), admin); w.Code != 409 {
		t.Fatal("accepted same telemetry", w.Code)
	}
	if w = request(qp, "POST", payload("save", 200), admin); w.Code != 422 {
		t.Fatal("accepted missing full", w.Code)
	}
	sample(900, false)
	if w = request(qp, "POST", payload("full", 0), admin); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(qp, "GET", "", admin)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "100.000000") || !strings.Contains(w.Body.String(), "900.000000") {
		t.Fatal("draft did not survive reopen", w.Body.String())
	}
	if w = request(qp, "POST", payload("save", 999), admin); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var state string
	var tank int64
	if e = db.QueryRow("SELECT approval_state,tank_profile_id FROM fuel_calibrations WHERE id=?", cid).Scan(&state, &tank); e != nil || state != "approved" {
		t.Fatal(e, state)
	}
	var duplicateColumns int
	if e = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='fuel_tank_profiles' AND column_name='capacity_l'").Scan(&duplicateColumns); e != nil || duplicateColumns != 0 {
		t.Fatal("tank profile still owns duplicate capacity", duplicateColumns, e)
	}
	var assignedTank int64
	db.QueryRow("SELECT tank_profile_id FROM fuel_sensors WHERE id=?", fs).Scan(&assignedTank)
	if tank != assignedTank {
		t.Fatal("receiver would not use calibration")
	}
	c := fuel.Calibration{Approved: true, Media: "fuel_tank", Quantity: "volume_raw", Capacity: 200}
	rows, e := db.Query("SELECT input_value,output_value FROM fuel_calibration_points WHERE calibration_id=? ORDER BY input_value", cid)
	if e != nil {
		t.Fatal(e)
	}
	for rows.Next() {
		var p fuel.Point
		rows.Scan(&p.Raw, &p.Liters)
		c.Points = append(c.Points, p)
	}
	rows.Close()
	raw := 500.0
	converted := c.Convert(&raw)
	if converted.Liters == nil || *converted.Liters != 100 {
		t.Fatal("incorrect volume", converted)
	}
	if w = request(qp, "POST", payload("empty", 0), admin); w.Code != 409 {
		t.Fatal("approved calibration was mutable", w.Code)
	}
}
