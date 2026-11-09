package platform

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestDashboardWebSocketCommitRollbackRemovalAndRevocation(t *testing.T) {
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
	result, err := db.Exec("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused','bcrypt',1,1,1)", "ws-"+suffix, "WebSocket QA")
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := result.LastInsertId()
	defer db.Exec("DELETE FROM users WHERE id=?", userID)
	if _, err = db.Exec("INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='viewer'", userID); err != nil {
		t.Fatal(err)
	}
	raw := token()
	if _, err = db.Exec("INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(?,?,1,UTC_TIMESTAMP(3)+INTERVAL 1 HOUR)", userID, hash(raw)); err != nil {
		t.Fatal(err)
	}
	plate := "WS-" + suffix
	result, err = db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'Before','NOT_INSTALLED',1)", plate)
	if err != nil {
		t.Fatal(err)
	}
	vehicleID, _ := result.LastInsertId()
	defer db.Exec("DELETE FROM vehicles WHERE id=?", vehicleID)

	app := New(db)
	server := httptest.NewServer(app)
	defer server.Close()
	config, err := websocket.NewConfig("ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	config.Header.Set("Cookie", "pastrack_v2_session="+raw)
	connection, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	var snapshot dashboardSocketEnvelope
	if err = websocket.JSON.Receive(connection, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Type != "snapshot" || snapshot.Cursor != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	notificationResult, err := db.Exec("INSERT INTO notifications(user_id,title,body,target) VALUES(?,'Realtime inbox','Per-user notification','/alerts')", userID)
	if err != nil {
		t.Fatal(err)
	}
	notificationID, _ := notificationResult.LastInsertId()
	defer db.Exec("DELETE FROM notifications WHERE id=?", notificationID)
	app.Notify()
	var notificationDelta dashboardSocketEnvelope
	if err = websocket.JSON.Receive(connection, &notificationDelta); err != nil {
		t.Fatal(err)
	}
	if notificationDelta.Cursor != 2 || len(notificationDelta.NotificationUpserts) != 1 || notificationDelta.NotificationUpserts[0].ID != fmt.Sprint(notificationID) || notificationDelta.NotificationUpserts[0].ReadAt != nil {
		t.Fatalf("unexpected notification delta: %+v", notificationDelta)
	}
	request, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/notifications/%d/read", server.URL, notificationID), strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "pastrack_v2_session", Value: raw})
	response, requestErr := http.DefaultClient.Do(request)
	if requestErr != nil {
		t.Fatal(requestErr)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("mark read status=%d", response.StatusCode)
	}
	if err = websocket.JSON.Receive(connection, &notificationDelta); err != nil {
		t.Fatal(err)
	}
	if notificationDelta.Cursor != 3 || len(notificationDelta.NotificationUpserts) != 1 || notificationDelta.NotificationUpserts[0].ReadAt == nil {
		t.Fatalf("read state did not synchronize: %+v", notificationDelta)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("UPDATE vehicles SET vehicle_name='Rolled back' WHERE id=?", vehicleID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	app.Notify()
	message := make(chan dashboardSocketEnvelope, 1)
	receiveError := make(chan error, 1)
	go func() {
		var envelope dashboardSocketEnvelope
		if receiveErr := websocket.JSON.Receive(connection, &envelope); receiveErr != nil {
			receiveError <- receiveErr
			return
		}
		message <- envelope
	}()
	select {
	case envelope := <-message:
		t.Fatalf("rollback emitted a frame: %+v", envelope)
	case receiveErr := <-receiveError:
		t.Fatal(receiveErr)
	case <-time.After(250 * time.Millisecond):
	}

	if _, err = db.Exec("UPDATE vehicles SET vehicle_name='Committed' WHERE id=?", vehicleID); err != nil {
		t.Fatal(err)
	}
	app.Notify()
	var delta dashboardSocketEnvelope
	select {
	case delta = <-message:
	case receiveErr := <-receiveError:
		t.Fatal(receiveErr)
	case <-time.After(3 * time.Second):
		t.Fatal("committed update did not emit a delta")
	}
	if delta.Type != "delta" || delta.Cursor != 4 || len(delta.Upserts) != 1 || delta.Upserts[0].ID != fmt.Sprint(vehicleID) || delta.Upserts[0].Name != "Committed" {
		t.Fatalf("unexpected update delta: %+v", delta)
	}

	if _, err = db.Exec("UPDATE vehicles SET deleted_at=UTC_TIMESTAMP(3) WHERE id=?", vehicleID); err != nil {
		t.Fatal(err)
	}
	app.Notify()
	if err = websocket.JSON.Receive(connection, &delta); err != nil {
		t.Fatal(err)
	}
	if delta.Type != "delta" || delta.Cursor != 5 || len(delta.Removed) != 1 || delta.Removed[0] != fmt.Sprint(vehicleID) {
		t.Fatalf("unexpected removal delta: %+v", delta)
	}

	if _, err = db.Exec("UPDATE users SET auth_version=auth_version+1 WHERE id=?", userID); err != nil {
		t.Fatal(err)
	}
	app.Notify()
	var revoked dashboardSocketEnvelope
	if err = websocket.JSON.Receive(connection, &revoked); err != nil {
		t.Fatal(err)
	}
	if revoked.Type != "session_revoked" {
		t.Fatalf("session stayed active: %+v", revoked)
	}
}

func TestDashboardWebSocketRejectsCrossOrigin(t *testing.T) {
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
	server := httptest.NewServer(New(db))
	defer server.Close()
	config, err := websocket.NewConfig("ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", "https://attacker.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if connection, dialErr := websocket.DialConfig(config); dialErr == nil {
		connection.Close()
		t.Fatal("cross-origin WebSocket was accepted")
	}
}

func TestDashboardFuelSamplesAreIdempotentAndVehicleScoped(t *testing.T) {
	oldID, newID := "91", "92"
	at := "2026-11-09T01:00:00Z"
	liters := 210.5
	before := []dashboardVehicle{{ID: "11", Fuel: dashboardFuel{SampleID: &oldID}}}
	after := []dashboardVehicle{{ID: "11", Fuel: dashboardFuel{SampleID: &newID, MeasuredAt: &at, Liters: &liters, Validity: "valid"}}}
	samples := dashboardFuelSamples(before, after)
	if len(samples) != 1 || samples[0].VehicleID != "11" || samples[0].SampleID != "92" || samples[0].At != at || samples[0].Liters == nil || *samples[0].Liters != liters {
		t.Fatalf("unexpected fuel sample delta: %+v", samples)
	}
	if duplicate := dashboardFuelSamples(after, after); len(duplicate) != 0 {
		t.Fatalf("duplicate fuel sample emitted: %+v", duplicate)
	}
}

func TestDashboardStreamCommitAndSessionRevocation(t *testing.T) {
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
	suffix := fmt.Sprint(time.Now().UnixNano())
	result, e := db.Exec("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'unused','bcrypt',1,1,1)", "stream-"+suffix, "Isolated stream QA")
	if e != nil {
		t.Fatal(e)
	}
	user, _ := result.LastInsertId()
	_, e = db.Exec("INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='viewer'", user)
	if e != nil {
		t.Fatal(e)
	}
	raw := token()
	_, e = db.Exec("INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(?,?,1,UTC_TIMESTAMP(3)+INTERVAL 1 HOUR)", user, hash(raw))
	if e != nil {
		t.Fatal(e)
	}
	result, e = db.Exec("INSERT INTO vehicles(plate_number,vehicle_name,physical_ignition_state,row_version) VALUES(?,'Before','NOT_INSTALLED',1)", "ST-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	vehicle, _ := result.LastInsertId()
	app := New(db)
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	source := "vehicles?search=ST-" + suffix
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/realtime?watch="+url.QueryEscape(source), nil)
	req.AddCookie(&http.Cookie{Name: "pastrack_v2_session", Value: raw})
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.StatusCode)
	}
	reader := bufio.NewScanner(response.Body)
	readPatch := func() map[string]liveSectionPatch {
		t.Helper()
		for reader.Scan() {
			line := reader.Text()
			if strings.HasPrefix(line, "data: ") && strings.Contains(line, "upserts") {
				var v map[string]liveSectionPatch
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &v); err != nil {
					t.Fatal(err)
				}
				return v
			}
		}
		t.Fatal("no patch", reader.Err())
		return nil
	}
	first := readPatch()
	id := fmt.Sprint(vehicle)
	if first[source].Upserts[id]["vehicle_name"] != "Before" {
		t.Fatal(first)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec("UPDATE vehicles SET vehicle_name='After' WHERE id=?", vehicle); e != nil {
		tx.Rollback()
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	app.Notify()
	patch := readPatch()
	if patch[source].Upserts[id]["vehicle_name"] != "After" {
		t.Fatal(patch)
	}
	if _, present := patch[source].Upserts[id]["plate_number"]; present {
		t.Fatal("unchanged field retransmitted")
	}
	if _, e = db.Exec("UPDATE users SET auth_version=auth_version+1 WHERE id=?", user); e != nil {
		t.Fatal(e)
	}
	app.Notify()
	for reader.Scan() {
	}
	if reader.Err() != nil {
		t.Fatal("stream did not close on revocation", reader.Err())
	}
}

func TestDashboardReadAccessForAllRoles(t *testing.T) {
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
	app := New(db)
	for _, role := range []string{"admin", "operator", "viewer"} {
		for _, path := range []string{"fleet/live", "fleet/overview", "fleet/activity"} {
			t.Run(role+"/"+path, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/api/v1/"+path, nil)
				app.api(w, r, &Identity{Role: role})
				if w.Code != 200 {
					t.Fatal(w.Code, w.Body.String())
				}
			})
		}
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/fleet/live", nil))
	if w.Code != 401 {
		t.Fatal("anonymous dashboard data exposed", w.Code)
	}
}
