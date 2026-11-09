package platform

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRegisterObservedDevice(t *testing.T) {
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
	var actor int64
	if e = db.QueryRow("SELECT id FROM users LIMIT 1").Scan(&actor); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("INSERT INTO gps_device_models(manufacturer,model,protocol_family) SELECT 'Queclink','GV30CEU','queclink' WHERE NOT EXISTS(SELECT 1 FROM gps_device_models WHERE model='GV30CEU' AND protocol_family='queclink')"); e != nil {
		t.Fatal(e)
	}
	imei := strconv.FormatInt(time.Now().UnixMicro(), 10)[1:]
	raw := "+RESP:GTSTT,80201E0306," + imei + ",GV30CEU,report$"
	if len(imei) != 15 {
		t.Fatal("invalid test IMEI")
	}
	result, e := db.Exec("INSERT INTO device_raw_messages(payload,payload_sha256,received_at) VALUES(?,UNHEX(SHA2(?,256)),UTC_TIMESTAMP(3))", raw, raw)
	if e != nil {
		t.Fatal(e)
	}
	receipt, _ := result.LastInsertId()
	s := New(db)
	call := func(method, role, payload string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.deviceDiscovery(w, httptest.NewRequest(method, "/api/v1/devices/discovered", strings.NewReader(payload)), &Identity{ID: actor, Role: role}, "devices/discovered")
		return w
	}
	if w := call("GET", "viewer", ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	list := call("GET", "admin", "")
	if list.Code != 200 || !strings.Contains(list.Body.String(), strconv.FormatInt(receipt, 10)) || strings.Contains(list.Body.String(), imei) {
		t.Fatal("invalid discovery result", list.Code)
	}
	body, _ := json.Marshal(map[string]any{"receipt_id": receipt})
	if w := call("POST", "admin", string(body)); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("POST", "admin", string(body)); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	var version string
	if e = db.QueryRow("SELECT protocol_version FROM gps_devices WHERE imei=?", imei).Scan(&version); e != nil || version != "80201E0306" {
		t.Fatal(version, e)
	}
	if w := call("GET", "admin", ""); strings.Contains(w.Body.String(), strconv.FormatInt(receipt, 10)) {
		t.Fatal("registered device still listed")
	}
}

func TestObservedDeviceAllowsOmittedModelAndAcknowledgements(t *testing.T) {
	const imei = "123456789012345"
	for _, tc := range []struct{ frame, model string }{
		{"+RESP:GTERI,80201E0306," + imei + ",,report$", ""},
		{"+BUFF:GTSTT,80201E0100," + imei + ",,report$", ""},
		{"+ACK:GTHBD,80201E0306," + imei + ",GV30CEU,report$", "GV30CEU"},
	} {
		version, gotIMEI, model := observedDevice(tc.frame)
		if version == "" || gotIMEI != imei || model != tc.model {
			t.Fatalf("unexpected header: %q %q %q", version, gotIMEI, model)
		}
	}
}
