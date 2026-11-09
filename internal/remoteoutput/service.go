package remoteoutput

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Profile struct {
	IMEI     string `json:"imei"`
	Password string `json:"password"`
	OutputID int    `json:"output_id"`
	Enabled  bool   `json:"enabled"`
}
type connection struct {
	conn net.Conn
	mu   sync.Mutex
}
type Service struct {
	DB          *sql.DB
	profiles    map[string]Profile
	mu          sync.Mutex
	connections map[string]*connection
	Notify      func()
	CommandLog  func(imei, serial, redactedFrame, state string)
}

func New(db *sql.DB, path string) (*Service, error) {
	s := &Service{DB: db, profiles: map[string]Profile{}, connections: map[string]*connection{}}
	if path == "" {
		s.profiles = nil
		return s, nil
	}
	info, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("remote profile file must be private")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var profiles []Profile
	if e = json.Unmarshal(data, &profiles); e != nil {
		return nil, e
	}
	for _, p := range profiles {
		if !regexp.MustCompile(`^[0-9]{15}$`).MatchString(p.IMEI) || !passwordPattern.MatchString(p.Password) || p.OutputID != 1 {
			return nil, fmt.Errorf("invalid remote output profile")
		}
		s.profiles[p.IMEI] = p
	}
	return s, nil
}

