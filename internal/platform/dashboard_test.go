package platform

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDashboardProjectionNumbersNullAndQuality(t *testing.T) {
	rows := []map[string]any{
		{"id": int64(11), "vehicle_code": "TRUCK-A", "plate_number": "A", "device_id": int64(41), "device_identifier": "••••1234", "movement_status": "STOPPED", "connectivity": "ONLINE", "speed_kmh": "0.00", "latitude": "0.000000", "fuel_sample_id": int64(91), "fuel_liters": "0.00", "tank_capacity_l": "400.00", "fuel_percent": "0", "fuel_validity": "valid", "fuel_fresh": true, "fuel_calibration_id": int64(7), "gps_fresh": true, "hdop": "0.80", "ignition_status": "UNKNOWN", "physical_ignition_state": "NOT_INSTALLED", "active_alerts": int64(2), "active_alert_codes": "FUEL_DROP,BLE_LOST"},
		{"id": int64(12), "plate_number": "B", "device_id": int64(42), "movement_status": "OFFLINE", "connectivity": "OFFLINE", "fuel_liters": "900", "tank_capacity_l": "1000", "fuel_validity": "valid", "fuel_fresh": false, "hdop": nil},
		{"id": int64(13), "plate_number": "C", "movement_status": "UNASSIGNED", "fuel_liters": nil, "tank_capacity_l": nil},
	}
	vehicles, summary := dashboardProject(rows)
	if vehicles[0].ID != "11" || vehicles[0].Code != "TRUCK-A" || vehicles[0].DeviceID == nil || *vehicles[0].DeviceID != "41" {
		t.Fatal("unstable identity")
	}
	if vehicles[0].GPS.Speed == nil || *vehicles[0].GPS.Speed != 0 || vehicles[0].GPS.Latitude == nil || *vehicles[0].GPS.Latitude != 0 {
		t.Fatal("valid zero lost")
	}
	if vehicles[0].Fuel.SampleID == nil || *vehicles[0].Fuel.SampleID != "91" {
		t.Fatal("fuel sample identity lost")
	}
	if vehicles[2].Fuel.Liters != nil || vehicles[2].GPS.Speed != nil || vehicles[2].Monitored {
		t.Fatal("missing data converted to measurements")
	}
	if vehicles[0].Ignition.Status != "UNKNOWN" || vehicles[0].Ignition.PhysicalState != "NOT_INSTALLED" {
		t.Fatal("ignition invented")
	}
	if summary.Monitored != 2 || summary.States["STOPPED"] != 1 || summary.States["OFFLINE"] != 1 || summary.States["UNASSIGNED"] != 1 || summary.ActiveAlerts != 2 {
		t.Fatal(summary)
	}
	if summary.FuelLiters == nil || *summary.FuelLiters != 0 || summary.FuelValidFresh != 1 || summary.FuelPercent == nil || *summary.FuelPercent != 0 || summary.HDOPSamples != 1 || *summary.AverageHDOP != 0.8 {
		t.Fatal("stale/null data biased aggregation", summary)
	}
	_, empty := dashboardProject(nil)
	if empty.FuelLiters != nil || empty.FuelPercent != nil || empty.AverageHDOP != nil {
		t.Fatal("empty aggregate must be null")
	}
	for _, value := range []any{nil, "", "bad", math.NaN(), math.Inf(1), true} {
		if dashboardNumber(value) != nil {
			t.Fatalf("invalid measurement accepted: %T", value)
		}
	}
	// New raw row with explicitly cleared fuel remains clear rather than inheriting the previous reading.
	rows[0]["fuel_liters"] = nil
	vehicles, _ = dashboardProject(rows)
	if vehicles[0].Fuel.Liters != nil {
		t.Fatal("null clearing failed")
	}
}

func TestDashboardNextFreshnessUsesEarliestMeasurementDeadline(t *testing.T) {
	now := time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	gateway := now.Add(-4 * time.Minute).Format(time.RFC3339Nano)
	gps := now.Add(-3 * time.Minute).Format(time.RFC3339Nano)
	fuel := now.Add(-2 * time.Minute).Format(time.RFC3339Nano)
	invalid := "not-a-time"
	vehicles := []dashboardVehicle{
		{GatewayReceivedAt: &gateway, GPS: dashboardGPS{MeasuredAt: &gps}, Fuel: dashboardFuel{MeasuredAt: &fuel}},
		{SensorReceivedAt: &invalid},
	}
	deadline := dashboardNextFreshness(vehicles, now)
	if want := now.Add(time.Minute); !deadline.Equal(want) {
		t.Fatalf("deadline=%s want=%s", deadline, want)
	}
	old := now.Add(-6 * time.Minute).Format(time.RFC3339Nano)
	if deadline = dashboardNextFreshness([]dashboardVehicle{{GatewayReceivedAt: &old}}, now); !deadline.IsZero() {
		t.Fatalf("expired measurement rescheduled: %s", deadline)
	}
}

