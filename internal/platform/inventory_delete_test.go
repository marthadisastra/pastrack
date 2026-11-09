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

func TestInventoryDeletePreservesAssignmentHistory(t *testing.T) {
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
		result, err := db.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	suffix := time.Now().UnixNano()
	actor := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'QA','unused','bcrypt',1,1,1)", fmt.Sprintf("delete-%d", suffix))
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'QA','NOT_INSTALLED',1)", fmt.Sprintf("QA-%d", suffix))
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'Queclink')", fmt.Sprint(suffix))
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", fmt.Sprint(suffix), model)
	assignment := insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 MINUTE,'qa')", vehicle, device)
	s := New(db)
	remove := func(path, role string, want int) {
		t.Helper()
		w := httptest.NewRecorder()
		s.api(w, httptest.NewRequest("DELETE", "/api/v1/"+path, nil), &Identity{ID: actor, Role: role})
		if w.Code != want {
			t.Fatalf("DELETE %s got %d, want %d: %s", path, w.Code, want, w.Body)
		}
	}
	devicePath := fmt.Sprintf("devices/%d", device)
	remove(devicePath, "operator", 403)
	remove(devicePath, "admin", 200)
	remove(devicePath, "admin", 404)
	var ended bool
	if err := db.QueryRow("SELECT ends_at IS NOT NULL FROM vehicle_assignments WHERE id=?", assignment).Scan(&ended); err != nil || !ended {
		t.Fatalf("assignment history was not closed: %v", err)
	}
	remove(fmt.Sprintf("vehicles/%d", vehicle), "admin", 200)
	var retained int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE actor_id=? AND action='delete'", actor).Scan(&retained); err != nil || retained != 2 {
		t.Fatalf("delete audit entries: %d, %v", retained, err)
	}
}
