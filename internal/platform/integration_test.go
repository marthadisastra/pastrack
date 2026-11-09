package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLocalAuthenticatedAPIIntegration(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated local integration database not configured")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("test database must be isolated")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	user := "qa-" + suffix
	pass := "test-only-credential-" + suffix
	// Insert an independent test user; existing user records are never modified.
	digest, e := testPassword(pass)
	if e != nil {
		t.Fatal(e)
	}
	result, e := db.Exec("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,?,'bcrypt',1,1,1)", user, "QA API account", digest)
	if e != nil {
		t.Fatal(e)
	}
	id, _ := result.LastInsertId()
	if _, e = db.Exec("INSERT INTO roles(code,name) VALUES('admin','Admin') ON DUPLICATE KEY UPDATE id=id"); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='admin'", id); e != nil {
		t.Fatal(e)
	}
	s := New(db)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, user, pass)))
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("login", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatal("insecure session")
	}
	for path := range resources {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/api/v1/"+path, nil)
		r.AddCookie(cookies[0])
		s.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Errorf("GET %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"fleet/overview", "fleet/live", "vehicles/1/positions", "vehicles/1/devices", "vehicles/1/calibrations"} {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/api/v1/"+path, nil)
		r.AddCookie(cookies[0])
		s.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Errorf("GET %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/v1/vehicles", strings.NewReader(`{"plate_number":"QA-`+suffix+`","vehicle_name":"Test vehicle"}`))
	r.AddCookie(cookies[0])
	r.Header.Set("Content-Type", "application/json")
	s.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Code, w.Body.String())
	}
	var state string
	if e = db.QueryRowContext(context.Background(), "SELECT physical_ignition_state FROM vehicles WHERE plate_number=?", "QA-"+suffix).Scan(&state); e != nil || state != "NOT_INSTALLED" {
		t.Fatal(state, e)
	}
	t.Setenv("LOCAL_REPORT_DIR", t.TempDir())
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/v1/reports", strings.NewReader(`{"report_type":"positions","from":"2026-10-01T00:00:00Z","to":"2026-10-02T00:00:00Z"}`))
	r.AddCookie(cookies[0])
	r.Header.Set("Content-Type", "application/json")
	s.ServeHTTP(w, r)
	if w.Code != 202 {
		t.Fatal("report create", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &created); e != nil {
		t.Fatal(e)
	}
	var jobState string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if e = db.QueryRow("SELECT state FROM report_jobs WHERE id=?", created.Data.ID).Scan(&jobState); e != nil {
			t.Fatal(e)
		}
		if jobState == "completed" || jobState == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if jobState != "completed" {
		t.Fatal("report did not complete", jobState)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/reports/%d/download", created.Data.ID), nil)
	r.AddCookie(cookies[0])
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "latitude") {
		t.Fatal("report download", w.Code)
	}

	s.ReadOnly = true
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/v1/vehicles", strings.NewReader(`{"plate_number":"MUST-NOT-WRITE","vehicle_name":"Blocked"}`))
	r.AddCookie(cookies[0])
	r.Header.Set("Content-Type", "application/json")
	s.ServeHTTP(w, r)
	if w.Code != 403 || !strings.Contains(w.Body.String(), "live_data_read_only") {
		t.Fatal("read-only write guard failed", w.Code)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/me", nil)
	r.AddCookie(cookies[0])
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"read_only":true`) {
		t.Fatal("read-only session identity", w.Code, w.Body.String())
	}
	s.ReadOnly = false

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/gps/positions?limit=999999", nil)
	r.AddCookie(cookies[0])
	s.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("unbounded query accepted")
	}
}
