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

func TestVehicleRegistryFilteringPaginationAndValidation(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated DB required")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated DB required")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	prefix := "INV-" + suffix
	insert := func(q string, args ...any) int64 {
		t.Helper()
		r, e := db.Exec(q, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return id
	}
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused','bcrypt',1,1,1)", prefix, "QA inventory")
	ids := []int64{}
	for i := 0; i < 4; i++ {
		ids = append(ids, insert("INSERT INTO vehicles(plate_number,vehicle_name,fleet_code,physical_ignition_state,row_version) VALUES(?,?,?,'NOT_INSTALLED',1)", fmt.Sprintf("%s-%d", prefix, i), "Needle "+prefix, "Fleet "+prefix))
	}
	app := New(db)
	admin := &Identity{ID: user, Role: "admin"}
	request := func(path, method, payload string, who *Identity) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/"+path, strings.NewReader(payload))
		app.api(w, r, who)
		return w
	}
	decode := func(w *httptest.ResponseRecorder) struct {
		Data []map[string]any `json:"data"`
		Page struct {
			More   bool   `json:"has_more"`
			Cursor string `json:"next_cursor"`
		} `json:"page"`
	} {
		t.Helper()
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var out struct {
			Data []map[string]any `json:"data"`
			Page struct {
				More   bool   `json:"has_more"`
				Cursor string `json:"next_cursor"`
			} `json:"page"`
		}
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	path := "vehicles?search=" + prefix + "&movement_status=UNASSIGNED&limit=2"
	first := decode(request(path, "GET", "", admin))
	if len(first.Data) != 2 || !first.Page.More {
		t.Fatal(first)
	}
	second := decode(request(path+"&cursor="+first.Page.Cursor, "GET", "", admin))
	if len(second.Data) != 2 || second.Page.More || first.Data[0]["id"] == second.Data[0]["id"] {
		t.Fatal(second)
	}
	if out := decode(request("vehicles?search=Needle%20"+prefix, "GET", "", admin)); len(out.Data) != 4 {
		t.Fatal("name search failed", out)
	}
	if out := decode(request("vehicles?search="+prefix+"&movement_status=MOVING", "GET", "", admin)); len(out.Data) != 0 {
		t.Fatal("status filter ignored", out)
	}
	if w := request("vehicles?movement_status=INVALID", "GET", "", admin); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := request("vehicles", "POST", `{"plate_number":"   ","vehicle_name":"QA"}`, admin); w.Code != 400 {
		t.Fatal("blank plate accepted", w.Code, w.Body.String())
	}
	if w := request("vehicles", "POST", `{"plate_number":"QA","vehicle_name":"QA"}`, &Identity{ID: user, Role: "viewer"}); w.Code != 403 {
		t.Fatal("viewer changed inventory", w.Code)
	}
	payload := fmt.Sprintf(`{"plate_number":%q,"vehicle_name":"Updated","fleet_code":"Updated code"}`, prefix+"-0")
	if w := request(fmt.Sprintf("vehicles/%d", ids[0]), "PUT", payload, admin); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}

	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA','GV30CEU','queclink') ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)")
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", suffix[len(suffix)-15:], model)
	movingVehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,fleet_code,physical_ignition_state,row_version) VALUES(?,'Moving QA',?,'NOT_INSTALLED',1)", prefix+"-M", prefix)
	assignment := insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 MINUTE,'qa_inventory')", movingVehicle, device)
	insert("INSERT INTO gps_device_status(device_id,last_received_at,connectivity) VALUES(?,UTC_TIMESTAMP(3),'online')", device)
	position := insert("INSERT INTO gps_positions(device_id,vehicle_id,assignment_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,fix_status,validation_state) VALUES(?,?,?,0,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3),-6.1,106.8,20,'valid','validated')", device, movingVehicle, assignment)
	insert("INSERT INTO latest_positions(vehicle_id,position_id,gps_time,received_at) VALUES(?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3))", movingVehicle, position)
	moving := decode(request("vehicles?search="+prefix+"&movement_status=MOVING", "GET", "", admin))
	if len(moving.Data) != 1 || moving.Data[0]["device_model"] != "GV30CEU" || moving.Data[0]["movement_status"] != "MOVING" {
		t.Fatal("moving telemetry filter failed", moving)
	}
	var name, ignition string
	var version int
	db.QueryRow("SELECT vehicle_name,physical_ignition_state,row_version FROM vehicles WHERE id=?", ids[0]).Scan(&name, &ignition, &version)
	if name != "Updated" || ignition != "NOT_INSTALLED" || version != 2 {
		t.Fatal(name, ignition, version)
	}
}
