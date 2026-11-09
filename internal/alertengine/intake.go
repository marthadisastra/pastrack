package alertengine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"pastrack/internal/telemetry"
)

type ValidationError struct{ Code string }

func (e ValidationError) Error() string { return e.Code }

type Intake struct {
	DB          *sql.DB
	MaxAttempts int
}

func ValidateEvent(event Event) error {
	switch {
	case event.Key == "":
		return ValidationError{"EVENT_KEY_REQUIRED"}
	case event.Type == "":
		return ValidationError{"EVENT_TYPE_REQUIRED"}
	case event.VehicleID <= 0:
		return ValidationError{"VEHICLE_REQUIRED"}
	case event.ResolutionStatus != telemetry.Resolved:
		return ValidationError{"IDENTITY_NOT_RESOLVED"}
	case event.SourceType == "" || event.SourceIdentifier == "":
		return ValidationError{"SOURCE_REQUIRED"}
	case event.ReceivedAt.IsZero():
		return ValidationError{"RECEIVED_TIME_REQUIRED"}
	case event.TimestampQuality == telemetry.OriginalDeviceTime || event.TimestampQuality == telemetry.CorrelatedTime:
		if event.EventTime == nil || event.EventTime.IsZero() {
			return ValidationError{"EVENT_TIME_REQUIRED"}
		}
	case event.TimestampQuality == telemetry.ReceivedTimeOnly || event.TimestampQuality == telemetry.UnknownTime:
		if event.EventTime != nil {
			return ValidationError{"RECEIVED_TIME_PROMOTED"}
		}
	default:
		return ValidationError{"TIMESTAMP_QUALITY_INVALID"}
	}
	if len(event.Payload) == 0 || !json.Valid(event.Payload) || len(event.Payload) > 64<<10 {
		return ValidationError{"PAYLOAD_INVALID"}
	}
	return nil
}

func (i Intake) Enqueue(ctx context.Context, event Event) (Event, Job, error) {
	if i.DB == nil {
		return Event{}, Job{}, fmt.Errorf("alert intake database is required")
	}
	if err := ValidateEvent(event); err != nil {
		i.reject(ctx, event, err)
		return Event{}, Job{}, err
	}
	maxAttempts := i.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, Job{}, err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM vehicles WHERE id=?", event.VehicleID).Scan(&exists); err != nil {
		return Event{}, Job{}, err
	}
	if exists != 1 {
		err = ValidationError{"VEHICLE_NOT_FOUND"}
		_ = tx.Rollback()
		i.reject(ctx, event, err)
		return Event{}, Job{}, err
	}
	key := sha256.Sum256([]byte(event.Key))
	_, err = tx.ExecContext(ctx, `INSERT INTO alert_input_events(event_key,event_type,vehicle_id,source_type,source_identifier,device_id,sensor_id,gps_position_id,fuel_reading_id,device_event_id,sensor_telemetry_id,raw_message_id,event_time,received_at,timestamp_quality,resolution_status,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, key[:], event.Type, event.VehicleID, event.SourceType, event.SourceIdentifier, event.DeviceID, event.SensorID, event.GPSPositionID, event.FuelReadingID, event.DeviceEventID, event.SensorTelemetryID, event.RawMessageID, event.EventTime, event.ReceivedAt.UTC(), event.TimestampQuality, event.ResolutionStatus, event.Payload)
	if err != nil {
		return Event{}, Job{}, err
	}
	if err = tx.QueryRowContext(ctx, "SELECT id,created_at FROM alert_input_events WHERE event_key=?", key[:]).Scan(&event.ID, &event.CreatedAt); err != nil {
		return Event{}, Job{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO alert_evaluation_jobs(event_id,vehicle_id,job_type,state,attempts,max_attempts,available_at) VALUES(?,?,'RULE_EVALUATION','PENDING',0,?,UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, event.ID, event.VehicleID, maxAttempts)
	if err != nil {
		return Event{}, Job{}, err
	}
	var job Job
	if err = tx.QueryRowContext(ctx, "SELECT id,event_id,vehicle_id,job_type,state,attempts,max_attempts,available_at FROM alert_evaluation_jobs WHERE event_id=? AND job_type='RULE_EVALUATION'", event.ID).Scan(&job.ID, &job.EventID, &job.VehicleID, &job.Type, &job.State, &job.Attempts, &job.MaxAttempts, &job.AvailableAt); err != nil {
		return Event{}, Job{}, err
	}
	if err = tx.Commit(); err != nil {
		return Event{}, Job{}, err
	}
	return event, job, nil
}

func (i Intake) reject(ctx context.Context, event Event, cause error) {
	if i.DB == nil {
		return
	}
	code := "EVENT_REJECTED"
	var validation ValidationError
	if errors.As(cause, &validation) {
		code = validation.Code
	}
	var key any
	if event.Key != "" {
		sum := sha256.Sum256([]byte(event.Key))
		key = sum[:]
	}
	_, _ = i.DB.ExecContext(ctx, `INSERT INTO alert_intake_rejections(event_key,event_type,vehicle_id,source_type,source_id,reason_code,timestamp_quality,resolution_status,detail_redacted) VALUES(?,?,?,?,?,?,?,?,?)`, key, nullableString(event.Type), nullablePositive(event.VehicleID), nullableString(event.SourceType), sourceID(event), code, nullableString(string(event.TimestampQuality)), nullableString(string(event.ResolutionStatus)), code)
}

func sourceID(event Event) any {
	if event.GPSPositionID != nil {
		return *event.GPSPositionID
	}
	if event.FuelReadingID != nil {
		return *event.FuelReadingID
	}
	if event.DeviceEventID != nil {
		return *event.DeviceEventID
	}
	return nil
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func nullablePositive(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func loadEvent(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (Event, error) {
	var event Event
	var eventKey []byte
	var device, sensor, position, fuel, deviceEvent, telemetryID, raw sql.NullInt64
	var eventTime sql.NullTime
	var payload []byte
	err := q.QueryRowContext(ctx, `SELECT id,event_key,event_type,vehicle_id,source_type,source_identifier,device_id,sensor_id,gps_position_id,fuel_reading_id,device_event_id,sensor_telemetry_id,raw_message_id,event_time,received_at,timestamp_quality,resolution_status,payload,created_at FROM alert_input_events WHERE id=?`, id).Scan(&event.ID, &eventKey, &event.Type, &event.VehicleID, &event.SourceType, &event.SourceIdentifier, &device, &sensor, &position, &fuel, &deviceEvent, &telemetryID, &raw, &eventTime, &event.ReceivedAt, &event.TimestampQuality, &event.ResolutionStatus, &payload, &event.CreatedAt)
	if err != nil {
		return event, err
	}
	if device.Valid {
		event.DeviceID = address(device.Int64)
	}
	if sensor.Valid {
		event.SensorID = address(sensor.Int64)
	}
	if position.Valid {
		event.GPSPositionID = address(position.Int64)
	}
	if fuel.Valid {
		event.FuelReadingID = address(fuel.Int64)
	}
	if deviceEvent.Valid {
		event.DeviceEventID = address(deviceEvent.Int64)
	}
	if telemetryID.Valid {
		event.SensorTelemetryID = address(telemetryID.Int64)
	}
	if raw.Valid {
		event.RawMessageID = address(raw.Int64)
	}
	if eventTime.Valid {
		value := eventTime.Time.UTC()
		event.EventTime = &value
	}
	event.Key = fmt.Sprintf("stored:%x", eventKey)
	event.Payload = append(json.RawMessage(nil), payload...)
	return event, nil
}
