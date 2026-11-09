package platform

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDeviceTransferMoveAndSwap(t *testing.T) {
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
	insert := func(q string, a ...any) int64 {
		t.Helper()
		r, e := db.Exec(q, a...)
		if e != nil {
			t.Fatal(e)
		}
		id, e := r.LastInsertId()
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'QA','unused','bcrypt',1,1,1)", "transfer-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'Queclink')", suffix)
	makeGPS := func(prefix string) int64 {
		return insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", prefix+suffix, model)
	}
	makeVehicle := func(prefix string) int64 {
		return insert("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'QA','NOT_INSTALLED',1)", prefix+suffix)
	}
	a, b := makeGPS("1"), makeGPS("2")
	one, two, three := makeVehicle("A"), makeVehicle("B"), makeVehicle("C")
	assignment := insert("INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 MINUTE,'qa')", a, one)
	insert("INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 MINUTE,'qa')", b, two)
	s := New(db)
	apply := func(device, dest, from, other int64, want int) {
		t.Helper()
		path := fmt.Sprintf("/api/v1/devices/%d/connect-vehicle", device)
		body := fmt.Sprintf(`{"vehicle_id":%d,"transfer":{"from_vehicle_id":%d,"target_device_id":%d}}`, dest, from, other)
		w := httptest.NewRecorder()
		s.api(w, httptest.NewRequest("POST", path, strings.NewReader(body)), &Identity{ID: user, Role: "admin"})
		if w.Code != want {
			t.Fatalf("status %d, want %d: %s", w.Code, want, w.Body)
		}
	}
	check := func(device, want int64) {
		t.Helper()
		var got int64
		if e := db.QueryRow("SELECT vehicle_id FROM vehicle_assignments WHERE device_id=? AND ends_at IS NULL", device).Scan(&got); e != nil || got != want {
			t.Fatalf("active assignment %d, want %d: %v", got, want, e)
		}
	}
	apply(a, two, one, 0, 409)
	apply(a, two, one, b, 200)
	check(a, two)
	check(b, one)
	apply(a, two, one, b, 409)
	apply(b, three, one, 0, 200)
	check(a, two)
	check(b, three)
	var oldVehicle int64
	var ended bool
	if e := db.QueryRow("SELECT vehicle_id,ends_at IS NOT NULL FROM vehicle_assignments WHERE id=?", assignment).Scan(&oldVehicle, &ended); e != nil || oldVehicle != one || !ended {
		t.Fatalf("history changed: %d %v %v", oldVehicle, ended, e)
	}
}
