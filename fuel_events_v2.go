package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"pastrack/internal/fuel"
	"time"
)

// Fuel event evaluation is opt-in through explicit, enabled rules. No production
// threshold or theft claim is inferred from a single sensor observation.
func processFuelEvents(ctx context.Context, tx *sql.Tx, sensorID, vehicleID int64) error {
	rows, e := tx.QueryContext(ctx, `SELECT r.id,r.version,r.configuration FROM alert_rules r JOIN event_types t ON t.id=r.event_type_id WHERE r.enabled=1 AND t.code IN ('fuel_drop','refueling') LIMIT 100`)
	if e != nil {
		return e
	}
	type rule struct {
		ID, Version int64
		Config      string
	}
	rules := []rule{}
	for rows.Next() {
		var r rule
		if e = rows.Scan(&r.ID, &r.Version, &r.Config); e != nil {
			rows.Close()
			return e
		}
		rules = append(rules, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, r := range rules {
		var c struct {
			MinimumSamples int     `json:"minimum_samples"`
			MinimumDelta   float64 `json:"minimum_delta_liters"`
			MaximumGap     int     `json:"maximum_gap_seconds"`
			Confirmation   int     `json:"confirmation_seconds"`
			RequireStopped bool    `json:"require_stopped"`
		}
		if json.Unmarshal([]byte(r.Config), &c) != nil || c.MinimumSamples < 2 || c.MinimumSamples > 30 || c.MinimumDelta <= 0 || c.MaximumGap <= 0 || c.Confirmation <= 0 {
			continue
		}
		if c.RequireStopped {
			var speed sql.NullFloat64
			var at sql.NullTime
			e = tx.QueryRowContext(ctx, "SELECT p.speed_kmh,p.gps_time FROM latest_positions l JOIN gps_positions p ON p.id=l.position_id WHERE l.vehicle_id=?", vehicleID).Scan(&speed, &at)
			if e != nil && e != sql.ErrNoRows {
				return e
			}
			if !speed.Valid || !at.Valid || speed.Float64 > 3 || time.Since(at.Time) > 5*time.Minute {
				continue
			}
		}
		samples, err := tx.QueryContext(ctx, `SELECT id,received_at,fuel_liters,validity,calibration_id FROM fuel_readings WHERE fuel_sensor_id=? ORDER BY id DESC LIMIT ?`, sensorID, 2*c.MinimumSamples)
		if err != nil {
			return err
		}
		list := []fuel.Sample{}
		ids := []int64{}
		for samples.Next() {
			var s fuel.Sample
			var id int64
			var value sql.NullFloat64
			var validity string
			var calibration sql.NullInt64
			if err = samples.Scan(&id, &s.At, &value, &validity, &calibration); err != nil {
				samples.Close()
				return err
			}
			s.Valid = validity == "valid" && calibration.Valid
			s.CalibrationID = calibration.Int64
			if value.Valid {
				v := value.Float64
				s.Liters = &v
			}
			list = append(list, s)
			ids = append(ids, id)
		}
		err = samples.Err()
		samples.Close()
		if err != nil {
			return err
		}
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
			ids[i], ids[j] = ids[j], ids[i]
		}
		event := fuel.Detect(list, fuel.Rule{MinimumSamples: c.MinimumSamples, MinDeltaLiters: c.MinimumDelta, MaxGap: time.Duration(c.MaximumGap) * time.Second, Confirmation: time.Duration(c.Confirmation) * time.Second})
		if event == nil {
			continue
		}
		// Rule type must match the detected change direction.
		var kind string
		if err = tx.QueryRowContext(ctx, "SELECT t.code FROM alert_rules r JOIN event_types t ON t.id=r.event_type_id WHERE r.id=?", r.ID).Scan(&kind); err != nil {
			return err
		}
		if kind != event.Type {
			continue
		}
		result, err := tx.ExecContext(ctx, `INSERT IGNORE INTO fuel_events(vehicle_id,fuel_sensor_id,start_reading_id,end_reading_id,event_type,delta_l,event_at,rule_version,confidence) VALUES(?,?,?,?,?,?,?,?, 'sustained_change')`, vehicleID, sensorID, ids[0], ids[len(ids)-1], event.Type, event.DeltaLiters, event.At, fmt.Sprintf("%d:%d", r.ID, r.Version))
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			continue
		}
		eventID, _ := result.LastInsertId()
		if _, err = tx.ExecContext(ctx, "INSERT INTO alert_events(rule_id,fuel_event_id,vehicle_id,state,raised_at) VALUES(?,?,?,'open',?)", r.ID, eventID, vehicleID, event.At); err != nil {
			return err
		}
	}
	return nil
}
