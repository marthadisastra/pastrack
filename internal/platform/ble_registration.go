package platform

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

// The gateway is selected from actual sensor observations, never from browser input.
const observedGateway = `(SELECT t.device_id FROM sensor_telemetry t JOIN device_raw_messages r ON r.id=t.raw_message_id AND r.device_id=t.device_id WHERE t.sensor_id=s.id ORDER BY t.received_at DESC,t.id DESC LIMIT 1)`

func (s *Server) bleRegistration(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	p := strings.Split(path, "/")
	if len(p) != 3 || p[0] != "vehicles" || (p[2] != "ble-options" && p[2] != "ble-registration") {
		return false
	}
	vehicle, err := parseID(p[1])
	if err != nil {
		fail(w, 400, "invalid_vehicle")
		return true
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	if p[2] == "ble-options" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		var exists int
		if err = s.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM vehicles WHERE id=? AND deleted_at IS NULL", vehicle).Scan(&exists); err != nil {
			databaseError(w, err)
			return true
		}
		if exists == 0 {
			fail(w, 404, "vehicle_not_found")
			return true
		}
		sensors, err := queryRows(r.Context(), s.DB, `SELECT s.id,s.sensor_identifier,s.model,d.id device_id,CONCAT('••••',RIGHT(d.imei,4)) gateway_identifier,st.last_received_at,st.status FROM ble_sensors s JOIN ble_sensor_status st ON st.sensor_id=s.id AND st.last_received_at IS NOT NULL JOIN gps_devices d ON d.id=`+observedGateway+` WHERE s.is_active=1 AND s.deleted_at IS NULL AND d.is_active=1 AND d.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM ble_sensor_bindings b WHERE b.sensor_id=s.id AND (b.ends_at IS NULL OR b.ends_at>UTC_TIMESTAMP(3)) AND b.vehicle_id IS NOT NULL AND b.vehicle_id<>?) AND NOT EXISTS(SELECT 1 FROM vehicle_assignments va WHERE va.device_id=d.id AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3)) AND va.vehicle_id<>?) ORDER BY s.id LIMIT 201`, vehicle, vehicle)
		if err != nil {
			databaseError(w, err)
			return true
		}
		tanks, err := queryRows(r.Context(), s.DB, `SELECT t.id,t.name,vi.tank_capacity_l capacity_l,t.vehicle_id,t.revision FROM fuel_tank_profiles t LEFT JOIN vehicle_inventory vi ON vi.vehicle_id=? WHERE (t.vehicle_id IS NULL OR t.vehicle_id=?) AND (t.effective_to IS NULL OR t.effective_to>UTC_TIMESTAMP(3)) ORDER BY t.id LIMIT 201`, vehicle, vehicle)
		if err != nil {
			databaseError(w, err)
			return true
		}
		more := len(sensors) > 200 || len(tanks) > 200
		if len(sensors) > 200 {
			sensors = sensors[:200]
		}
		if len(tanks) > 200 {
			tanks = tanks[:200]
		}
		respond(w, 200, map[string]any{"data": map[string]any{"vehicle_id": vehicle, "sensors": sensors, "tank_profiles": tanks}, "meta": map[string]any{"limit": 200, "has_more": more}})
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	var input struct {
		SensorID      int64 `json:"sensor_id"`
		TankProfileID int64 `json:"tank_profile_id"`
	}
	if body(w, r, &input) != nil || input.SensorID <= 0 || input.TankProfileID <= 0 {
		fail(w, 400, "sensor_and_tank_profile_required")
		return true
	}
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		databaseError(w, err)
		return true
	}
	defer tx.Rollback()
	now := time.Now().UTC().Truncate(time.Millisecond)
	var gateway sql.NullInt64
	if err = tx.QueryRowContext(r.Context(), `SELECT `+observedGateway+` FROM ble_sensors s JOIN ble_sensor_status st ON st.sensor_id=s.id AND st.last_received_at IS NOT NULL WHERE s.id=? AND s.is_active=1 AND s.deleted_at IS NULL`, input.SensorID).Scan(&gateway); err == sql.ErrNoRows || !gateway.Valid {
		fail(w, 409, "sensor_not_observed_by_server")
		return true
	} else if err != nil {
		databaseError(w, err)
		return true
	}
	var lock int64
	if err = tx.QueryRowContext(r.Context(), "SELECT id FROM gps_devices WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", gateway).Scan(&lock); err != nil {
		fail(w, 409, "observed_gateway_not_active")
		return true
	}
	if err = tx.QueryRowContext(r.Context(), "SELECT id FROM vehicles WHERE id=? AND deleted_at IS NULL FOR UPDATE", vehicle).Scan(&lock); err != nil {
		fail(w, 404, "vehicle_not_found")
		return true
	}
	if err = tx.QueryRowContext(r.Context(), "SELECT id FROM ble_sensors WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", input.SensorID).Scan(&lock); err != nil {
		fail(w, 409, "sensor_not_active")
		return true
	}
	var confirmed sql.NullInt64
	if err = tx.QueryRowContext(r.Context(), `SELECT `+observedGateway+` FROM ble_sensors s WHERE s.id=?`, input.SensorID).Scan(&confirmed); err != nil {
		databaseError(w, err)
		return true
	}
	if !confirmed.Valid || confirmed.Int64 != gateway.Int64 {
		fail(w, 409, "sensor_gateway_changed_reload_options")
		return true
	}
	var owner sql.NullInt64
	var end sql.NullTime
	if err = tx.QueryRowContext(r.Context(), "SELECT vehicle_id,effective_to FROM fuel_tank_profiles WHERE id=? FOR UPDATE", input.TankProfileID).Scan(&owner, &end); err == sql.ErrNoRows {
		fail(w, 404, "tank_profile_not_found")
		return true
	} else if err != nil {
		databaseError(w, err)
		return true
	}
	if owner.Valid && owner.Int64 != vehicle {
		fail(w, 409, "tank_profile_already_assigned")
		return true
	}
	if end.Valid && !end.Time.After(now) {
		fail(w, 409, "tank_profile_expired")
		return true
	}
	var conflicts int
	if err = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM vehicle_assignments WHERE device_id=? AND (ends_at IS NULL OR ends_at>?) AND vehicle_id<>?`, gateway, now, vehicle).Scan(&conflicts); err != nil {
		databaseError(w, err)
		return true
	}
	if conflicts > 0 {
		fail(w, 409, "gateway_already_assigned")
		return true
	}
	if err = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM ble_sensor_bindings WHERE sensor_id=? AND (ends_at IS NULL OR ends_at>?) AND ((vehicle_id IS NOT NULL AND vehicle_id<>?) OR starts_at>?)`, input.SensorID, now, vehicle, now).Scan(&conflicts); err != nil {
		databaseError(w, err)
		return true
	}
	if conflicts > 0 {
		fail(w, 409, "sensor_already_assigned")
		return true
	}
	var assignment int64
	err = tx.QueryRowContext(r.Context(), "SELECT id FROM vehicle_assignments WHERE device_id=? AND vehicle_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC LIMIT 1", gateway, vehicle, now, now).Scan(&assignment)
	if err == sql.ErrNoRows {
		result, e := tx.ExecContext(r.Context(), "INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'ble_registration')", vehicle, gateway, now)
		if e != nil {
			databaseError(w, e)
			return true
		}
		assignment, _ = result.LastInsertId()
	} else if err != nil {
		databaseError(w, err)
		return true
	}
	var binding int64
	err = tx.QueryRowContext(r.Context(), "SELECT id FROM ble_sensor_bindings WHERE sensor_id=? AND vehicle_id=? AND device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC LIMIT 1", input.SensorID, vehicle, gateway, now, now).Scan(&binding)
	if err == sql.ErrNoRows {
		if _, err = tx.ExecContext(r.Context(), "UPDATE ble_sensor_bindings SET ends_at=? WHERE sensor_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) AND (vehicle_id IS NULL OR vehicle_id=?)", now, input.SensorID, now, now, vehicle); err != nil {
			databaseError(w, err)
			return true
		}
		result, e := tx.ExecContext(r.Context(), "INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(?,?,?,?,'administrative')", input.SensorID, vehicle, gateway, now)
		if e != nil {
			databaseError(w, e)
			return true
		}
		binding, _ = result.LastInsertId()
	} else if err != nil {
		databaseError(w, err)
		return true
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE fuel_tank_profiles SET vehicle_id=? WHERE id=?", vehicle, input.TankProfileID); err != nil {
		databaseError(w, err)
		return true
	}
	if _, err = tx.ExecContext(r.Context(), "INSERT INTO fuel_sensors(sensor_id,tank_profile_id,decoding_state) VALUES(?,?,'raw_units_unverified') ON DUPLICATE KEY UPDATE tank_profile_id=VALUES(tank_profile_id)", input.SensorID, input.TankProfileID); err != nil {
		databaseError(w, err)
		return true
	}
	if _, err = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'ble_register','vehicles',?,?,?)", u.ID, vehicle, token(), now); err != nil {
		databaseError(w, err)
		return true
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return true
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"vehicle_id": vehicle, "sensor_id": input.SensorID, "tank_profile_id": input.TankProfileID, "device_id": gateway.Int64, "assignment_id": assignment, "binding_id": binding, "state": "registered", "physical_ignition_state": "NOT_INSTALLED", "fuel_calibration_required": true}})
	return true
}
