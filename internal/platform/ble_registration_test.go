package platform

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBLERegistrationAtomicAndExclusive(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database not configured")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("requires isolated test database")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	insert := func(query string, args ...any) int64 {
		t.Helper()
		result, e := db.Exec(query, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, e := result.LastInsertId()
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused-for-handler-test','bcrypt',1,1,1)", "ble-qa-"+suffix, "QA")
	v1 := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'QA BLE A',1)", "A-"+suffix)
	v2 := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'QA BLE B',1)", "B-"+suffix)
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", suffix)
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "005"+suffix[len(suffix)-12:], model)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,model,is_active) VALUES(?,'CFS300B',1)", "QA-"+suffix)
	sum := sha256.Sum256([]byte(suffix))
	raw := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,?,UTC_TIMESTAMP(3))", device, []byte("QA-isolated-fixture$"), sum[:])
	telemetry := insert("INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,validation_state,decoder_version) VALUES(?,?,?,0,UTC_TIMESTAMP(3),'valid_raw','qa')", sensor, device, raw)
	insert("INSERT INTO ble_sensor_status(sensor_id,telemetry_id,last_received_at,status,stale_after_seconds) VALUES(?,?,UTC_TIMESTAMP(3),'connected',300)", sensor, telemetry)
	tank := insert("INSERT INTO fuel_tank_profiles(name,revision,effective_from) VALUES('QA unassigned',1,UTC_TIMESTAMP(3))")
	otherTank := insert("INSERT INTO fuel_tank_profiles(vehicle_id,name,revision,effective_from) VALUES(?,'QA other vehicle',1,UTC_TIMESTAMP(3))", v2)
	s := New(db)
	admin := &Identity{ID: user, Role: "admin"}
	request := func(vehicle int64, method, action, payload string, identity *Identity) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/vehicles/"+fmt.Sprint(vehicle)+"/"+action, strings.NewReader(payload))
		s.bleRegistration(w, r, identity, "vehicles/"+fmt.Sprint(vehicle)+"/"+action)
		return w
	}
	w := request(v1, "GET", "ble-options", "", admin)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var options struct {
		Data struct {
			Sensors []struct {
				ID int64 `json:"id"`
			} `json:"sensors"`
			Tanks []struct {
				ID int64 `json:"id"`
			} `json:"tank_profiles"`
		} `json:"data"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range options.Data.Sensors {
		if item.ID == sensor {
			found = true
		}
	}
	if !found {
		t.Fatal("observed available sensor missing")
	}
	for _, item := range options.Data.Tanks {
		if item.ID == otherTank {
			t.Fatal("foreign tank offered")
		}
	}
	body := fmt.Sprintf(`{"sensor_id":%d,"tank_profile_id":%d}`, sensor, tank)
	codes := make([]int, 2)
	var group sync.WaitGroup
	for i, v := range []int64{v1, v2} {
		group.Add(1)
		go func(i int, vehicle int64) {
			defer group.Done()
			codes[i] = request(vehicle, "POST", "ble-registration", body, admin).Code
		}(i, v)
	}
	group.Wait()
	if !((codes[0] == 200 && codes[1] == 409) || (codes[1] == 200 && codes[0] == 409)) {
		t.Fatal("exclusive registration failed", codes)
	}
	winner, loser := v1, v2
	if codes[1] == 200 {
		winner, loser = v2, v1
	}
	w = request(winner, "POST", "ble-registration", body, admin)
	if w.Code != 200 {
		t.Fatal("idempotent registration", w.Code, w.Body.String())
	}
	var count int
	var owner int64
	db.QueryRow("SELECT COUNT(*) FROM vehicle_assignments WHERE device_id=?", device).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate GPS assignment", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM ble_sensor_bindings WHERE sensor_id=? AND ends_at IS NULL", sensor).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate sensor binding", count)
	}
	db.QueryRow("SELECT vehicle_id FROM fuel_tank_profiles WHERE id=?", tank).Scan(&owner)
	if owner != winner {
		t.Fatal("tank owner mismatch")
	}
	db.QueryRow("SELECT COUNT(*) FROM vehicle_assignments WHERE device_id=? AND vehicle_id=?", device, loser).Scan(&count)
	if count != 0 {
		t.Fatal("loser transaction leaked assignment")
	}
	w = request(loser, "GET", "ble-options", "", admin)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if err = json.Unmarshal(w.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	for _, item := range options.Data.Sensors {
		if item.ID == sensor {
			t.Fatal("occupied sensor still offered")
		}
	}
	w = request(winner, "POST", "ble-registration", body, &Identity{ID: user, Role: "viewer"})
	if w.Code != 403 {
		t.Fatal("viewer mutation allowed")
	}
	var fuelTank int64
	if err = db.QueryRow("SELECT tank_profile_id FROM fuel_sensors WHERE sensor_id=?", sensor).Scan(&fuelTank); err != nil || fuelTank != tank {
		t.Fatal("fuel pipeline not mapped", fuelTank, err)
	}
	var ignition string
	db.QueryRow("SELECT physical_ignition_state FROM vehicles WHERE id=?", winner).Scan(&ignition)
	if ignition != "NOT_INSTALLED" {
		t.Fatal("ignition changed")
	}
}
