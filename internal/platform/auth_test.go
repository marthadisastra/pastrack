package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestAuthRecoveryAndSMTPIntegration(t *testing.T) {
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
	username := "auth-" + suffix
	email := username + "@example.test"
	password := "Initial-Password9!"
	digest, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	result, err := db.Exec("INSERT INTO users(username,email,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,'Auth QA',?,'bcrypt',1,1,1)", username, email, digest)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	defer func() {
		db.Exec("DELETE FROM auth_password_resets WHERE user_id=?", id)
		db.Exec("DELETE FROM sessions WHERE user_id=?", id)
		db.Exec("DELETE FROM user_roles WHERE user_id=?", id)
		db.Exec("DELETE FROM users WHERE id=?", id)
		db.Exec("DELETE FROM auth_smtp_settings WHERE updated_by=?", id)
	}()
	db.Exec("INSERT INTO roles(code,name) VALUES('admin','Admin') ON DUPLICATE KEY UPDATE id=id")
	db.Exec("INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='admin'", id)
	s := New(db)
	s.ResetMailKey = []byte("01234567890123456789012345678901")
	s.PublicAppURL = "https://example.test/app"
	var delivered string
	s.SendResetMail = func(_ context.Context, to, link string) error {
		if to != email {
			t.Error("wrong recipient")
		}
		delivered = link
		return nil
	}
	if os.Getenv("RUN_AUTH_BROWSER") == "1" {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/__test/mail" {
				if err := s.DeliverResetEmails(r.Context()); err != nil {
					t.Error(err)
				}
				json.NewEncoder(w).Encode(map[string]string{"link": delivered})
				return
			}
			s.ServeHTTP(w, r)
		}))
		command := exec.Command("node", "../../front/tests/auth-smoke.mjs")
		command.Env = append(os.Environ(), "AUTH_BACKEND_URL="+server.URL, "AUTH_TEST_EMAIL="+email, "AUTH_TEST_PASSWORD="+password)
		output, err := command.CombinedOutput()
		server.Close()
		if err != nil {
			t.Fatalf("auth browser: %v: %s", err, output)
		}
		t.Log(string(output))
		db.Exec("DELETE FROM auth_smtp_settings WHERE updated_by=?", id)
		db.Exec("DELETE FROM auth_password_resets WHERE user_id=?", id)
		s.attempts = map[string][]time.Time{}
	}

	call := func(method, path string, input any, cookie *http.Cookie) *httptest.ResponseRecorder {
		var body string
		if input != nil {
			raw, _ := json.Marshal(input)
			body = string(raw)
		}
		r := httptest.NewRequest(method, "https://example.test/api/v1/auth/"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://example.test")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	check := func(w *httptest.ResponseRecorder, code int) {
		t.Helper()
		if w.Code != code {
			t.Fatalf("status %d expected %d: %s", w.Code, code, w.Body.String())
		}
	}
	check(call("GET", "me", nil, nil), 401)
	login := call("POST", "login", map[string]any{"email": email, "password": password, "remember": true}, nil)
	check(login, 200)
	cookie := login.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.MaxAge <= 0 || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie flags")
	}
	check(call("GET", "me", nil, cookie), 200)
	bad := httptest.NewRequest("POST", "https://example.test/api/v1/auth/logout", strings.NewReader("{}"))
	bad.Header.Set("Content-Type", "application/json")
	bad.Header.Set("Origin", "https://evil.test")
	bad.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, bad)
	check(w, 403)
	check(call("POST", "forgot-password", map[string]string{"email": email}, nil), 202)
	check(call("POST", "forgot-password", map[string]string{"email": "unknown@example.test"}, nil), 202)
	var encrypted []byte
	var storedHash string
	if err = db.QueryRow("SELECT token_hash,delivery_payload FROM auth_password_resets WHERE user_id=?", id).Scan(&storedHash, &encrypted); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encrypted), email) {
		t.Fatal("plaintext payload")
	}
	if err = s.DeliverResetEmails(context.Background()); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(delivered)
	fragment, _ := url.Parse(u.Fragment)
	token := fragment.Query().Get("token")
	if len(token) != 64 || storedHash != hash(token) {
		t.Fatal("token delivery/hash")
	}
	check(call("POST", "reset-password", map[string]string{"token": token, "password": "weak"}, nil), 400)
	reset := map[string]any{"token": token, "password": "Replacement-Password8!", "terminateSessions": false}
	check(call("POST", "reset-password", reset, nil), 200)
	check(call("POST", "reset-password", reset, nil), 400)
	check(call("GET", "me", nil, cookie), 401)
	check(call("POST", "login", map[string]string{"username": username, "password": password}, nil), 401)
	login = call("POST", "login", map[string]string{"username": username, "password": "Replacement-Password8!"}, nil)
	check(login, 200)
	cookie = login.Result().Cookies()[0]
	if cookie.MaxAge != 0 {
		t.Fatal("session cookie must not persist")
	}
	cfg := map[string]any{"enabled": true, "host": "smtp.example.test", "port": 587, "tlsMode": "starttls", "username": "qa", "password": "smtp-private-secret", "fromEmail": "noreply@example.test", "fromName": "QA", "clearPassword": false}
	check(call("PUT", "smtp", cfg, cookie), 200)
	w = call("GET", "smtp", nil, cookie)
	check(w, 200)
	if strings.Contains(w.Body.String(), "smtp-private-secret") || !strings.Contains(w.Body.String(), `"passwordConfigured":true`) {
		t.Fatal("SMTP secret disclosure")
	}
	db.QueryRow("SELECT encrypted_config FROM auth_smtp_settings WHERE id=1").Scan(&encrypted)
	if strings.Contains(string(encrypted), "smtp-private-secret") {
		t.Fatal("SMTP stored plaintext")
	}
	cfg["password"] = ""
	check(call("PUT", "smtp", cfg, cookie), 200)
	saved, err := s.smtpConfig(context.Background())
	if err != nil || saved.Password != "smtp-private-secret" {
		t.Fatal("blank password did not preserve credential")
	}
	check(call("POST", "logout", map[string]any{}, cookie), 200)
	check(call("GET", "me", nil, cookie), 401)
}

func TestSMTPTransportValidation(t *testing.T) {
	cfg := smtpConfig{Enabled: true, Host: "smtp.example.test", Port: 587, TLSMode: "none", FromEmail: "sender@example.test"}
	if validateSMTP(cfg) == nil {
		t.Fatal("remote plaintext allowed")
	}
	cfg.Host = "127.0.0.1"
	if validateSMTP(cfg) != nil {
		t.Fatal("local relay rejected")
	}
	cfg.Username = "secret"
	if validateSMTP(cfg) == nil {
		t.Fatal("plaintext authentication allowed")
	}
}
