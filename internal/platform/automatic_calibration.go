package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"pastrack/internal/fuel"
	"strings"
	"time"
)

type autoCalibrationSession struct {
	ID, Sensor, Fuel, Device, Vehicle, PhaseTelemetry int64
	Quantity, State                                   string
	Capacity                                          float64
	Started, PhaseStarted                             time.Time
	Empty, Full                                       sql.NullFloat64
	EmptyTelemetry, Calibration                       sql.NullInt64
	Error                                             sql.NullString
}

type calibrationSource struct {
	Fuel, Device, Vehicle, Telemetry int64
	MAC, IMEI, Name, Plate, Quantity string
	Capacity                         sql.NullFloat64
}

type stableCalibrationSample struct {
	ID                 int64
	Raw                float64
	Received, Measured time.Time
}

type stabilityResult struct {
	Ready       bool     `json:"ready"`
	Count       int      `json:"sample_count"`
	Seconds     float64  `json:"stable_seconds"`
	Raw         *float64 `json:"raw"`
	TelemetryID int64    `json:"telemetry_id"`
}

// A stable window needs distinct physical measurements received after this phase
// began. Buffered batches and repeated requests do not advance its duration.
func calibrationStability(samples []stableCalibrationSample, now, phase time.Time) stabilityResult {
	result := stabilityResult{}
	if len(samples) == 0 {
		return result
	}
	latest := samples[0]
	if now.Sub(latest.Received) > 30*time.Second || now.Sub(latest.Measured) > 30*time.Second || latest.Measured.After(now.Add(time.Minute)) {
		return result
	}
	result.Raw = &latest.Raw
	result.TelemetryID = latest.ID
	minimum, maximum := latest.Raw, latest.Raw
	seen := map[int64]bool{}
	previous := latest.Received
	for _, sample := range samples {
		if sample.Measured.Before(phase) || sample.Received.Before(phase) || sample.Measured.After(now.Add(time.Minute)) || previous.Sub(sample.Received) > 30*time.Second {
			break
		}
		if seen[sample.Measured.UnixMilli()] {
			continue
		}
		minimum = math.Min(minimum, sample.Raw)
		maximum = math.Max(maximum, sample.Raw)
		if maximum-minimum > 1 {
			break
		}
		seen[sample.Measured.UnixMilli()] = true
		result.Count++
		duration := math.Min(latest.Received.Sub(sample.Received).Seconds(), latest.Measured.Sub(sample.Measured).Seconds())
		result.Seconds = math.Max(0, duration)
		if result.Count >= 3 && duration >= 10 {
			result.Ready = true
			break
		}
		previous = sample.Received
	}
	return result
}

func scanAutoSession(tx *sql.Tx, ctx context.Context, sensor int64) (autoCalibrationSession, error) {
	var row autoCalibrationSession
	err := tx.QueryRowContext(ctx, `SELECT s.id,s.sensor_id,s.fuel_sensor_id,s.device_id,s.vehicle_id,s.quantity,vi.tank_capacity_l,s.state,s.started_at,s.phase_started_at,s.phase_telemetry_id,s.empty_raw,s.empty_telemetry_id,s.full_raw,s.calibration_id,s.error_message FROM sensor_calibration_sessions s JOIN vehicle_inventory vi ON vi.vehicle_id=s.vehicle_id WHERE s.sensor_id=? ORDER BY s.id DESC LIMIT 1 FOR UPDATE`, sensor).Scan(&row.ID, &row.Sensor, &row.Fuel, &row.Device, &row.Vehicle, &row.Quantity, &row.Capacity, &row.State, &row.Started, &row.PhaseStarted, &row.PhaseTelemetry, &row.Empty, &row.EmptyTelemetry, &row.Full, &row.Calibration, &row.Error)
	return row, err
}

