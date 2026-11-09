package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"pastrack/internal/fuel"
	"strings"
	"time"
)

// setup keeps physical measurements server-sourced and calibration revisions immutable.
func (s *Server) simpleSetup(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	p := strings.Split(path, "/")
	if len(p) != 3 || !((p[0] == "devices" && p[2] == "connect-vehicle") || (p[0] == "sensors" && p[2] == "quick-calibration")) {
		return false
	}
	id, err := parseID(p[1])
	if err != nil {
		fail(w, 400, "invalid_id")
		return true
	}
	if p[0] == "devices" {
		s.connectVehicle(w, r, u, id)
		return true
	}
	s.quickCalibration(w, r, u, id)
	return true
}
func (s *Server) connectVehicle(w http.ResponseWriter, r *http.Request, u *Identity, id int64) {
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return
	}
	var v struct {
		VehicleID int64           `json:"vehicle_id"`
		Transfer  *deviceTransfer `json:"transfer"`
	}
	if body(w, r, &v) != nil || v.VehicleID <= 0 {
		fail(w, 400, "Pilih kendaraan.")
		return
	}
	tx, e := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		databaseError(w, e)
		return
	}
	defer tx.Rollback()
	var locked int64
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM gps_devices WHERE imei=(SELECT imei FROM gps_devices WHERE id=?) AND is_active=1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&locked); e != nil {
		fail(w, 404, "GPS tidak aktif atau tidak ditemukan.")
		return
	}
	id = locked
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM vehicles WHERE id=? AND deleted_at IS NULL FOR UPDATE", v.VehicleID).Scan(&locked); e != nil {
		fail(w, 404, "Kendaraan tidak ditemukan.")
		return
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	var assignment int64
	action := "connect_vehicle"
	if v.Transfer != nil {
		assignment, action, e = transferGPS(r.Context(), tx, id, v.VehicleID, *v.Transfer)
		if errors.Is(e, errAssignmentChanged) {
			fail(w, 409, e.Error())
			return
		}
		if e != nil {
			databaseError(w, e)
			return
		}
	} else {
		var conflicts int
		e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM vehicle_assignments WHERE (device_id=? OR vehicle_id=?) AND (ends_at IS NULL OR ends_at>?) AND NOT(device_id=? AND vehicle_id=? AND starts_at<=?)", id, v.VehicleID, now, id, v.VehicleID, now).Scan(&conflicts)
		if e != nil {
			databaseError(w, e)
			return
		}
		if conflicts > 0 {
			fail(w, 409, "GPS atau kendaraan sudah terhubung. Lepaskan relasi sebelumnya terlebih dahulu.")
			return
		}
		e = tx.QueryRowContext(r.Context(), "SELECT id FROM vehicle_assignments WHERE device_id=? AND vehicle_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?)", id, v.VehicleID, now, now).Scan(&assignment)
		if e == sql.ErrNoRows {
			res, err := tx.ExecContext(r.Context(), "INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,?,'device_management')", id, v.VehicleID, now)
			if err != nil {
				databaseError(w, err)
				return
			}
			assignment, _ = res.LastInsertId()
		} else if e != nil {
			databaseError(w, e)
			return
		}
	}
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,?,'vehicle_assignments',?,?,UTC_TIMESTAMP(3))", u.ID, action, assignment, token()); e != nil {
		databaseError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"assignment_id": assignment, "vehicle_id": v.VehicleID, "device_id": id}})
}

type setupSample struct {
	DeviceID int64        `json:"device_id"`
	ID       int64        `json:"id"`
	Raw      float64      `json:"value"`
	Received time.Time    `json:"received_at"`
	Measured sql.NullTime `json:"-"`
}

