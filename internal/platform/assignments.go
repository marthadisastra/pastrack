package platform

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Assignments retain history and serialize modifications against their inventory row.
func (s *Server) assignment(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[2] != "assignments" || (parts[0] != "vehicles" && parts[0] != "sensors") {
		return false
	}
	entity, e := parseID(parts[1])
	if e != nil {
		fail(w, 400, "invalid_id")
		return true
	}
	sensor := parts[0] == "sensors"
	table := "vehicle_assignments"
	field := "vehicle_id"
	if sensor {
		table = "ble_sensor_bindings"
		field = "sensor_id"
	}
	if r.Method == "GET" {
		rows, e := queryRows(r.Context(), s.DB, "SELECT id,"+field+",device_id,starts_at,ends_at FROM "+table+" WHERE "+field+"=? ORDER BY starts_at DESC LIMIT 200", entity)
		if e != nil {
			databaseError(w, e)
			return true
		}
		respond(w, 200, map[string]any{"data": rows})
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	var v struct {
		DeviceID  int64      `json:"device_id"`
		VehicleID *int64     `json:"vehicle_id"`
		StartsAt  time.Time  `json:"starts_at"`
		EndsAt    *time.Time `json:"ends_at"`
	}
	if body(w, r, &v) != nil || v.DeviceID <= 0 || v.StartsAt.IsZero() || v.EndsAt != nil && !v.EndsAt.After(v.StartsAt) {
		fail(w, 400, "invalid_assignment")
		return true
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return true
	}
	defer tx.Rollback()
	var locked int64
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM gps_devices WHERE id=? AND deleted_at IS NULL FOR UPDATE", v.DeviceID).Scan(&locked); e != nil {
		fail(w, 404, "device_not_found")
		return true
	}
	lockTable, lockID := "vehicles", entity
	if sensor {
		lockTable = "ble_sensors"
	}
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM "+lockTable+" WHERE id=? AND deleted_at IS NULL FOR UPDATE", lockID).Scan(&locked); e != nil {
		fail(w, 404, "entity_not_found")
		return true
	}
	var overlap int
	end := any(v.EndsAt)
	if v.EndsAt == nil {
		end = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	where := "device_id=?"
	subject := v.DeviceID
	if sensor {
		where = "sensor_id=?"
		subject = entity
	}
	e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+table+" WHERE "+where+" AND starts_at<? AND (ends_at IS NULL OR ends_at>?)", subject, end, v.StartsAt).Scan(&overlap)
	if e != nil {
		databaseError(w, e)
		return true
	}
	if overlap > 0 {
		fail(w, 409, "overlapping_assignment")
		return true
	}
	var result sql.Result
	if sensor {
		result, e = tx.ExecContext(r.Context(), "INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,ends_at,binding_source) VALUES(?,?,?,?,?,'administrative')", entity, v.VehicleID, v.DeviceID, v.StartsAt.UTC(), v.EndsAt)
	} else {
		result, e = tx.ExecContext(r.Context(), "INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,ends_at,provenance) VALUES(?,?,?,?, 'administrative')", entity, v.DeviceID, v.StartsAt.UTC(), v.EndsAt)
	}
	if e != nil {
		fail(w, 422, "invalid_assignment_relation")
		return true
	}
	id, _ := result.LastInsertId()
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'assign',?,?,?,UTC_TIMESTAMP(3))", u.ID, table, id, token()); e != nil {
		databaseError(w, e)
		return true
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return true
	}
	s.Notify()
	respond(w, 201, map[string]any{"data": map[string]any{"id": id}})
	return true
}
func validateNumber(v any, min, max float64) error {
	n, ok := v.(float64)
	if !ok || n < min || n > max {
		return fmt.Errorf("invalid numeric value")
	}
	return nil
}