func autoCalibrationSource(ctx context.Context, tx *sql.Tx, sensor int64) (calibrationSource, error) {
	var source calibrationSource
	var device sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT `+observedGateway+` FROM ble_sensors s WHERE s.id=? AND s.is_active=1 AND s.deleted_at IS NULL`, sensor).Scan(&device)
	if err != nil {
		return source, err
	}
	if !device.Valid {
		return source, fmt.Errorf("Belum ada GPS penerima dari raw sensor ini.")
	}
	source.Device = device.Int64
	err = tx.QueryRowContext(ctx, `SELECT f.id,s.mac,d.imei,COALESCE(n.reported_name,s.model),COALESCE((SELECT c.input_quantity FROM fuel_calibrations c WHERE c.fuel_sensor_id=f.id AND c.approval_state='approved' ORDER BY c.revision DESC,c.id DESC LIMIT 1),'fuel_height_raw') FROM ble_sensors s JOIN fuel_sensors f ON f.sensor_id=s.id JOIN gps_devices d ON d.id=? AND d.is_active=1 AND d.deleted_at IS NULL LEFT JOIN sensor_reported_names n ON n.sensor_id=s.id WHERE s.id=?`, source.Device, sensor).Scan(&source.Fuel, &source.MAC, &source.IMEI, &source.Name, &source.Quantity)
	if err != nil {
		return source, err
	}
	if len(source.MAC) != 12 || len(source.IMEI) != 15 {
		return source, fmt.Errorf("Identitas MAC sensor atau IMEI GPS belum valid.")
	}
	if source.Quantity != "volume_raw" && source.Quantity != "fuel_height_raw" {
		return source, fmt.Errorf("Sinyal kalibrasi sensor belum didukung.")
	}
	rows, err := tx.QueryContext(ctx, `SELECT v.id,v.plate_number FROM vehicle_assignments a JOIN vehicles v ON v.id=a.vehicle_id AND v.deleted_at IS NULL WHERE a.device_id=? AND a.starts_at<=UTC_TIMESTAMP(3) AND (a.ends_at IS NULL OR a.ends_at>UTC_TIMESTAMP(3)) ORDER BY a.id LIMIT 2`, source.Device)
	if err != nil {
		return source, err
	}
	count := 0
	for rows.Next() {
		if err = rows.Scan(&source.Vehicle, &source.Plate); err != nil {
			rows.Close()
			return source, err
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return source, err
	}
	if count != 1 {
		return source, fmt.Errorf("GPS penerima harus terhubung ke tepat satu kendaraan aktif.")
	}
	err = tx.QueryRowContext(ctx, "SELECT "+strings.ReplaceAll(vehicleCapacitySQL, "a.id", "v.id")+" FROM vehicles v WHERE v.id=? AND v.deleted_at IS NULL FOR UPDATE", source.Vehicle).Scan(&source.Capacity)
	if err != nil {
		return source, err
	}
	if !source.Capacity.Valid || source.Capacity.Float64 <= 0 || source.Capacity.Float64 > 999999999 {
		return source, fmt.Errorf("Isi kapasitas tangki di Edit Kendaraan terlebih dahulu.")
	}
	err = tx.QueryRowContext(ctx, "SELECT id FROM sensor_telemetry WHERE sensor_id=? AND device_id=? ORDER BY received_at DESC,id DESC LIMIT 1", sensor, source.Device).Scan(&source.Telemetry)
	return source, err
}

func autoSessionData(row autoCalibrationSession, source calibrationSource, stability stabilityResult) map[string]any {
	out := map[string]any{"session_id": row.ID, "state": row.State, "sensor_mac": source.MAC, "gps_imei": source.IMEI, "sensor_name": source.Name, "plate_number": source.Plate, "capacity_l": source.Capacity, "quantity": source.Quantity, "stability": stability, "empty_raw": nil, "full_raw": nil, "calibration_id": nil, "error": nil}
	if source.Capacity.Valid {
		out["capacity_l"] = source.Capacity.Float64
	} else {
		out["capacity_l"] = nil
	}
	if row.Empty.Valid {
		out["empty_raw"] = row.Empty.Float64
	}
	if row.Full.Valid {
		out["full_raw"] = row.Full.Float64
	}
	if row.Calibration.Valid {
		out["calibration_id"] = row.Calibration.Int64
	}
	if row.Error.Valid {
		out["error"] = row.Error.String
	}
	return out
}

func (s *Server) automaticCalibration(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "sensors" || parts[2] != "calibration-session" {
		return false
	}
	sensor, err := parseID(parts[1])
	if err != nil {
		fail(w, 400, "invalid_sensor")
		return true
	}
	if r.Method != "GET" && r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if r.Method == "POST" && (u.Role != "admin" || s.ReadOnly) {
		fail(w, 403, "Kalibrasi memerlukan akun admin.")
		return true
	}
	var input struct {
		Action    string `json:"action"`
		SessionID int64  `json:"session_id"`
	}
	if r.Method == "POST" {
		if body(w, r, &input) != nil {
			fail(w, 400, "invalid_calibration_action")
			return true
		}
		if input.Action != "start" && input.Action != "progress" && input.Action != "confirm_full" && input.Action != "cancel" {
			fail(w, 400, "invalid_calibration_action")
			return true
		}
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		databaseError(w, err)
		return true
	}
	defer tx.Rollback()
	var locked int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM ble_sensors WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", sensor).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "Sensor aktif tidak ditemukan.")
		return true
	}
	if err != nil {
		databaseError(w, err)
		return true
	}
	row, err := scanAutoSession(tx, ctx, sensor)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		databaseError(w, err)
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		row.State = "idle"
	}
	source, sourceErr := autoCalibrationSource(ctx, tx, sensor)
	now := time.Now().UTC().Truncate(time.Millisecond)
	running := row.State == "empty" || row.State == "fill" || row.State == "full"
	if r.Method == "GET" {
		data := autoSessionData(row, source, stabilityResult{})
		if sourceErr != nil {
			data["source_error"] = sourceErr.Error()
		}
		respond(w, 200, map[string]any{"data": data})
		return true
	}
	if input.Action == "cancel" && input.SessionID == row.ID && running {
		_, err = tx.ExecContext(ctx, "UPDATE sensor_calibration_sessions SET state='cancelled' WHERE id=?", row.ID)
		row.State = "cancelled"
	} else if input.Action == "start" {
		if sourceErr != nil {
			fail(w, 422, sourceErr.Error())
			return true
		}
		if !running {
			result, e := tx.ExecContext(ctx, `INSERT INTO sensor_calibration_sessions(sensor_id,fuel_sensor_id,device_id,vehicle_id,quantity,state,started_by,started_at,phase_started_at,phase_telemetry_id) VALUES(?,?,?,?,?,'empty',?,?,?,?)`, sensor, source.Fuel, source.Device, source.Vehicle, source.Quantity, u.ID, now, now, source.Telemetry)
			if e != nil {
				databaseError(w, e)
				return true
			}
			id, e := result.LastInsertId()
			if e != nil {
				databaseError(w, e)
				return true
			}
			row = autoCalibrationSession{ID: id, Sensor: sensor, Fuel: source.Fuel, Device: source.Device, Vehicle: source.Vehicle, Quantity: source.Quantity, Capacity: source.Capacity.Float64, State: "empty", Started: now, PhaseStarted: now, PhaseTelemetry: source.Telemetry}
		}
	} else if input.SessionID != row.ID || row.ID == 0 {
		fail(w, 409, "Sesi kalibrasi berubah. Buka kembali form.")
		return true
	}
	if err != nil {
		databaseError(w, err)
		return true
	}
	running = row.State == "empty" || row.State == "fill" || row.State == "full"
	if running && (sourceErr != nil || row.Quantity != source.Quantity || row.Device != source.Device || row.Vehicle != source.Vehicle || !source.Capacity.Valid || row.Capacity != source.Capacity.Float64 || now.Sub(row.Started) > 2*time.Hour) {
		message := "GPS, kendaraan, atau kapasitas berubah. Mulai ulang kalibrasi."
		if sourceErr != nil {
			message = sourceErr.Error()
		}
		if now.Sub(row.Started) > 2*time.Hour {
			message = "Sesi kalibrasi kedaluwarsa. Mulai ulang."
		}
		_, err = tx.ExecContext(ctx, "UPDATE sensor_calibration_sessions SET state='failed',error_message=? WHERE id=?", message, row.ID)
		row.State = "failed"
		row.Error = sql.NullString{String: message, Valid: true}
		running = false
	}
	if err != nil {
		databaseError(w, err)
		return true
	}
	if input.Action == "confirm_full" && row.State == "fill" {
		_, err = tx.ExecContext(ctx, "UPDATE sensor_calibration_sessions SET state='full',phase_started_at=?,phase_telemetry_id=? WHERE id=?", now, source.Telemetry, row.ID)
		row.State = "full"
		row.PhaseStarted = now
		row.PhaseTelemetry = source.Telemetry
	}
	if err != nil {
		databaseError(w, err)
		return true
	}
	completedNow := false
	stability := stabilityResult{}
	if running && (row.State == "empty" || row.State == "fill" || row.State == "full") && input.Action == "progress" {
		rows, e := tx.QueryContext(ctx, `SELECT st.id,st.`+row.Quantity+`,st.received_at,COALESCE(st.sensor_time,(SELECT MAX(p.gps_time) FROM gps_positions p WHERE p.raw_message_id=st.raw_message_id),st.received_at) FROM sensor_telemetry st JOIN device_raw_messages r ON r.id=st.raw_message_id AND r.device_id=st.device_id WHERE st.sensor_id=? AND st.device_id=? AND st.id>? AND st.received_at>=? AND st.validation_state='valid_raw' AND st.`+row.Quantity+` IS NOT NULL AND SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(r.payload USING utf8mb4),',',3),',',-1)=? ORDER BY st.received_at DESC,st.id DESC LIMIT 64`, sensor, row.Device, row.PhaseTelemetry, row.PhaseStarted, source.IMEI)
		if e != nil {
			databaseError(w, e)
			return true
		}
		samples := []stableCalibrationSample{}
		for rows.Next() {
			var sample stableCalibrationSample
			var measured sql.NullTime
			if e = rows.Scan(&sample.ID, &sample.Raw, &sample.Received, &measured); e != nil {
				break
			}
			if measured.Valid {
				sample.Measured = measured.Time
				samples = append(samples, sample)
			}
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			databaseError(w, e)
			return true
		}
		stability = calibrationStability(samples, now, row.PhaseStarted)
		if stability.Ready && row.State == "empty" {
			evidence, _ := json.Marshal(map[string]any{"telemetry_id": stability.TelemetryID, "sensor_id": sensor, "device_id": row.Device, "vehicle_id": row.Vehicle, "sample_count": stability.Count, "stable_seconds": stability.Seconds})
			_, err = tx.ExecContext(ctx, "UPDATE sensor_calibration_sessions SET state='fill',empty_raw=?,empty_telemetry_id=?,empty_evidence=? WHERE id=?", *stability.Raw, stability.TelemetryID, string(evidence), row.ID)
			row.State = "fill"
			row.Empty = sql.NullFloat64{Float64: *stability.Raw, Valid: true}
			row.EmptyTelemetry = sql.NullInt64{Int64: stability.TelemetryID, Valid: true}
		} else if stability.Ready && row.State == "full" && row.Empty.Valid && *stability.Raw > row.Empty.Float64+1 {
			err = saveAutomaticCalibration(ctx, tx, &row, stability, u.ID)
			completedNow = err == nil
		}
	}
	if err != nil {
		databaseError(w, err)
		return true
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return true
	}
	if completedNow {
		s.Notify()
	}
	respond(w, 200, map[string]any{"data": autoSessionData(row, source, stability)})
	return true
}

func saveAutomaticCalibration(ctx context.Context, tx *sql.Tx, row *autoCalibrationSession, stable stabilityResult, actor int64) error {
	calibration := fuel.Calibration{Approved: true, Media: "fuel_tank", Quantity: row.Quantity, Capacity: row.Capacity, Points: []fuel.Point{{Raw: row.Empty.Float64, Liters: 0}, {Raw: *stable.Raw, Liters: row.Capacity}}}
	if err := calibration.Validate(); err != nil {
		return err
	}
	var revision int64
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(revision),0)+1 FROM fuel_tank_profiles WHERE vehicle_id=?", row.Vehicle).Scan(&revision); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO fuel_tank_profiles(vehicle_id,name,revision,effective_from) VALUES(?,?,?,UTC_TIMESTAMP(3))", row.Vehicle, fmt.Sprintf("Kalibrasi sensor %d", row.Sensor), revision)
	if err != nil {
		return err
	}
	tank, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(revision),0)+1 FROM fuel_calibrations WHERE fuel_sensor_id=?", row.Fuel).Scan(&revision); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE fuel_calibrations SET effective_to=UTC_TIMESTAMP(3) WHERE fuel_sensor_id=? AND approval_state='approved' AND effective_to IS NULL", row.Fuel)
	if err != nil {
		return err
	}
	result, err = tx.ExecContext(ctx, `INSERT INTO fuel_calibrations(fuel_sensor_id,tank_profile_id,revision,media,input_quantity,input_unit,output_unit,approval_state,approved_by,effective_from) VALUES(?,?,?,'fuel_tank',?,'raw_counts','liters','approved',?,UTC_TIMESTAMP(3))`, row.Fuel, tank, revision, row.Quantity, actor)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for i, point := range calibration.Points {
		telemetry := row.EmptyTelemetry.Int64
		if i == 1 {
			telemetry = stable.TelemetryID
		}
		evidence, _ := json.Marshal(map[string]any{"session_id": row.ID, "telemetry_id": telemetry, "sensor_id": row.Sensor, "device_id": row.Device, "vehicle_id": row.Vehicle, "full_confirmed": i == 1})
		_, err = tx.ExecContext(ctx, "INSERT INTO fuel_calibration_points(calibration_id,input_value,output_value,evidence) VALUES(?,?,?,?)", id, point.Raw, point.Liters, string(evidence))
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, "UPDATE fuel_sensors SET tank_profile_id=?,decoding_state='calibrated_raw_signal' WHERE id=?", tank, row.Fuel)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE sensor_calibration_sessions SET state='complete',full_raw=?,full_telemetry_id=?,calibration_id=? WHERE id=?", *stable.Raw, stable.TelemetryID, id, row.ID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'automatic_calibration','fuel_calibrations',?,?,UTC_TIMESTAMP(3))", actor, id, token())
	if err != nil {
		return err
	}
	row.State = "complete"
	row.Full = sql.NullFloat64{Float64: *stable.Raw, Valid: true}
	row.Calibration = sql.NullInt64{Int64: id, Valid: true}
	return nil
}
