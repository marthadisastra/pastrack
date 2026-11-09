package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type MySQLRepository struct{ DB Queryer }

func (r MySQLRepository) GPSDevice(ctx context.Context, identity GPSIdentity) (GPSDevice, error) {
	var value GPSDevice
	var row *sql.Row
	switch {
	case identity.DeviceID > 0 && identity.IMEI != "":
		row = r.DB.QueryRowContext(ctx, "SELECT id,imei FROM gps_devices WHERE id=? AND imei=?", identity.DeviceID, identity.IMEI)
	case identity.DeviceID > 0:
		row = r.DB.QueryRowContext(ctx, "SELECT id,imei FROM gps_devices WHERE id=?", identity.DeviceID)
	case identity.IMEI != "":
		row = r.DB.QueryRowContext(ctx, "SELECT id,imei FROM gps_devices WHERE imei=?", identity.IMEI)
	default:
		return value, sql.ErrNoRows
	}
	err := row.Scan(&value.ID, &value.IMEI)
	return value, err
}

func (r MySQLRepository) VehicleAssignmentsAt(ctx context.Context, deviceID int64, at time.Time) ([]VehicleAssignment, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,vehicle_id FROM vehicle_assignments WHERE device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC,id DESC LIMIT 2`, deviceID, at, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []VehicleAssignment{}
	for rows.Next() {
		var value VehicleAssignment
		if err = rows.Scan(&value.ID, &value.VehicleID); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r MySQLRepository) BLESensor(ctx context.Context, identity BLEIdentity) (BLESensor, error) {
	var value BLESensor
	mac, valid := NormalizeMAC(identity.MAC)
	var row *sql.Row
	switch {
	case identity.SensorID > 0 && valid:
		row = r.DB.QueryRowContext(ctx, "SELECT id,mac FROM ble_sensors WHERE id=? AND mac=?", identity.SensorID, mac)
	case identity.SensorID > 0:
		row = r.DB.QueryRowContext(ctx, "SELECT id,mac FROM ble_sensors WHERE id=? AND mac IS NOT NULL", identity.SensorID)
	case valid:
		row = r.DB.QueryRowContext(ctx, "SELECT id,mac FROM ble_sensors WHERE mac=?", mac)
	default:
		return value, sql.ErrNoRows
	}
	err := row.Scan(&value.ID, &value.MAC)
	return value, err
}

func (r MySQLRepository) SensorBindingsAt(ctx context.Context, sensorID int64, at time.Time) ([]SensorBinding, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,vehicle_id,device_id FROM ble_sensor_bindings WHERE sensor_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC,id DESC LIMIT 2`, sensorID, at, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []SensorBinding{}
	for rows.Next() {
		var value SensorBinding
		var vehicle, device sql.NullInt64
		if err = rows.Scan(&value.ID, &vehicle, &device); err != nil {
			return nil, err
		}
		if vehicle.Valid {
			value.VehicleID = pointer(vehicle.Int64)
		}
		if device.Valid {
			value.DeviceID = pointer(device.Int64)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r MySQLRepository) GPSFact(ctx context.Context, id int64) (GPSFact, error) {
	var value GPSFact
	var speed sql.NullFloat64
	var gpsTime sql.NullTime
	err := r.DB.QueryRowContext(ctx, `SELECT p.id,p.device_id,d.imei,p.latitude,p.longitude,p.speed_kmh,p.gps_time,p.received_at,p.fix_status FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id WHERE p.id=?`, id).Scan(&value.ID, &value.DeviceID, &value.IMEI, &value.Latitude, &value.Longitude, &speed, &gpsTime, &value.ReceivedAt, &value.FixStatus)
	if speed.Valid {
		value.SpeedKMH = pointer(speed.Float64)
	}
	if gpsTime.Valid {
		value.GPSTime = pointer(gpsTime.Time.UTC())
	}
	return value, err
}

func (r MySQLRepository) FuelFact(ctx context.Context, id int64) (FuelFact, error) {
	var value FuelFact
	var percent, liters, raw sql.NullFloat64
	var unit sql.NullString
	var calibration sql.NullInt64
	var sensorTime sql.NullTime
	err := r.DB.QueryRowContext(ctx, `SELECT f.id,f.telemetry_id,t.sensor_id,t.device_id,t.raw_message_id,s.mac,d.imei,f.validity,f.fuel_percent,f.fuel_liters,f.raw_measurement,f.raw_unit,f.calibration_id,t.sensor_time,f.received_at FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id JOIN device_raw_messages r ON r.id=t.raw_message_id AND r.device_id=t.device_id JOIN ble_sensors s ON s.id=t.sensor_id JOIN gps_devices d ON d.id=t.device_id WHERE f.id=?`, id).Scan(&value.ID, &value.TelemetryID, &value.SensorID, &value.DeviceID, &value.RawMessageID, &value.MAC, &value.IMEI, &value.Validity, &percent, &liters, &raw, &unit, &calibration, &sensorTime, &value.ReceivedAt)
	if percent.Valid {
		value.FuelPercent = pointer(percent.Float64)
	}
	if liters.Valid {
		value.FuelLiters = pointer(liters.Float64)
	}
	if raw.Valid {
		value.RawFuel = pointer(raw.Float64)
	}
	if unit.Valid {
		value.RawUnit = pointer(unit.String)
	}
	if calibration.Valid {
		value.CalibrationID = pointer(calibration.Int64)
	}
	if sensorTime.Valid {
		value.SensorTime = pointer(sensorTime.Time.UTC())
	}
	return value, err
}

func (r MySQLRepository) CorrelatedGPSTime(ctx context.Context, rawMessageID int64) (*time.Time, error) {
	var at sql.NullTime
	err := r.DB.QueryRowContext(ctx, `SELECT CASE WHEN COUNT(DISTINCT p.gps_time)=1 THEN MAX(p.gps_time) END FROM gps_positions p JOIN device_raw_messages r ON r.id=p.raw_message_id AND r.device_id=p.device_id WHERE p.raw_message_id=? AND p.gps_time IS NOT NULL AND p.fix_status='valid' AND p.validation_state='validated'`, rawMessageID).Scan(&at)
	if err != nil || !at.Valid {
		return nil, err
	}
	return pointer(at.Time.UTC()), nil
}

func (r MySQLRepository) Validate() error {
	if r.DB == nil {
		return fmt.Errorf("telemetry database is required")
	}
	return nil
}
