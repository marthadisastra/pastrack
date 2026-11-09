package platform

import (
	"net/http"
	"pastrack/internal/fuel"
	"strconv"
	"strings"
	"time"
)

func (s *Server) calibration(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	p := strings.Split(path, "/")
	if len(p) != 3 || p[0] != "calibrations" || (p[2] != "points" && p[2] != "approve") {
		return false
	}
	id, e := parseID(p[1])
	if e != nil {
		fail(w, 400, "invalid_id")
		return true
	}
	if p[2] == "points" && r.Method == "GET" {
		rows, e := queryRows(r.Context(), s.DB, "SELECT id,input_value,output_value,evidence FROM fuel_calibration_points WHERE calibration_id=? ORDER BY input_value LIMIT 200", id)
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
	if u.Role == "viewer" || p[2] == "approve" && u.Role != "admin" {
		fail(w, 403, "permission_denied")
		return true
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return true
	}
	defer tx.Rollback()
	var state string
	if e = tx.QueryRowContext(r.Context(), "SELECT approval_state FROM fuel_calibrations WHERE id=? FOR UPDATE", id).Scan(&state); e != nil {
		fail(w, 404, "calibration_not_found")
		return true
	}
	if state == "approved" {
		fail(w, 409, "approved_calibration_is_immutable")
		return true
	}
	if p[2] == "points" {
		var input struct {
			Raw      float64 `json:"input_value"`
			Liters   float64 `json:"output_value"`
			Evidence string  `json:"evidence"`
		}
		if body(w, r, &input) != nil || input.Raw < 0 || input.Liters < 0 || len(input.Evidence) < 5 || len(input.Evidence) > 1000 {
			fail(w, 400, "invalid_calibration_evidence")
			return true
		}
		_, e = tx.ExecContext(r.Context(), "INSERT INTO fuel_calibration_points(calibration_id,input_value,output_value,evidence) VALUES(?,?,?,?)", id, input.Raw, input.Liters, input.Evidence)
		if e != nil {
			fail(w, 422, "invalid_or_duplicate_point")
			return true
		}
	} else {
		var c fuel.Calibration
		var tankID, sensorID, vehicleID int64
		var output string
		e = tx.QueryRowContext(r.Context(), `SELECT c.id,c.fuel_sensor_id,c.tank_profile_id,c.media,c.input_quantity,c.output_unit,vi.tank_capacity_l,t.vehicle_id FROM fuel_calibrations c JOIN fuel_tank_profiles t ON t.id=c.tank_profile_id JOIN vehicle_inventory vi ON vi.vehicle_id=t.vehicle_id WHERE c.id=?`, id).Scan(&c.ID, &sensorID, &tankID, &c.Media, &c.Quantity, &output, &c.Capacity, &vehicleID)
		if e != nil || output != "liters" {
			fail(w, 422, "tank_volume_profile_required")
			return true
		}
		c.Approved = true
		rows, e := tx.QueryContext(r.Context(), "SELECT input_value,output_value FROM fuel_calibration_points WHERE calibration_id=? ORDER BY input_value", id)
		if e != nil {
			databaseError(w, e)
			return true
		}
		for rows.Next() {
			var point fuel.Point
			if e = rows.Scan(&point.Raw, &point.Liters); e != nil {
				rows.Close()
				databaseError(w, e)
				return true
			}
			c.Points = append(c.Points, point)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			databaseError(w, e)
			return true
		}
		if c.Validate() != nil {
			fail(w, 422, "invalid_tank_calibration")
			return true
		}
		now := time.Now().UTC()
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_calibrations SET effective_to=? WHERE fuel_sensor_id=? AND approval_state='approved' AND effective_to IS NULL", now, sensorID); e != nil {
			databaseError(w, e)
			return true
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_calibrations SET approval_state='approved',approved_by=?,effective_from=? WHERE id=?", u.ID, now, id); e != nil {
			databaseError(w, e)
			return true
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_sensors SET tank_profile_id=?,decoding_state='calibrated_raw_signal' WHERE id=?", tankID, sensorID); e != nil {
			databaseError(w, e)
			return true
		}
	}
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,?, 'fuel_calibrations',?,?,UTC_TIMESTAMP(3))", u.ID, "calibration_"+p[2], id, token()); e != nil {
		databaseError(w, e)
		return true
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return true
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]string{"id": strconv.FormatInt(id, 10), "action": p[2]}})
	return true
}