// Without an override file, use the manufacturer-documented default for output1.
// Supplying a profile file retains its explicit allowlist behavior.
func (s *Service) profile(imei string) (Profile, bool) {
	if s.profiles == nil {
		return Profile{IMEI: imei, Password: DefaultPassword, OutputID: 1, Enabled: true}, true
	}
	p, ok := s.profiles[imei]
	return p, ok
}
func (s *Service) Bind(imei string, c net.Conn) func() {
	p, ok := s.profile(imei)
	if !ok || !p.Enabled {
		return func() {}
	}
	entry := &connection{conn: c}
	s.mu.Lock()
	s.connections[imei] = entry
	s.mu.Unlock()
	if s.Notify != nil {
		s.Notify()
	}
	return func() {
		s.mu.Lock()
		if s.connections[imei] == entry {
			delete(s.connections, imei)
		}
		s.mu.Unlock()
		if s.Notify != nil {
			s.Notify()
		}
	}
}
func (s *Service) capability(ctx context.Context, vehicle int64) (map[string]any, string, error) {
	var device int64
	var imei string
	e := s.DB.QueryRowContext(ctx, `SELECT d.id,d.imei FROM vehicle_assignments va JOIN gps_devices d ON d.id=va.device_id AND d.is_active=1 AND d.deleted_at IS NULL JOIN vehicles v ON v.id=va.vehicle_id AND v.deleted_at IS NULL WHERE va.vehicle_id=? AND va.starts_at<=UTC_TIMESTAMP(3) AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3))`, vehicle).Scan(&device, &imei)
	if e != nil {
		return map[string]any{"configured": false, "connected": false, "reason": "GPS belum terhubung ke kendaraan"}, "", nil
	}
	p, ok := s.profile(imei)
	s.mu.Lock()
	connected := s.connections[imei] != nil
	s.mu.Unlock()
	out := map[string]any{"device_id": device, "vehicle_id": vehicle, "configured": ok && p.Enabled, "connected": connected, "output_id": 1, "confirmed_active": nil, "reason": "Konfigurasi output dan kredensial belum tersedia"}
	if ok && p.Enabled {
		out["reason"] = ""
		if !connected {
			out["reason"] = "GPS belum memiliki koneksi TCP aktif"
		}
	}
	var payload string
	e = s.DB.QueryRowContext(ctx, `SELECT CONVERT(payload USING utf8mb4) FROM device_raw_messages WHERE device_id=? AND (payload LIKE '+RESP:GTDOS,%' OR payload LIKE '+BUFF:GTDOS,%') ORDER BY SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(payload USING utf8mb4),',',-2),',',1) DESC,id DESC LIMIT 1`, device).Scan(&payload)
	if e != nil && e != sql.ErrNoRows {
		return nil, "", e
	}
	if reply, e := Decode(payload); e == nil && reply.Active != nil {
		out["confirmed_active"] = *reply.Active
		out["output_reported_at"] = reply.At
	}
	var lastID int64
	var lastState string
	var desired bool
	var expires, created time.Time
	e = s.DB.QueryRowContext(ctx, "SELECT id,state,desired_active,expires_at,created_at FROM remote_output_commands WHERE device_id=? ORDER BY id DESC LIMIT 1", device).Scan(&lastID, &lastState, &desired, &expires, &created)
	if e != nil && e != sql.ErrNoRows {
		return nil, "", e
	}
	if e == nil {
		if time.Now().UTC().After(expires) && (lastState == "sent" || lastState == "dispatching" || lastState == "uncertain") {
			lastState = "timeout"
		}
		out["last_command"] = map[string]any{"id": lastID, "state": lastState, "desired_active": desired, "expires_at": expires, "created_at": created}
	}
	return out, imei, nil
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	vehicle, e := strconv.ParseInt(r.URL.Query().Get("vehicle_id"), 10, 64)
	if e != nil || vehicle <= 0 {
		respond(w, 400, map[string]any{"error": "invalid_vehicle"})
		return
	}
	cap, imei, e := s.capability(ctx, vehicle)
	if e != nil {
		respond(w, 503, map[string]any{"error": "database_unavailable"})
		return
	}
	if r.Method == "GET" {
		respond(w, 200, map[string]any{"data": cap})
		return
	}
	if r.Method != "POST" {
		respond(w, 405, map[string]any{"error": "method_not_allowed"})
		return
	}
	var in struct {
		Active *bool  `json:"active"`
		Actor  int64  `json:"actor_id"`
		Key    string `json:"request_key"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2048))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil || in.Active == nil || in.Actor <= 0 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(in.Key) {
		respond(w, 400, map[string]any{"error": "invalid_request"})
		return
	}
	if cap["configured"] != true || cap["connected"] != true {
		respond(w, 409, map[string]any{"error": "Output belum dikonfigurasi atau GPS offline"})
		return
	}
	profile, _ := s.profile(imei)
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		respond(w, 503, map[string]any{"error": "database_unavailable"})
		return
	}
	defer tx.Rollback()
	var device int64
	e = tx.QueryRowContext(ctx, "SELECT id FROM gps_devices WHERE id=? AND is_active=1 FOR UPDATE", cap["device_id"]).Scan(&device)
	if e != nil {
		respond(w, 409, map[string]any{"error": "device_unavailable"})
		return
	}
	// Recheck current ownership before sending the requested output command.
	var matches int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicle_assignments va JOIN vehicles v ON v.id=va.vehicle_id AND v.deleted_at IS NULL WHERE va.device_id=? AND va.vehicle_id=? AND va.starts_at<=UTC_TIMESTAMP(3) AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3))`, device, vehicle).Scan(&matches)
	if e != nil || matches != 1 {
		respond(w, 409, map[string]any{"error": "vehicle_assignment_changed"})
		return
	}
	var roleCount int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles rr ON rr.id=ur.role_id WHERE u.id=? AND u.is_active=1 AND u.deleted_at IS NULL AND rr.code='admin'`, in.Actor).Scan(&roleCount)
	if e != nil || roleCount == 0 {
		respond(w, 403, map[string]any{"error": "admin_required"})
		return
	}
	var existing int64
	var state string
	e = tx.QueryRowContext(ctx, "SELECT id,state FROM remote_output_commands WHERE request_key=? AND requested_by=? AND vehicle_id=? AND desired_active=?", in.Key, in.Actor, vehicle, *in.Active).Scan(&existing, &state)
	if e == nil {
		respond(w, 200, map[string]any{"data": map[string]any{"id": existing, "state": state, "duplicate": true}})
		return
	}
	if e != sql.ErrNoRows {
		respond(w, 503, map[string]any{"error": "database_unavailable"})
		return
	}
	var seq int
	e = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(CAST(CONV(serial,16,10) AS UNSIGNED)),0)+1 FROM remote_output_commands WHERE device_id=?", device).Scan(&seq)
	if e != nil || seq > 65535 {
		respond(w, 409, map[string]any{"error": "serial_exhausted"})
		return
	}
	serial := fmt.Sprintf("%04X", seq)
	frame, e := Frame(profile.Password, *in.Active, serial)
	if e != nil {
		respond(w, 409, map[string]any{"error": "invalid_output_configuration"})
		return
	}
	res, e := tx.ExecContext(ctx, `INSERT INTO remote_output_commands(device_id,vehicle_id,requested_by,serial,request_key,desired_active,state,expires_at) VALUES(?,?,?,?,?,?,'dispatching',UTC_TIMESTAMP(3)+INTERVAL 2 MINUTE)`, device, vehicle, in.Actor, serial, in.Key, *in.Active)
	if e != nil {
		respond(w, 409, map[string]any{"error": "command_conflict"})
		return
	}
	id, _ := res.LastInsertId()
	if _, e = tx.ExecContext(ctx, `INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'request_output','remote_output_commands',?,?,UTC_TIMESTAMP(3))`, in.Actor, id, in.Key); e != nil {
		respond(w, 503, map[string]any{"error": "audit_unavailable"})
		return
	}
	if e = tx.Commit(); e != nil {
		respond(w, 503, map[string]any{"error": "command_not_committed"})
		return
	}
	s.mu.Lock()
	conn := s.connections[imei]
	s.mu.Unlock()
	state = "uncertain"
	if conn != nil {
		conn.mu.Lock()
		conn.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		n, err := io.WriteString(conn.conn, frame)
		conn.mu.Unlock()
		if err == nil && n == len(frame) {
			state = "sent"
		}
	}
	if s.CommandLog != nil {
		fields := strings.Split(frame, ",")
		fields[0] = "AT+GTDOS=[REDACTED]"
		s.CommandLog(imei, serial, strings.Join(fields, ","), state)
	}
	_, e = s.DB.ExecContext(ctx, "UPDATE remote_output_commands SET state=?,sent_at=IF(?='sent',UTC_TIMESTAMP(3),NULL) WHERE id=? AND state='dispatching'", state, state, id)
	if s.Notify != nil {
		s.Notify()
	}
	if e != nil {
		respond(w, 503, map[string]any{"error": "command_status_uncertain_do_not_retry"})
		return
	}
	respond(w, 202, map[string]any{"data": map[string]any{"id": id, "state": state, "serial": serial, "message": "ACK belum diterima; status mesin tidak disimpulkan"}})
}

// Record is called only after immutable raw persistence. An ACK does not confirm physical output.
func (s *Service) Record(ctx context.Context, payload string) error {
	reply, e := Decode(payload)
	if e != nil {
		return nil
	}
	if reply.Kind != "ack" {
		if s.Notify != nil {
			s.Notify()
		}
		return nil
	}
	_, e = s.DB.ExecContext(ctx, `UPDATE remote_output_commands c JOIN gps_devices d ON d.id=c.device_id JOIN device_raw_messages r ON r.id=(SELECT MAX(r2.id) FROM device_raw_messages r2 WHERE (r2.device_id=d.id OR r2.device_id IS NULL) AND r2.payload=?) SET c.state='acknowledged',c.acknowledged_at=r.received_at,c.acknowledged_raw_id=r.id WHERE d.imei=? AND c.serial=? AND c.state IN ('dispatching','sent','uncertain') AND c.created_at<=r.received_at AND c.expires_at>=r.received_at`, []byte(payload), reply.IMEI, reply.Serial)
	if s.Notify != nil {
		s.Notify()
	}
	return e
}
