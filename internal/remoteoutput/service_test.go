package remoteoutput

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDOSProtocol(t *testing.T) {
	frame, e := Frame("test1234", true, "000A")
	if e != nil || frame != "AT+GTDOS=test1234,0,1,1,1,0,0,,,1,0,5,,1,,000A$" {
		t.Fatal(frame, e)
	}
	off, e := Frame("test1234", false, "000B")
	if e != nil || !strings.Contains(off, ",0,1,1,0,0,0,") {
		t.Fatal(off, e)
	}
	if _, e = Frame("unsafe,$", true, "000A"); e == nil {
		t.Fatal("injection accepted")
	}
	r, e := Decode("+ACK:GTDOS,80201E0306,000000000000001,GV30CEU,000A,20261103094921,4E95$")
	if e != nil || r.Serial != "000A" || r.Active != nil {
		t.Fatal(r, e)
	}
	r, e = Decode("+RESP:GTDOS,80201E0306,000000000000001,GV30CEU,1,1,1,0.0,0,99.8,106.0,-6.0,20261103094920,0510,0001,7D06,033CA45D,01,8,20261103094921,4E96$")
	if e != nil || r.Active == nil || !*r.Active {
		t.Fatal(r, e)
	}
	if _, e = Decode("+RESP:GTDOS,80201E0306,000000000000001,GV30CEU,1,2,1,0,0,0,0,0,20261103094920,20261103094921,FFFF$"); e == nil {
		t.Fatal("gradual mode mistaken for boolean")
	}
}
func TestDOSDispatchIdempotencyACKAndOutputAreSeparate(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated test DB required")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated test DB required")
	}
	db, e := sql.Open("mysql", dsn+"&time_zone=%27%2B00%3A00%27")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	migration, e := os.ReadFile("../../database/migrations/005_remote_output.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(migration)); e != nil {
		t.Fatal(e)
	}
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
	imei := suffix[len(suffix)-15:]
	user := insert("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused','bcrypt',1,1,1)", "output-"+suffix, "Isolated output QA")
	_, e = db.Exec("INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='admin'", user)
	if e != nil {
		t.Fatal(e)
	}
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", suffix)
	device := insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, model)
	vehicle := insert("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'Isolated output test','NOT_INSTALLED',1)", "OUT-"+suffix)
	insert("INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,UTC_TIMESTAMP()-INTERVAL 1 MINUTE,'qa')", device, vehicle)
	position := insert("INSERT INTO gps_positions(device_id,vehicle_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,fix_status,validation_state) VALUES(?,?,0,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3),-6,106,0,'valid','validated')", device, vehicle)
	insert("INSERT INTO latest_positions(vehicle_id,position_id,gps_time,received_at) VALUES(?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3))", vehicle, position)
	file := filepath.Join(t.TempDir(), "profiles.json")
	config, _ := json.Marshal([]Profile{{IMEI: imei, Password: "test1234", OutputID: 1, Enabled: true}})
	if e = os.WriteFile(file, config, 0600); e != nil {
		t.Fatal(e)
	}
	s, e := New(db, file)
	if e != nil {
		t.Fatal(e)
	}
	var commandLogs []string
	s.CommandLog = func(imei, serial, frame, state string) { commandLogs = append(commandLogs, frame+" state="+state) }
	server, peer := net.Pipe()
	defer server.Close()
	defer peer.Close()
	unbind := s.Bind(imei, server)
	defer unbind()
	frames := make(chan string, 4)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, e := peer.Read(buf)
			if e != nil {
				return
			}
			frames <- string(buf[:n])
		}
	}()
	key := sha256.Sum256([]byte(suffix))
	payload := fmt.Sprintf(`{"active":true,"actor_id":%d,"request_key":"%x"}`, user, key)
	call := func(method, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, fmt.Sprintf("http://receiver/output?vehicle_id=%d", vehicle), strings.NewReader(body))
		s.ServeHTTP(w, r)
		return w
	}
	w := call("POST", payload)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Data struct {
			ID     int64  `json:"id"`
			Serial string `json:"serial"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	select {
	case frame := <-frames:
		want, _ := Frame("test1234", true, result.Data.Serial)
		if frame != want {
			t.Fatal(frame, want)
		}
	case <-time.After(time.Second):
		t.Fatal("command not sent")
	}
	if len(commandLogs) != 1 || strings.Contains(commandLogs[0], "test1234") || !strings.Contains(commandLogs[0], "AT+GTDOS=[REDACTED],0,1,1,1,0,0,,,1,0,5,,1,,") {
		t.Fatal("missing or unsafe TX log", commandLogs)
	}
	if strings.Contains(w.Body.String(), "test1234") {
		t.Fatal("secret exposed")
	}
	if w = call("POST", payload); w.Code != 200 {
		t.Fatal("idempotent retry", w.Code, w.Body.String())
	}
	select {
	case f := <-frames:
		t.Fatal("duplicate transmission", f)
	default:
	}
	ack := fmt.Sprintf("+ACK:GTDOS,80201E0306,%s,GV30CEU,%s,%s,AAAA$", imei, result.Data.Serial, time.Now().UTC().Format("20060102150405"))
	sum := sha256.Sum256([]byte(ack))
	insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,?,UTC_TIMESTAMP(3))", device, []byte(ack), sum[:])
	if e = s.Record(context.Background(), ack); e != nil {
		t.Fatal(e)
	}
	var state string
	db.QueryRow("SELECT state FROM remote_output_commands WHERE id=?", result.Data.ID).Scan(&state)
	if state != "acknowledged" {
		t.Fatal(state)
	}
	cap, _, e := s.capability(context.Background(), vehicle)
	if e != nil || cap["confirmed_active"] != nil {
		t.Fatal("ACK treated as output proof", cap, e)
	}
	report := fmt.Sprintf("+RESP:GTDOS,80201E0306,%s,GV30CEU,1,1,1,0.0,0,99.8,106.0,-6.0,%s,0510,0001,7D06,033CA45D,01,8,%s,AAAB$", imei, time.Now().UTC().Format("20060102150405"), time.Now().UTC().Format("20060102150405"))
	sum = sha256.Sum256([]byte(report))
	insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,?,UTC_TIMESTAMP(3))", device, []byte(report), sum[:])
	cap, _, e = s.capability(context.Background(), vehicle)
	if e != nil || cap["confirmed_active"] != true {
		t.Fatal(cap, e)
	}
	for _, serial := range []string{"0009", "000A"} {
		key := sha256.Sum256([]byte(suffix + serial))
		insert("INSERT INTO remote_output_commands(device_id,vehicle_id,requested_by,serial,request_key,desired_active,state,expires_at) VALUES(?,?,?,?,?,1,'sent',UTC_TIMESTAMP(3))", device, vehicle, user, serial, fmt.Sprintf("%x", key))
	}
	key2 := sha256.Sum256([]byte(suffix + "2"))
	_, e = db.Exec("UPDATE gps_positions SET speed_kmh=30,fix_status='no_fix',gps_time=UTC_TIMESTAMP()-INTERVAL 1 HOUR WHERE id=?", position)
	if e != nil {
		t.Fatal(e)
	}
	w = call("POST", fmt.Sprintf(`{"active":false,"actor_id":%d,"request_key":"%x"}`, user, key2))
	if w.Code != 202 {
		t.Fatal("explicit new command blocked by GNSS or prior ACK", w.Code, w.Body.String())
	}
	select {
	case frame := <-frames:
		if !strings.Contains(frame, ",0,1,1,0,0,0,") || !strings.HasSuffix(frame, ",000B$") {
			t.Fatal("expected OFF frame", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("OFF not transmitted")
	}
}

func TestDefaultDOSNeedsNoProfileOrUserPassword(t *testing.T) {
	s, e := New(nil, "")
	if e != nil {
		t.Fatal(e)
	}
	p, ok := s.profile("000000000000001")
	if !ok || !p.Enabled || p.OutputID != 1 || p.Password != "gv30ceu" {
		t.Fatal(p, ok)
	}
	frame, e := Frame("", true, "0001")
	if e != nil || !strings.HasPrefix(frame, "AT+GTDOS=gv30ceu,") {
		t.Fatal(frame, e)
	}
}
