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

func TestSensorIdentityFollowsReceivingIMEIAndMAC(t *testing.T) {
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
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'Queclink')", suffix)
	oldGPS := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", "X"+suffix, model)
	gps := insert("INSERT INTO gps_devices(imei,model_id,is_active,protocol_version) VALUES(?,?,1,'80201E0100')", imei, model)
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'QA actual receiver',1)", "QA-"+suffix)
	insert("INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 DAY,'qa')", gps, vehicle)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1)", mac, mac)
	insert("INSERT INTO ble_sensor_bindings(sensor_id,device_id,starts_at,binding_source) VALUES(?,?,UTC_TIMESTAMP(3)-INTERVAL 1 DAY,'qa_legacy')", sensor, oldGPS)
	payload := "+RESP:GTERI,80201E0306," + imei + ",,report$"
	raw := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at,protocol_version) VALUES(?,?,UNHEX(SHA2(?,256)),UTC_TIMESTAMP(3),'80201E0306')", gps, payload, payload)
	insert("INSERT INTO sensor_telemetry(sensor_id,device_id,raw_message_id,accessory_index,received_at,validation_state,decoder_version) VALUES(?,?,?,0,UTC_TIMESTAMP(3),'valid_raw','qa')", sensor, gps, raw)
	s := New(db)
	w := httptest.NewRecorder()
	s.api(w, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/sensors/%d", sensor), nil), &Identity{Role: "admin"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var body struct{ Data map[string]any }
	if err = json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	row := body.Data
	if row["gps_imei"] != imei || row["sensor_mac"] != mac || row["device_protocol_version"] != "80201E0306" || row["vehicle_id"] != float64(vehicle) || row["device_id"] != float64(gps) || row["pairing_active"] != float64(1) {
		t.Fatalf("mapping differs from raw: %#v", row)
	}
}
