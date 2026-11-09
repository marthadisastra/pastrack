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

func TestHistoryJourneys(t *testing.T) {
	rows := []map[string]any{}
	start := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		speed := 30.0
		lat := -6.0 - float64(i)*.001
		if i >= 5 {
			speed = 0
			lat = -6.005
		}
		rows = append(rows, map[string]any{"gps_time": start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339), "latitude": lat, "longitude": 107.0, "speed_kmh": speed})
	}
	journeys := historyJourneys(rows)
	if len(journeys) != 1 || journeys[0]["end_reason"] != "stopped" {
		t.Fatal(journeys)
	}
	rows = rows[:5]
	journeys = historyJourneys(rows)
	if len(journeys) != 1 || journeys[0]["end_reason"] != "period_end" {
		t.Fatal(journeys)
	}
	// A large gap cannot add an invented connecting distance.
	rows = append(rows, map[string]any{"gps_time": start.Add(time.Hour).Format(time.RFC3339), "latitude": -6.5, "longitude": 107.0, "speed_kmh": 30.0})
	journeys = historyJourneys(rows)
	if len(journeys) != 1 || journeys[0]["end_reason"] != "data_gap" {
		t.Fatal(journeys)
	}
	if journeys[0]["distance_m"].(float64) > 1000 {
		t.Fatal("distance bridged gap", journeys)
	}
}
func TestTripHistoryValidation(t *testing.T) {
	s := New(nil)
	for _, tc := range []struct {
		method, path string
		code         int
	}{{"POST", "/api/v1/trips/history", 405}, {"GET", "/api/v1/trips/history?vehicle_id=0", 400}, {"GET", "/api/v1/trips/history?vehicle_id=1&from=2026-10-01T00:00:00Z&to=2026-11-03T00:00:00Z", 400}} {
		w := httptest.NewRecorder()
		s.tripHistory(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code {
			t.Fatal(tc, w.Code)
		}
	}
}
func TestTripSimulationHasJourneys(t *testing.T) {
	for _, name := range []string{"normal", "anomaly"} {
		data := fuelSimulation(name)
		journeys, ok := data["trips"].([]map[string]any)
		if !ok || len(journeys) != 2 {
			t.Fatal(name, "missing two moving segments", data["trips"])
		}
	}
}

func TestTripHistoryDatabase(t *testing.T) {
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
	insert := func(q string, args ...any) int64 {
		t.Helper()
		r, e := db.Exec(q, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Trip QA',1)", "TRIP-QA-"+suffix)
	other := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Trip Other QA',1)", "TRIP-OTHER-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", suffix)
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "008"+suffix[len(suffix)-12:], model)
	secondary := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "007"+suffix[len(suffix)-12:], model)
	for i := 0; i < 11; i++ {
		insert("INSERT INTO gps_positions(vehicle_id,device_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,fix_status,validation_state) VALUES(?,?,0,DATE_ADD('2026-10-31 01:00:00',INTERVAL ? MINUTE),'2026-10-31 01:00:00',?,107,30,'valid','validated')", vehicle, device, i, -6-float64(i)*.001)
	}
	insert("INSERT INTO gps_positions(vehicle_id,device_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,fix_status,validation_state) VALUES(?,?,0,'2026-10-31 01:00:00','2026-10-31 01:00:00',-6,107,30,'valid','validated')", vehicle, secondary)
	s := New(db)
	get := func(id int64) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.tripHistory(w, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/trips/history?vehicle_id=%d&from=2026-10-30T17:00:00Z&to=2026-10-31T16:59:59Z", id), nil))
		return w
	}
	w := get(vehicle)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Total   int              `json:"total"`
			Trips   []map[string]any `json:"trips"`
			Devices int              `json:"devices_in_period"`
			Vehicle map[string]any   `json:"vehicle"`
		} `json:"data"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &response); e != nil {
		t.Fatal(e)
	}
	if response.Data.Total != 11 || len(response.Data.Trips) != 1 || response.Data.Devices != 2 || response.Data.Vehicle["latest_gps_at"] == nil {
		t.Fatal(w.Body.String())
	}
	w = get(other)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Data.Total != 0 || len(response.Data.Trips) != 0 {
		t.Fatal("vehicle filter leaked", w.Body.String())
	}
}
