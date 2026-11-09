package main

import (
	"context"
	"database/sql"
	"pastrack/internal/alertengine"
	"pastrack/internal/fuel"
	"pastrack/internal/protocol"
	"pastrack/internal/telemetry"
	"time"
)

// enqueueFuelAlerts bridges persisted BLE readings into the centralized alert
// engine only after the transaction is committed. This keeps alert evaluation
// consistent with the same vehicle/sensor identity resolver used by telemetry.
func enqueueFuelAlerts(ctx context.Context, db *sql.DB, rawID int64) error {
	rows, err := db.QueryContext(ctx, "SELECT id,vehicle_id FROM fuel_readings WHERE telemetry_id IN (SELECT id FROM sensor_telemetry WHERE raw_message_id=?)", rawID)
	if err != nil {
		return err
	}
	defer rows.Close()
	repo := telemetry.MySQLRepository{DB: db}
	normalizer := telemetry.Normalizer{Facts: repo, Resolver: telemetry.Resolver{Repository: repo}}
	intake := alertengine.Intake{DB: db}
	for rows.Next() {
		var id int64
		var vehicleID sql.NullInt64
		if err = rows.Scan(&id, &vehicleID); err != nil {
			return err
		}
		value, e := normalizer.Fuel(ctx, id)
		// fuel_readings.vehicle_id is assigned during the same BLE persistence
		// transaction and is the authoritative vehicle identity for this sample.
		// BLE binding history can be absent for legacy installations, so do not
		// discard an otherwise valid reading when the resolver has no binding row.
		if value.VehicleID == nil && vehicleID.Valid && vehicleID.Int64 > 0 {
			value.VehicleID = &vehicleID.Int64
			value.ResolutionStatus = telemetry.Resolved
		}
		if e != nil || value.VehicleID == nil || value.ResolutionStatus != telemetry.Resolved {
			continue
		}
		if _, _, e = intake.Enqueue(ctx, alertengine.EventFromFuel(value)); e != nil {
			// Invalid/unassigned fuel is intentionally rejected by intake and must
			// not make GPS/BLE persistence fail.
			continue
		}
	}
	return rows.Err()
}

