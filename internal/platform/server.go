// Package platform serves the isolated v2 application. Existing production entrypoints remain intact.
package platform

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed ui/*
var assets embed.FS

var layouts = sync.OnceValue(func() map[string]json.RawMessage {
	data, err := assets.ReadFile("ui/stitch-pages.json")
	if err != nil {
		panic(err)
	}
	var pages map[string]json.RawMessage
	if err := json.Unmarshal(data, &pages); err != nil {
		panic(err)
	}
	return pages
})

type Server struct {
	PublicAppURL  string
	ResetMailKey  []byte
	SendResetMail func(context.Context, string, string) error
	ReadOnly      bool
	DB            *sql.DB
	mu            sync.Mutex
	attempts      map[string][]time.Time
	subscribers   map[chan struct{}]struct{}
	reportSlots   chan struct{}
	OnNotify      func()
	wsConnections atomic.Int64
	wsHandshakes  atomic.Int64
	wsAuthFailure atomic.Int64
	wsMessages    atomic.Int64
	wsCursor      atomic.Uint64
}
type Identity struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"full_name"`
	Role     string `json:"role"`
	ReadOnly bool   `json:"read_only"`
	Email    string `json:"email"`
}
type key struct{}

func New(db *sql.DB) *Server {
	s := &Server{ReadOnly: os.Getenv("PASTRACK_READ_ONLY") == "true", DB: db, attempts: map[string][]time.Time{}, subscribers: map[chan struct{}]struct{}{}, reportSlots: make(chan struct{}, 2)}
	if db != nil {
		var cursor uint64
		if err := db.QueryRow("SELECT COALESCE(MAX(replay_cursor),0) FROM dashboard_replay_events").Scan(&cursor); err == nil {
			s.wsCursor.Store(cursor)
		}
	}
	return s
}
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func hash(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code string) {
	respond(w, status, map[string]any{"error": map[string]string{"code": code, "message": strings.ReplaceAll(code, "_", " ")}})
}
func body(w http.ResponseWriter, r *http.Request, dst any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	d.DisallowUnknownFields()
	if e := d.Decode(dst); e != nil {
		return e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("one JSON value required")
	}
	return nil
}
func (s *Server) identity(r *http.Request) *Identity {
	u, _ := s.sessionIdentity(r)
	return u
}
func (s *Server) sessionIdentity(r *http.Request) (*Identity, time.Time) {
	c, e := r.Cookie("pastrack_v2_session")
	if e != nil || len(c.Value) != 64 {
		return nil, time.Time{}
	}
	if _, e = hex.DecodeString(c.Value); e != nil {
		return nil, time.Time{}
	}
	var u Identity
	var expires time.Time
	e = s.DB.QueryRowContext(r.Context(), `SELECT u.id,u.username,u.full_name,COALESCE(u.email,''),COALESCE((SELECT r.code FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=u.id ORDER BY FIELD(r.code,'admin','operator','viewer') LIMIT 1),'viewer'),s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>UTC_TIMESTAMP(3) AND s.auth_version=u.auth_version AND u.is_active=1 AND u.deleted_at IS NULL`, hash(c.Value)).Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.Role, &expires)
	if e != nil {
		return nil, time.Time{}
	}
	u.ReadOnly = s.ReadOnly
	return &u, expires.UTC()
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://tile.openstreetmap.org https://server.arcgisonline.com; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if r.URL.Path != "/api/v1/realtime" && r.URL.Path != "/api/v1/ws" {
		r = r.WithContext(ctx)
	}
	if r.URL.Path == "/healthz" {
		if e := s.DB.PingContext(ctx); e != nil {
			fail(w, 503, "database_unavailable")
			return
		}
		respond(w, 200, map[string]any{"status": "ok", "schema": "v2", "physical_ignition": "NOT_INSTALLED", "read_only": s.ReadOnly, "websocket": map[string]any{"connections": s.wsConnections.Load(), "handshakes": s.wsHandshakes.Load(), "authFailures": s.wsAuthFailure.Load(), "messages": s.wsMessages.Load()}})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
		s.authRoute(w, r)
		return
	}
	if r.URL.Path == "/app" || strings.HasPrefix(r.URL.Path, "/app/") {
		s.serveFrontend(w, r)
		return
	}
	if r.URL.Path == "/api/v1/login" {
		s.login(w, r)
		return
	}
	if key, ok := strings.CutPrefix(r.URL.Path, "/assets/layout/"); ok {
		key, valid := strings.CutSuffix(key, ".json")
		page := layouts()[key]
		if !valid || page == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/json")
		http.ServeContent(w, r, key+".json", time.Time{}, bytes.NewReader(page))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		w.Header().Set("Cache-Control", "no-cache")
		sub, _ := fs.Sub(assets, "ui")
		http.StripPrefix("/assets/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
		return
	}
	u := s.identity(r)
	if u == nil {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			fail(w, 401, "authentication_required")
		} else {
			data, _ := assets.ReadFile("ui/login.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		}
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), key{}, u))
	if s.ReadOnly && r.Method != "GET" && r.Method != "HEAD" && r.URL.Path != "/api/v1/logout" {
		fail(w, 403, "live_data_read_only")
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host && r.Header.Get("Origin") != "https://"+r.Host {
			fail(w, 403, "origin_rejected")
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(w, 415, "json_required")
			return
		}
	}
	if s.proxyLive(w, r) {
		return
	}
	switch r.URL.Path {
	case "/api/v1/me":
		respond(w, 200, map[string]any{"data": u})
		return
	case "/api/v1/logout":
		if r.Method != "POST" {
			fail(w, 405, "method_not_allowed")
			return
		}
		c, _ := r.Cookie("pastrack_v2_session")
		if _, e := s.DB.ExecContext(r.Context(), "DELETE FROM sessions WHERE token_hash=?", hash(c.Value)); e != nil {
			fail(w, 503, "database_unavailable")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "pastrack_v2_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		s.Notify()
		respond(w, 200, map[string]any{"data": true})
		return
	case "/api/v1/realtime":
		s.stream(w, r)
		return
	case "/api/v1/ws":
		s.dashboardWebSocket(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		s.api(w, r, u)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	data, _ := assets.ReadFile("ui/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "" {
		r.Header.Set("Content-Type", "application/json")
	}
	s.authLogin(w, r)
}

// Bootstrap creates only an explicit local account, never fake fleet or telemetry data.
func Bootstrap(ctx context.Context, db *sql.DB, username, password string) error {
	if len(password) < 12 || len(password) > 72 || len(username) < 3 {
		return fmt.Errorf("bootstrap requires username and password of 12..72 bytes")
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	for _, role := range []string{"admin", "operator", "viewer"} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO roles(code,name) VALUES(?,?) ON DUPLICATE KEY UPDATE name=VALUES(name)", role, role); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,?,'bcrypt',1,1,1)`, username, username, string(digest))
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	if _, err = tx.ExecContext(ctx, "INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code='admin'", id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Server) Notify() {
	s.mu.Lock()
	for ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
	if s.OnNotify != nil {
		s.OnNotify()
	}
}
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		fail(w, 500, "stream_unavailable")
		return
	}
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.subscribers, ch); s.mu.Unlock() }()
	endpoints := r.URL.Query()["watch"]
	if len(endpoints) > 8 {
		fail(w, 400, "too_many_live_resources")
		return
	}
	if len(endpoints) == 0 {
		endpoints = []string{"fleet/live", "fleet/overview"}
	}
	state := map[string]liveRows{}
	deadline := time.NewTimer(time.Hour)
	deadline.Stop()
	defer deadline.Stop()
	push := func() bool {
		user := s.identity(r)
		if user == nil {
			return false
		}
		patches := map[string]liveSectionPatch{}
		for _, endpoint := range endpoints {
			next, err := s.liveSnapshot(r, user, endpoint)
			if err != nil {
				fmt.Fprint(w, "event: live-error\ndata: {\"error\":\"live_snapshot_failed\"}\n\n")
				flusher.Flush()
				return true
			}
			delta := diffLive(state[endpoint], next)
			state[endpoint] = next
			if len(delta.Upserts) > 0 || len(delta.Removed) > 0 || delta.Order != nil || delta.Page != nil {
				patches[endpoint] = delta
			}
		}
		if len(patches) > 0 {
			b, _ := json.Marshal(patches)
			fmt.Fprintf(w, "event: values\ndata: %s\n\n", b)
			flusher.Flush()
		}
		// A one-shot expiration event updates freshness without polling.
		deadline.Stop()
		var nextDeadline time.Time
		now := time.Now().UTC()
		for _, snapshot := range state {
			for _, row := range snapshot.Rows {
				if at := timestamp(row["next_freshness_change_at"]); at.After(now) && (nextDeadline.IsZero() || at.Before(nextDeadline)) {
					nextDeadline = at
				}
				for _, field := range []string{"gps_time", "device_last_received_at", "fuel_received_at", "fuel_measured_at", "sensor_last_received_at", "last_received_at"} {
					at := timestamp(row[field])
					if at.IsZero() {
						continue
					}
					at = at.Add(5 * time.Minute)
					if at.After(now) && (nextDeadline.IsZero() || at.Before(nextDeadline)) {
						nextDeadline = at
					}
				}
			}
		}
		if !nextDeadline.IsZero() {
			deadline.Reset(time.Until(nextDeadline))
		}
		return true
	}
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, "retry: 3000\nevent: ready\ndata: {}\n\n")
	flusher.Flush()
	if !push() {
		return
	}
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	expiry := time.NewTimer(10 * time.Minute)
	defer expiry.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-expiry.C:
			return
		case <-deadline.C:
			if !push() {
				return
			}
		case <-ticker.C:
			if s.identity(r) == nil {
				return
			}
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-ch:
			if s.identity(r) == nil {
				return
			}
			if !push() {
				return
			}
		}
	}
}

type rowQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryRows(ctx context.Context, db rowQuerier, q string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range values {
			ptr[i] = &values[i]
		}
		if err = rows.Scan(ptr...); err != nil {
			return nil, err
		}
		row := map[string]any{}
		for i, c := range cols {
			switch v := values[i].(type) {
			case []byte:
				row[c] = string(v)
			case time.Time:
				row[c] = v.UTC().Format(time.RFC3339Nano)
			default:
				row[c] = v
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
func databaseError(w http.ResponseWriter, err error) {
	slog.Error("v2 database request failed", "kind", fmt.Sprintf("%T", err))
	fail(w, 503, "database_request_failed")
}
func parseID(s string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n <= 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return n, nil
}
