package crash

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"pastrack/internal/protocol"
	"pastrack/internal/telemetry"
)

type Service struct {
	DB                *sql.DB
	CorrelationWindow time.Duration
}

func (s Service) Process(ctx context.Context, rawID, deviceID int64, received time.Time, report protocol.CrashReport) (telemetry.NormalizedCrash, error) {
	if s.DB == nil {
		return telemetry.NormalizedCrash{}, fmt.Errorf("crash database is required")
	}
	if rawID <= 0 || deviceID <= 0 {
		return telemetry.NormalizedCrash{}, fmt.Errorf("crash source identity is required")
	}
	window := s.CorrelationWindow
	if window <= 0 {
		window = 10 * time.Minute
	}
	at := report.SendTime.UTC()
	if report.GPSTime != nil && report.Accuracy != nil && *report.Accuracy > 0 {
		at = report.GPSTime.UTC()
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	defer tx.Rollback()
	var lock int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM gps_devices WHERE id=? FOR UPDATE", deviceID).Scan(&lock); err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	resolver := telemetry.Resolver{Repository: telemetry.MySQLRepository{DB: tx}}
	resolution, err := resolver.ResolveGPS(ctx, telemetry.GPSIdentity{DeviceID: deviceID, IMEI: report.IMEI}, &at)
	if err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	if resolution.Status != telemetry.Resolved {
		return telemetry.NormalizedCrash{RawMessageID: rawID, DeviceID: deviceID, IMEI: report.IMEI, MessageType: report.MessageType, EventTime: &at, ReceivedAt: received.UTC(), TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: resolution.Status, ResolutionReason: resolution.Reason, ProtocolVersion: report.ProtocolVersion}, tx.Commit()
	}
	var firmware sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT firmware_version FROM gps_devices WHERE id=?", deviceID).Scan(&firmware); err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	var existingID int64
	var existingSet sql.NullInt64
	var existingKey, evidence sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT c.id,c.evidence_set_id,s.sequence_key,s.evidence_status FROM crash_signal_events c LEFT JOIN crash_evidence_sets s ON s.id=c.evidence_set_id WHERE c.raw_message_id=?`, rawID).Scan(&existingID, &existingSet, &existingKey, &evidence)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return telemetry.NormalizedCrash{}, commitErr
		}
		return normalized(report, rawID, deviceID, existingID, resolution, received, at, existingKey.String, evidence.String, firmware.String, nil), nil
	}
	if err != sql.ErrNoRows {
		return telemetry.NormalizedCrash{}, err
	}
	var setID sql.NullInt64
	sequenceKey := ""
	evidenceStatus := "UNAVAILABLE"
	var counter *uint8
	if report.MessageType != "GTACC" {
		counter = &report.CrashCounter
		setID, sequenceKey, err = s.correlate(ctx, tx, deviceID, *resolution.VehicleID, *resolution.AssignmentID, report, at, window)
		if err != nil {
			return telemetry.NormalizedCrash{}, err
		}
		evidenceStatus = "PENDING"
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO crash_signal_events(evidence_set_id,raw_message_id,device_id,vehicle_id,assignment_id,message_type,crash_counter,data_type,total_frames,frame_number,acceleration_hex,trigger_degrees,engine_status,event_time,received_at,timestamp_quality,latitude,longitude,speed_kmh,firmware_version,protocol_version) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, nullableInt(setID), rawID, deviceID, *resolution.VehicleID, *resolution.AssignmentID, report.MessageType, counter, report.DataType, report.TotalFrames, report.FrameNumber, nullableText(report.AccelerationHex), report.TriggerDegrees, report.EngineStatus, at, received.UTC(), telemetry.OriginalDeviceTime, report.Latitude, report.Longitude, report.Speed, nullableText(firmware.String), report.ProtocolVersion)
	if err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	signalID, err := result.LastInsertId()
	if err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	if setID.Valid {
		if err = s.refreshEvidence(ctx, tx, setID.Int64, report); err != nil {
			return telemetry.NormalizedCrash{}, err
		}
		if err = tx.QueryRowContext(ctx, "SELECT evidence_status FROM crash_evidence_sets WHERE id=?", setID.Int64).Scan(&evidenceStatus); err != nil {
			return telemetry.NormalizedCrash{}, err
		}
	}
	var related []int64
	if setID.Valid && (report.MessageType == "GTCRA" || report.MessageType == "GTECR") {
		rows, e := tx.QueryContext(ctx, `SELECT a.id FROM crash_signal_events c JOIN alert_input_events a ON a.raw_message_id=c.raw_message_id WHERE c.evidence_set_id=? ORDER BY a.id`, setID.Int64)
		if e != nil {
			return telemetry.NormalizedCrash{}, e
		}
		for rows.Next() {
			var id int64
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return telemetry.NormalizedCrash{}, e
			}
			related = append(related, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return telemetry.NormalizedCrash{}, e
		}
	}
	if err = tx.Commit(); err != nil {
		return telemetry.NormalizedCrash{}, err
	}
	return normalized(report, rawID, deviceID, signalID, resolution, received, at, sequenceKey, evidenceStatus, firmware.String, related), nil
}

func (s Service) correlate(ctx context.Context, tx *sql.Tx, device, vehicle, assignment int64, r protocol.CrashReport, at time.Time, window time.Duration) (sql.NullInt64, string, error) {
	primary := r.MessageType == "GTCRA" || r.MessageType == "GTECR"
	query := `SELECT id,sequence_key,primary_message_type FROM crash_evidence_sets WHERE device_id=? AND crash_counter=? AND anchor_at BETWEEN ? AND ? ORDER BY ABS(TIMESTAMPDIFF(MICROSECOND,anchor_at,?)),id LIMIT 1 FOR UPDATE`
	var id int64
	var key string
	var primaryType sql.NullString
	err := tx.QueryRowContext(ctx, query, device, r.CrashCounter, at.Add(-window), at.Add(window), at).Scan(&id, &key, &primaryType)
	reuse := err == nil && (!primary || !primaryType.Valid || primaryType.String != r.MessageType)
	if err != nil && err != sql.ErrNoRows {
		return sql.NullInt64{}, "", err
	}
	if reuse {
		if primary && !primaryType.Valid {
			_, err = tx.ExecContext(ctx, "UPDATE crash_evidence_sets SET primary_message_type=?,anchor_at=?,row_version=row_version+1 WHERE id=?", r.MessageType, at, id)
		}
		return sql.NullInt64{Int64: id, Valid: true}, key, err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%02X:%d:%s", device, r.CrashCounter, at.UnixNano(), r.MessageType)))
	key = hex.EncodeToString(sum[:])
	var primaryValue any
	if primary {
		primaryValue = r.MessageType
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO crash_evidence_sets(sequence_key,device_id,vehicle_id,assignment_id,crash_counter,anchor_at,primary_message_type,evidence_status) VALUES(?,?,?,?,?,?,?,'PENDING')`, key, device, vehicle, assignment, r.CrashCounter, at, primaryValue)
	if err != nil {
		return sql.NullInt64{}, "", err
	}
	id, err = result.LastInsertId()
	return sql.NullInt64{Int64: id, Valid: err == nil}, key, err
}
func (s Service) refreshEvidence(ctx context.Context, tx *sql.Tx, setID int64, r protocol.CrashReport) error {
	if r.MessageType != "GTCRD" {
		return nil
	}
	var frames, phases, incomplete int
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(frame_count),0),COUNT(DISTINCT (data_type & 1)),COALESCE(SUM(frame_count<total_count),0) FROM (SELECT data_type,COUNT(DISTINCT frame_number) frame_count,MAX(total_frames) total_count FROM crash_signal_events WHERE evidence_set_id=? AND message_type='GTCRD' GROUP BY data_type) x`, setID).Scan(&frames, &phases, &incomplete)
	if err != nil {
		return err
	}
	status := "PARTIAL"
	if phases >= 2 && incomplete == 0 {
		status = "COMPLETE"
	}
	_, err = tx.ExecContext(ctx, `UPDATE crash_evidence_sets SET expected_frames=GREATEST(COALESCE(expected_frames,0),?),received_frames=?,evidence_status=?,row_version=row_version+1 WHERE id=?`, *r.TotalFrames, frames, status, setID)
	return err
}

func normalized(r protocol.CrashReport, rawID, deviceID, signalID int64, res telemetry.Resolution, received, at time.Time, key, status, firmware string, related []int64) telemetry.NormalizedCrash {
	var counter *uint8
	if r.MessageType != "GTACC" {
		value := r.CrashCounter
		counter = &value
	}
	return telemetry.NormalizedCrash{Kind: "CRASH_SIGNAL", SignalEventID: signalID, RawMessageID: rawID, DeviceID: deviceID, IMEI: r.IMEI, VehicleID: res.VehicleID, AssignmentID: res.AssignmentID, MessageType: r.MessageType, SequenceKey: key, CrashCounter: counter, DataType: r.DataType, TotalFrames: r.TotalFrames, FrameNumber: r.FrameNumber, TriggerDegrees: r.TriggerDegrees, EngineStatus: r.EngineStatus, Latitude: r.Latitude, Longitude: r.Longitude, SpeedKMH: r.Speed, EventTime: &at, ReceivedAt: received.UTC(), TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: res.Status, ResolutionReason: res.Reason, EvidenceStatus: status, FirmwareVersion: firmware, ProtocolVersion: r.ProtocolVersion, RelatedInputEventIDs: related}
}
func nullableInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
func nullableText(v string) any {
	if v == "" {
		return nil
	}
	return v
}