func persistBLE(ctx context.Context, tx *sql.Tx, device, rawID int64, received, reported time.Time, accessories []protocol.Accessory) error {
	for _, a := range accessories {
		if a.CFS == nil {
			continue
		}
		// Without an identity in a supported optional field, preserve the raw report
		// instead of manufacturing a sensor ID from the accessory name/index.
		if a.MAC == "" {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id),mac=IF(mac IS NULL,VALUES(mac),mac)`, a.MAC, a.MAC)
		if err != nil {
			return err
		}
		var sensor int64
		if err = tx.QueryRowContext(ctx, "SELECT id FROM ble_sensors WHERE mac=?", a.MAC).Scan(&sensor); err != nil {
			return err
		}
		if a.Name != "" && len(a.Name) <= 80 {
			at := reported
			if a.SensorTime != nil {
				at = *a.SensorTime
			}
			if !at.IsZero() {
				_, err = tx.ExecContext(ctx, `INSERT INTO sensor_reported_names(sensor_id,reported_name,reported_at,received_at,raw_message_id) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE reported_name=IF(reported_at<=VALUES(reported_at),VALUES(reported_name),reported_name),received_at=IF(reported_at<=VALUES(reported_at),VALUES(received_at),received_at),raw_message_id=IF(reported_at<=VALUES(reported_at),VALUES(raw_message_id),raw_message_id),reported_at=GREATEST(reported_at,VALUES(reported_at))`, sensor, a.Name, at, received, rawID)
				if err != nil {
					return err
				}
			}
		}
		var binding, vehicle sql.NullInt64
		at := received
		if a.SensorTime != nil {
			at = *a.SensorTime
		}
		rows, err := tx.QueryContext(ctx, `SELECT NULL,vehicle_id FROM vehicle_assignments WHERE device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC LIMIT 2`, device, at, at)
		if err != nil {
			return err
		}
		count := 0
		for rows.Next() {
			if err = rows.Scan(&binding, &vehicle); err != nil {
				rows.Close()
				return err
			}
			count++
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if count > 1 {
			binding = sql.NullInt64{}
			vehicle = sql.NullInt64{}
		}
		// Optional historical binding metadata never overrides the receiving GPS.
		err = tx.QueryRowContext(ctx, `SELECT CASE WHEN COUNT(*)=1 THEN MAX(id) END FROM ble_sensor_bindings WHERE sensor_id=? AND device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?)`, sensor, device, at, at).Scan(&binding)
		if err != nil {
			return err
		}
		state := "valid_raw"
		if count > 1 {
			state = "ambiguous_binding"
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO sensor_telemetry(sensor_id,device_id,binding_id,raw_message_id,accessory_index,sensor_time,received_at,accessory_type,accessory_model,append_mask,payload_hex,tilt_raw,fuel_height_raw,volume_raw,battery_mv,battery_percent,temperature_c,rssi,validation_state,decoder_version) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, sensor, device, binding, rawID, a.Index, a.SensorTime, received, a.Type, a.Model, a.AppendMask, a.Raw, a.CFS.TiltRaw, a.CFS.FuelHeightRaw, a.CFS.VolumeRaw, a.BatteryMV, a.BatteryPercent, a.TemperatureC, a.RSSI, state, "cfs300b-v0306-raw-v1")
		if err != nil {
			return err
		}
		telemetry, _ := result.LastInsertId()
		status := "observed"
		if a.Connected != nil {
			status = "disconnected"
			if *a.Connected {
				status = "connected"
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO ble_connection_logs(sensor_id,device_id,raw_message_id,event_at,received_at,status,source) VALUES(?,?,?,?,?,?,'protocol')`, sensor, device, rawID, a.SensorTime, received, status)
			if err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ble_sensor_status(sensor_id,telemetry_id,last_received_at,status,stale_after_seconds) VALUES(?,?,?,?,300) ON DUPLICATE KEY UPDATE telemetry_id=IF(last_received_at<=VALUES(last_received_at),VALUES(telemetry_id),telemetry_id),status=IF(last_received_at<=VALUES(last_received_at),VALUES(status),status),last_received_at=GREATEST(last_received_at,VALUES(last_received_at))`, sensor, telemetry, received, status)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'raw_units_unverified') ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, sensor)
		if err != nil {
			return err
		}
		var fuelID int64
		var tank sql.NullInt64
		if err = tx.QueryRowContext(ctx, "SELECT id,tank_profile_id FROM fuel_sensors WHERE sensor_id=?", sensor).Scan(&fuelID, &tank); err != nil {
			return err
		}
		reading := fuel.Reading{Validity: "uncalibrated"}
		var raw any
		var unit any
		var calibration any
		if status == "disconnected" {
			reading.Validity = "disconnected"
		} else if vehicle.Valid && tank.Valid && count == 1 {
			var c fuel.Calibration
			var approved string
			err = tx.QueryRowContext(ctx, `SELECT c.id,c.media,c.input_quantity,c.approval_state,vi.tank_capacity_l FROM fuel_calibrations c JOIN fuel_tank_profiles t ON t.id=c.tank_profile_id JOIN vehicle_inventory vi ON vi.vehicle_id=t.vehicle_id WHERE c.fuel_sensor_id=? AND c.tank_profile_id=? AND t.vehicle_id=? AND vi.tank_capacity_l>0 AND c.approval_state='approved' AND c.output_unit='liters' AND c.effective_from<=? AND (c.effective_to IS NULL OR c.effective_to>?) ORDER BY c.revision DESC LIMIT 1`, fuelID, tank, vehicle, at, at).Scan(&c.ID, &c.Media, &c.Quantity, &approved, &c.Capacity)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil {
				c.Approved = true
				points, e := tx.QueryContext(ctx, "SELECT input_value,output_value FROM fuel_calibration_points WHERE calibration_id=? ORDER BY input_value", c.ID)
				if e != nil {
					return e
				}
				for points.Next() {
					var p fuel.Point
					if e = points.Scan(&p.Raw, &p.Liters); e != nil {
						points.Close()
						return e
					}
					c.Points = append(c.Points, p)
				}
				e = points.Err()
				points.Close()
				if e != nil {
					return e
				}
				// Calibration points preserve the measured tank curve. Scale its
				// historical full-volume output to the vehicle's single capacity.
				if len(c.Points) > 0 {
					full := c.Points[len(c.Points)-1].Liters
					if full > 0 {
						for i := range c.Points {
							c.Points[i].Liters = c.Points[i].Liters / full * c.Capacity
						}
					}
				}
				value := float64(a.CFS.FuelHeightRaw)
				if c.Quantity == "volume_raw" {
					value = float64(a.CFS.VolumeRaw)
				}
				raw = value
				unit = c.Quantity
				reading = c.Convert(&value)
				calibration = c.ID
			}
		}
		result, err = tx.ExecContext(ctx, `INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,calibration_id,reading_at,received_at,raw_measurement,raw_unit,fuel_liters,fuel_percent,validity) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, fuelID, telemetry, vehicle, calibration, a.SensorTime, received, raw, unit, reading.Liters, reading.Percent, reading.Validity)
		if err != nil {
			return err
		}
		readingID, _ := result.LastInsertId()
		var valid any
		if reading.Validity == "valid" {
			valid = readingID
		}
		// Current observation advances independently of the last valid volume.
		_, err = tx.ExecContext(ctx, `INSERT INTO latest_fuel_readings(fuel_sensor_id,reading_id,last_valid_reading_id,updated_at) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE reading_id=IF(updated_at<=VALUES(updated_at),VALUES(reading_id),reading_id),last_valid_reading_id=IF(updated_at<=VALUES(updated_at),COALESCE(VALUES(last_valid_reading_id),last_valid_reading_id),last_valid_reading_id),updated_at=GREATEST(updated_at,VALUES(updated_at))`, fuelID, readingID, valid, received)
		if err != nil {
			return err
		}
		if vehicle.Valid && reading.Validity == "valid" {
			if err = processFuelEvents(ctx, tx, fuelID, vehicle.Int64); err != nil {
				return err
			}
		}
	}
	return nil
}