func sampleFresh(p setupSample) bool {
	now := time.Now().UTC()
	return now.Sub(p.Received) <= 5*time.Minute && p.Received.Sub(now) <= time.Minute && (!p.Measured.Valid || (now.Sub(p.Measured.Time) <= 5*time.Minute && p.Measured.Time.Sub(now) <= time.Minute))
}
func (s *Server) quickCalibration(w http.ResponseWriter, r *http.Request, u *Identity, id int64) {
	if r.Method != "GET" && r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return
	}
	if r.Method == "POST" && u.Role != "admin" {
		fail(w, 403, "Kalibrasi memerlukan akun admin.")
		return
	}
	if r.Method == "GET" {
		rows, e := queryRows(r.Context(), s.DB, `SELECT s.id,s.sensor_identifier,s.model,(SELECT n.reported_name FROM sensor_reported_names n WHERE n.sensor_id=s.id) sensor_name,gateway.id device_id,f.id fuel_sensor_id,f.tank_profile_id,`+strings.ReplaceAll(vehicleCapacitySQL, "a.id", "v.id")+` capacity_l,va.vehicle_id vehicle_id,v.plate_number FROM ble_sensors s LEFT JOIN fuel_sensors f ON f.sensor_id=s.id LEFT JOIN fuel_tank_profiles t ON t.id=f.tank_profile_id LEFT JOIN gps_devices gateway ON gateway.id=`+observedGateway+` AND gateway.is_active=1 AND gateway.deleted_at IS NULL LEFT JOIN vehicle_assignments va ON va.device_id=gateway.id AND va.starts_at<=UTC_TIMESTAMP(3) AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3)) LEFT JOIN vehicles v ON v.id=va.vehicle_id WHERE s.id=? AND s.is_active=1 AND s.deleted_at IS NULL`, id)
		if e != nil {
			databaseError(w, e)
			return
		}
		if len(rows) != 1 {
			fail(w, 404, "Sensor aktif tidak ditemukan atau relasi sensor ambigu.")
			return
		}
		samples, e := queryRows(r.Context(), s.DB, `SELECT t.id,t.volume_raw,t.fuel_height_raw,t.received_at,COALESCE(t.sensor_time,r.device_time,(SELECT MAX(g.gps_time) FROM gps_positions g WHERE g.raw_message_id=t.raw_message_id)) measured_at FROM sensor_telemetry t JOIN device_raw_messages r ON r.id=t.raw_message_id WHERE t.sensor_id=? AND t.device_id=? AND t.validation_state='valid_raw' ORDER BY t.received_at DESC,t.id DESC LIMIT 1`, id, rows[0]["device_id"])
		if e != nil {
			databaseError(w, e)
			return
		}
		drafts, e := queryRows(r.Context(), s.DB, `SELECT c.id,c.input_quantity,p.input_value,p.output_value,p.evidence FROM fuel_calibrations c JOIN fuel_sensors f ON f.id=c.fuel_sensor_id LEFT JOIN fuel_calibration_points p ON p.calibration_id=c.id WHERE f.sensor_id=? AND c.approval_state='draft' AND c.media='fuel_tank' AND c.input_unit='two_point_capture' AND c.id=(SELECT MAX(c2.id) FROM fuel_calibrations c2 WHERE c2.fuel_sensor_id=f.id AND c2.input_unit='two_point_capture') ORDER BY p.output_value LIMIT 2`, id)
		if e != nil {
			databaseError(w, e)
			return
		}
		respond(w, 200, map[string]any{"data": rows[0], "samples": samples, "draft": drafts, "server_time": time.Now().UTC()})
		return
	}
	var in struct {
		Action        string  `json:"action"`
		CalibrationID int64   `json:"calibration_id"`
		Capacity      float64 `json:"capacity_l"`
		Quantity      string  `json:"quantity"`
	}
	if body(w, r, &in) != nil || (in.Action != "empty" && in.Action != "full" && in.Action != "save") || (in.Quantity != "volume_raw" && in.Quantity != "fuel_height_raw") {
		fail(w, 400, "invalid_calibration_request")
		return
	}
	tx, e := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		databaseError(w, e)
		return
	}
	defer tx.Rollback()
	var sensor int64
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM ble_sensors WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&sensor); e != nil {
		fail(w, 404, "Sensor tidak ditemukan.")
		return
	}
	var fs int64
	e = tx.QueryRowContext(r.Context(), "SELECT id FROM fuel_sensors WHERE sensor_id=? FOR UPDATE", id).Scan(&fs)
	if e != nil {
		fail(w, 422, "Sensor belum terdaftar sebagai sensor fuel.")
		return
	}
	var automatic int
	if e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM sensor_calibration_sessions WHERE sensor_id=? AND state IN ('empty','fill','full')", id).Scan(&automatic); e != nil {
		databaseError(w, e)
		return
	}
	if automatic > 0 {
		fail(w, 409, "Sesi kalibrasi otomatis sedang berjalan.")
		return
	}
	var vehicle int64
	var bindings int
	var gateway int64
	e = tx.QueryRowContext(r.Context(), `SELECT COUNT(*),COALESCE(MAX(va.vehicle_id),0),COALESCE(MAX(va.device_id),0) FROM vehicle_assignments va JOIN gps_devices d ON d.id=va.device_id AND d.is_active=1 AND d.deleted_at IS NULL WHERE va.device_id=(SELECT `+observedGateway+` FROM ble_sensors s WHERE s.id=?) AND va.starts_at<=UTC_TIMESTAMP(3) AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3))`, id).Scan(&bindings, &vehicle, &gateway)
	if e != nil {
		databaseError(w, e)
		return
	}
	if bindings != 1 {
		fail(w, 422, "Hubungkan GPS penerima sensor ke kendaraan di Device Management terlebih dahulu.")
		return
	}
	if e = tx.QueryRowContext(r.Context(), "SELECT tank_capacity_l FROM vehicle_inventory WHERE vehicle_id=? AND tank_capacity_l>0 FOR UPDATE", vehicle).Scan(&in.Capacity); e != nil {
		fail(w, 422, "Isi kapasitas tangki di Edit Kendaraan terlebih dahulu.")
		return
	}
	cid := in.CalibrationID
	if cid == 0 && in.Action == "empty" {
		var revision int64
		if e = tx.QueryRowContext(r.Context(), "SELECT COALESCE(MAX(revision),0)+1 FROM fuel_calibrations WHERE fuel_sensor_id=?", fs).Scan(&revision); e != nil {
			databaseError(w, e)
			return
		}
		res, err := tx.ExecContext(r.Context(), "INSERT INTO fuel_calibrations(fuel_sensor_id,revision,media,input_quantity,input_unit,output_unit,approval_state) VALUES(?,?,'fuel_tank',?,'two_point_capture','liters','draft')", fs, revision, in.Quantity)
		if err != nil {
			databaseError(w, err)
			return
		}
		cid, _ = res.LastInsertId()
	}
	var state, quantity, unit string
	if e = tx.QueryRowContext(r.Context(), "SELECT approval_state,input_quantity,input_unit FROM fuel_calibrations WHERE id=? AND fuel_sensor_id=? FOR UPDATE", cid, fs).Scan(&state, &quantity, &unit); e != nil || state != "draft" || quantity != in.Quantity || unit != "two_point_capture" {
		fail(w, 409, "Draft kalibrasi tidak sesuai. Muat ulang form.")
		return
	}
	if in.Action != "save" {
		var sample setupSample
		var raw sql.NullFloat64
		e = tx.QueryRowContext(r.Context(), "SELECT t.id,t.device_id,t."+in.Quantity+",t.received_at,COALESCE(t.sensor_time,r.device_time,(SELECT MAX(g.gps_time) FROM gps_positions g WHERE g.raw_message_id=t.raw_message_id)) FROM sensor_telemetry t JOIN device_raw_messages r ON r.id=t.raw_message_id WHERE t.sensor_id=? AND t.device_id=? AND t.validation_state='valid_raw' ORDER BY t.received_at DESC,t.id DESC LIMIT 1", id, gateway).Scan(&sample.ID, &sample.DeviceID, &raw, &sample.Received, &sample.Measured)
		if e != nil || !raw.Valid {
			fail(w, 422, "Belum ada nilai sensor yang dapat direkam.")
			return
		}
		sample.Raw = raw.Float64
		var disconnected int
		if e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM ble_sensor_status WHERE sensor_id=? AND status='disconnected'", id).Scan(&disconnected); e != nil {
			databaseError(w, e)
			return
		}
		if disconnected > 0 {
			fail(w, 409, "Sensor sedang terputus. Tunggu kiriman sensor yang terhubung.")
			return
		}
		if sample.DeviceID != gateway {
			fail(w, 409, "Kiriman terbaru berasal dari GPS berbeda. Periksa relasi sensor.")
			return
		}
		if !sampleFresh(sample) {
			fail(w, 409, "Data sensor sudah kedaluwarsa. Tunggu kiriman baru (maksimal 5 menit).")
			return
		}
		liters := 0.0
		if in.Action == "full" {
			liters = 1
			var empty float64
			var evidence string
			if e = tx.QueryRowContext(r.Context(), "SELECT input_value,evidence FROM fuel_calibration_points WHERE calibration_id=? AND output_value=0", cid).Scan(&empty, &evidence); e != nil {
				fail(w, 422, "Klik Set Empty terlebih dahulu.")
				return
			}
			var prior struct {
				Vehicle   int64 `json:"vehicle_id"`
				Telemetry int64 `json:"telemetry_id"`
			}
			if json.Unmarshal([]byte(evidence), &prior) != nil || prior.Vehicle != vehicle || sample.ID <= prior.Telemetry {
				fail(w, 409, "Tunggu kiriman baru setelah Empty dan pastikan kendaraan sensor tidak berubah.")
				return
			}
			if sample.Raw <= empty {
				fail(w, 422, "Nilai Full harus lebih besar dari Empty. Pastikan tangki penuh dan tunggu data sensor baru.")
				return
			}
		}
		// Empty and full are durable measurements; full uses a normalized marker until capacity is saved.
		if _, e = tx.ExecContext(r.Context(), "DELETE FROM fuel_calibration_points WHERE calibration_id=? AND output_value=?", cid, liters); e != nil {
			databaseError(w, e)
			return
		}
		if in.Action == "empty" {
			if _, e = tx.ExecContext(r.Context(), "DELETE FROM fuel_calibration_points WHERE calibration_id=? AND output_value=1", cid); e != nil {
				databaseError(w, e)
				return
			}
		}
		evidence, _ := json.Marshal(map[string]any{"telemetry_id": sample.ID, "received_at": sample.Received, "raw": sample.Raw, "capture": in.Action, "actor_id": u.ID, "vehicle_id": vehicle})
		if _, e = tx.ExecContext(r.Context(), "INSERT INTO fuel_calibration_points(calibration_id,input_value,output_value,evidence) VALUES(?,?,?,?)", cid, sample.Raw, liters, string(evidence)); e != nil {
			databaseError(w, e)
			return
		}
	} else {
		var capacity sql.NullFloat64
		e = tx.QueryRowContext(r.Context(), "SELECT "+strings.ReplaceAll(vehicleCapacitySQL, "a.id", "v.id")+" FROM vehicles v WHERE v.id=? AND v.deleted_at IS NULL FOR UPDATE", vehicle).Scan(&capacity)
		if e != nil {
			databaseError(w, e)
			return
		}
		if !capacity.Valid || capacity.Float64 <= 0 || capacity.Float64 > 999999999 {
			fail(w, 422, "Isi kapasitas tangki di Edit Kendaraan terlebih dahulu.")
			return
		}
		in.Capacity = capacity.Float64
		c := fuel.Calibration{Approved: true, Media: "fuel_tank", Quantity: quantity, Capacity: in.Capacity}
		rows, err := tx.QueryContext(r.Context(), "SELECT input_value,output_value,evidence FROM fuel_calibration_points WHERE calibration_id=? ORDER BY output_value", cid)
		if err != nil {
			databaseError(w, err)
			return
		}
		for rows.Next() {
			var p fuel.Point
			var evidence string
			if e = rows.Scan(&p.Raw, &p.Liters, &evidence); e != nil {
				break
			}
			var captured struct {
				Vehicle int64 `json:"vehicle_id"`
			}
			if json.Unmarshal([]byte(evidence), &captured) != nil || captured.Vehicle != vehicle {
				e = fmt.Errorf("sensor vehicle changed during calibration")
				break
			}
			p.Liters *= in.Capacity
			c.Points = append(c.Points, p)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			databaseError(w, e)
			return
		}
		if len(c.Points) != 2 || c.Validate() != nil || in.Capacity > 999999999 {
			fail(w, 422, "Rekam Empty dan Full yang berbeda, lalu pastikan kapasitas tangki di kendaraan sudah diisi.")
			return
		}
		var vehicleLock int64
		if e = tx.QueryRowContext(r.Context(), "SELECT id FROM vehicles WHERE id=? AND deleted_at IS NULL FOR UPDATE", vehicle).Scan(&vehicleLock); e != nil {
			fail(w, 409, "Kendaraan tidak aktif.")
			return
		}
		var revision int64
		if e = tx.QueryRowContext(r.Context(), "SELECT COALESCE(MAX(revision),0)+1 FROM fuel_tank_profiles WHERE vehicle_id=?", vehicle).Scan(&revision); e != nil {
			databaseError(w, e)
			return
		}
		res, err := tx.ExecContext(r.Context(), "INSERT INTO fuel_tank_profiles(vehicle_id,name,revision,effective_from) VALUES(?,?,?,UTC_TIMESTAMP(3))", vehicle, fmt.Sprintf("Kalibrasi sensor %d", id), revision)
		if err != nil {
			databaseError(w, err)
			return
		}
		tank, _ := res.LastInsertId()
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_calibration_points SET output_value=output_value*? WHERE calibration_id=?", in.Capacity, cid); e != nil {
			databaseError(w, e)
			return
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_calibrations SET effective_to=UTC_TIMESTAMP(3) WHERE fuel_sensor_id=? AND approval_state='approved' AND effective_to IS NULL", fs); e != nil {
			databaseError(w, e)
			return
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_calibrations SET tank_profile_id=?,approval_state='approved',approved_by=?,effective_from=UTC_TIMESTAMP(3) WHERE id=?", tank, u.ID, cid); e != nil {
			databaseError(w, e)
			return
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE fuel_sensors SET tank_profile_id=?,decoding_state='calibrated_raw_signal' WHERE id=?", tank, fs); e != nil {
			databaseError(w, e)
			return
		}
	}
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,?,'fuel_calibrations',?,?,UTC_TIMESTAMP(3))", u.ID, "capture_"+in.Action, cid, token()); e != nil {
		databaseError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"calibration_id": cid, "action": in.Action}})
}
