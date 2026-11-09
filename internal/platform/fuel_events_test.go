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

func TestFuelInvestigationValidation(t *testing.T) {
	for _, p := range []fuelInvestigation{{State: "resolved"}, {State: "dismissed", Note: " "}, {State: "unknown"}, {State: "open", Version: -1}, {State: "open", ReceiptNumber: strings.Repeat("x", 101)}} {
		if validFuelInvestigation(p) {
			t.Fatalf("accepted invalid input: %+v", p)
		}
	}
	if !validFuelInvestigation(fuelInvestigation{State: "resolved", Note: "Diperiksa petugas"}) {
		t.Fatal("valid resolution rejected")
	}
}
func TestFuelEventsWorkspaceAndInvestigation(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database not configured")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	insert := func(q string, args ...any) int64 {
		t.Helper()
		r, e := db.Exec(q, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'QA Fuel','unused','bcrypt',1,1,1)", "fuel-qa-"+suffix)
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Fuel QA',1)", "F-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", suffix)
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "009"+suffix[len(suffix)-12:], model)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,model,is_active) VALUES(?,'CFS300B',1)", "F-"+suffix)
	fs := insert("INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'verified')", sensor)
	readings := []int64{}
	for i := 0; i < 2; i++ {
		raw := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,'QA',UNHEX(SHA2(?,256)),'2026-11-04 01:00:00')", device, fmt.Sprint(i)+suffix)
		telemetry := insert("INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,validation_state,decoder_version) VALUES(?,?,?,0,'2026-11-04 01:00:00','valid_raw','qa')", sensor, device, raw)
		readings = append(readings, insert("INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,received_at,fuel_liters,validity) VALUES(?,?,?,DATE_ADD('2026-11-04 01:00:00',INTERVAL ? MINUTE),?,'valid')", fs, telemetry, vehicle, i*10, 100+i*20))
	}
	event := insert("INSERT INTO fuel_events(vehicle_id,fuel_sensor_id,start_reading_id,end_reading_id,event_type,delta_l,event_at,rule_version,confidence) VALUES(?,?,?,?,'refueling',20,'2026-11-04 01:10:00','qa:1','sustained_change')", vehicle, fs, readings[0], readings[1])
	s := New(db)
	admin := &Identity{ID: user, Name: "QA", Role: "admin"}
	request := func(path, method, payload string, u *Identity) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/"+path, strings.NewReader(payload))
		if !s.fuelEventOperations(w, r, u, strings.Split(path, "?")[0]) {
			t.Fatal("not handled")
		}
		return w
	}
	path := fmt.Sprintf("fuel/events/workspace?vehicle_id=%d&from=2026-11-04T00:00:00Z&to=2026-11-05T00:00:00Z", vehicle)
	w := request(path, "GET", "", admin)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Events   []map[string]any `json:"events"`
			Readings []map[string]any `json:"readings"`
		} `json:"data"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &response); e != nil {
		t.Fatal(e)
	}
	if len(response.Data.Events) != 1 || len(response.Data.Readings) != 2 {
		t.Fatal(w.Body.String())
	}
	all := request("fuel/events/workspace?vehicle_id=0&from=2026-11-04T00:00:00Z&to=2026-11-05T00:00:00Z", "GET", "", admin)
	if all.Code != 200 {
		t.Fatal(all.Code, all.Body.String())
	}
	var overview struct {
		Data struct {
			All       bool             `json:"all_vehicles"`
			Events    []map[string]any `json:"events"`
			Readings  []any            `json:"readings"`
			Positions []any            `json:"positions"`
		} `json:"data"`
	}
	if e = json.Unmarshal(all.Body.Bytes(), &overview); e != nil || !overview.Data.All || len(overview.Data.Events) < 1 || len(overview.Data.Readings) != 0 || len(overview.Data.Positions) != 0 {
		t.Fatal("All must return events without mixing vehicle telemetry", all.Body.String(), e)
	}
	target := fmt.Sprintf("fuel/events/%d/investigation", event)
	payload := `{"state":"resolved","note":"<script>alert(1)</script>","receipt_number":"QA-001","station_name":"QA SPBU","receipt_liters":20,"receipt_amount":200000,"row_version":0}`
	if w = request(target, "POST", payload, &Identity{ID: user, Role: "viewer"}); w.Code != 403 {
		t.Fatal("viewer write", w.Code)
	}
	if w = request(target, "POST", payload, admin); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(target, "POST", payload, admin); w.Code != 409 {
		t.Fatal("lost update not rejected", w.Code)
	}
	if w = request(fmt.Sprintf("fuel/events/%d/bap", event), "GET", "", admin); w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("unsafe/incomplete BAP", w.Code, w.Body.String())
	}
	if w = request("fuel/events/workspace?vehicle_id=0", "GET", "", admin); w.Code != 400 {
		t.Fatal("invalid vehicle accepted")
	}
	if w = request(fmt.Sprintf("fuel/events/%d/investigation", event+99999), "POST", payload, admin); w.Code != 404 {
		t.Fatal("missing event", w.Code)
	}
	var audits int
	if e = db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE entity='fuel_events' AND entity_id=? AND actor_id=?", event, user).Scan(&audits); e != nil || audits != 1 {
		t.Fatal("audit missing", audits, e)
	}
}