func TestDashboardHistoryRangesAreBounded(t *testing.T) {
	for value, expected := range map[string]time.Duration{"12h": 12 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour} {
		if duration, ok := dashboardHistoryDuration(value); !ok || duration != expected {
			t.Fatalf("range %s=%s valid=%v", value, duration, ok)
		}
	}
	for _, value := range []string{"", "1h", "48h", "8d", "12H", "all"} {
		if _, ok := dashboardHistoryDuration(value); ok {
			t.Fatalf("unbounded range accepted: %q", value)
		}
	}
}

func TestDashboardSnapshotDatabaseContract(t *testing.T) {
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
	suffix := fmt.Sprint(time.Now().UnixNano())
	username := "dashboard-" + suffix
	password := "Fixture-Password9!"
	digest, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	result, err := db.Exec("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'Dashboard fixture',?,'bcrypt',1,1,1)", username, digest)
	if err != nil {
		t.Fatal(err)
	}
	uid, _ := result.LastInsertId()
	result, err = db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'Snapshot fixture','NOT_INSTALLED',1)", "DB-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	vehicleID, _ := result.LastInsertId()
	defer func() {
		db.Exec("DELETE FROM sessions WHERE user_id=?", uid)
		db.Exec("DELETE FROM user_roles WHERE user_id=?", uid)
		db.Exec("DELETE FROM users WHERE id=?", uid)
		db.Exec("DELETE FROM vehicles WHERE id=?", vehicleID)
	}()
	raw := token()
	_, err = db.Exec("INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(?,?,1,UTC_TIMESTAMP(3)+INTERVAL 1 HOUR)", uid, hash(raw))
	if err != nil {
		t.Fatal(err)
	}
	app := New(db)
	call := func(method string, cookie bool) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/dashboard/snapshot", strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		if cookie {
			r.AddCookie(&http.Cookie{Name: "pastrack_v2_session", Value: raw})
		}
		app.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", false); w.Code != 401 {
		t.Fatal("anonymous data leak", w.Code)
	}
	if w := call("POST", true); w.Code != 405 {
		t.Fatal("snapshot mutation allowed", w.Code)
	}
	w := call("GET", true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Vehicles []dashboardVehicle `json:"vehicles"`
			Summary  dashboardSummary   `json:"summary"`
		} `json:"data"`
		Meta struct {
			Version  int    `json:"schemaVersion"`
			Mode     string `json:"mode"`
			Realtime bool   `json:"realtime"`
		} `json:"meta"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var expectedTotal int
	if err = db.QueryRow("SELECT COUNT(*) FROM vehicles WHERE deleted_at IS NULL").Scan(&expectedTotal); err != nil {
		t.Fatal(err)
	}
	if response.Meta.Version != 1 || response.Meta.Mode != "snapshot" || response.Meta.Realtime || response.Data.Summary.Total != expectedTotal || len(response.Data.Vehicles) != expectedTotal {
		t.Fatal("envelope/scope mismatch")
	}
	found := false
	for _, v := range response.Data.Vehicles {
		if v.ID == fmt.Sprint(vehicleID) {
			found = true
			if v.Monitored || v.Movement != "UNASSIGNED" || v.Fuel.Liters != nil || v.GPS.Latitude != nil || v.Ignition.Status != "UNKNOWN" {
				t.Fatal("fixture differs from database", v)
			}
		}
	}
	if !found {
		t.Fatal("new vehicle omitted")
	}
	if os.Getenv("RUN_DASHBOARD_BROWSER") == "1" {
		server := httptest.NewServer(app)
		defer server.Close()
		command := exec.Command("node", "../../front/tests/dashboard-smoke.mjs")
		command.Env = append(os.Environ(), "AUTH_BACKEND_URL="+server.URL, "AUTH_TEST_USERNAME="+username, "AUTH_TEST_PASSWORD="+password, "DASHBOARD_TEST_PLATE="+"DB-"+suffix)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("dashboard browser: %v: %s", err, output)
		}
		t.Log(string(output))
	}
}
