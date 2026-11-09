package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"pastrack/internal/alertengine"
	crashsvc "pastrack/internal/crash"
	"pastrack/internal/protocol"
	"strconv"
	"strings"
	"time"
)

type normalizationError struct{ code string }

func (e normalizationError) Error() string { return e.code }

const parserVersion = "gv30ceu-stt-v2.3"

// Battery events carry a send timestamp. Only GTBPL carries the voltage.
func gpsBatteryReport(m Message) (time.Time, *float64, error) {
	if (m.Version != "80201E0100" && m.Version != "80201E0306") ||
		(m.Model != "" && m.Model != "GV30CEU") || len(m.Fields) < 14 {
		return time.Time{}, nil, fmt.Errorf("unverified GPS battery report")
	}
	sent, err := deviceTime(m.Fields[len(m.Fields)-2], time.UTC)
	if err != nil || sent == nil {
		return time.Time{}, nil, fmt.Errorf("invalid GPS battery report time")
	}
	if m.Type == "GTBTC" {
		return *sent, nil, nil
	}
	voltage, err := strconv.ParseFloat(m.Fields[0], 64)
	if err != nil || !(voltage >= 0 && voltage <= 4.5) {
		return time.Time{}, nil, fmt.Errorf("invalid GPS backup battery voltage")
	}
	return *sent, &voltage, nil
}

// saveV2 never updates a receipt. Parse attempts and normalized facts live separately.
func saveV2(ctx context.Context, db *sql.DB, r Record) error {
	m, parseErr := parseMessage(r.Payload, time.UTC)
	var crashReport *protocol.CrashReport
	if parseErr == nil && (m.Type == "GTCRA" || m.Type == "GTECR" || m.Type == "GTCRD" || m.Type == "GTACC") {
		decoded, e := protocol.DecodeCrash(r.Payload)
		if e != nil {
			parseErr = e
		} else {
			crashReport = &decoded
		}
	}
	batteryEvent := m.Type == "GTBPL" || m.Type == "GTBTC"
	var batteryAt time.Time
	var batteryVoltage *float64
	if batteryEvent && parseErr == nil {
		batteryAt, batteryVoltage, parseErr = gpsBatteryReport(m)
	}
	var accessories []protocol.Accessory
	var bleErr error
	var eriGPSValid bool
	var eriSent time.Time
	positions := []*Position{}
	if m.Position != nil {
		positions = append(positions, m.Position)
	}
	if m.Type == "GTERI" && parseErr == nil {
		var report *protocol.ERIReport
		report, bleErr = protocol.DecodeERI(strings.Split(strings.TrimSuffix(strings.TrimSpace(r.Payload), "$"), ","))
		if bleErr == nil {
			accessories = report.Accessories
		} else {
			report, _ = protocol.DecodeERIPositions(strings.Split(strings.TrimSuffix(strings.TrimSpace(r.Payload), "$"), ","))
		}
		if report != nil {
			eriGPSValid = true
			eriSent = report.SendTime
			for _, p := range report.Positions {
				send := report.SendTime
				positions = append(positions, &Position{Speed: p.Speed, Heading: p.Heading, Altitude: p.Altitude, Latitude: p.Latitude, Longitude: p.Longitude, GPSTime: p.GPSTime, SendTime: &send, Satellites: p.Satellites, HDOP: p.HDOP, FixStatus: p.FixStatus, AccuracyRaw: p.Accuracy})
			}
		}
	}
	sum := sha256.Sum256([]byte(r.Payload))
	normalizedPayload := strings.TrimSpace(r.Payload)
	if strings.HasPrefix(normalizedPayload, "+BUFF:") {
		normalizedPayload = "+RESP:" + strings.TrimPrefix(normalizedPayload, "+BUFF:")
	}
	processingKey := sha256.Sum256([]byte(normalizedPayload))
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var device sql.NullInt64
	var registeredModel sql.NullString
	if len(m.IMEI) == 15 {
		err = tx.QueryRowContext(ctx, "SELECT d.id,m.model FROM gps_devices d LEFT JOIN gps_device_models m ON m.id=d.model_id WHERE d.imei=? AND d.is_active=1 AND d.deleted_at IS NULL", m.IMEI).Scan(&device, &registeredModel)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	if device.Valid && registeredModel.String != "GV30CEU" && (batteryEvent || crashReport != nil || (m.Type == "GTSTT" && m.Model == "")) {
		parseErr = fmt.Errorf("report requires registered GV30CEU model")
	}
	var messageType sql.NullInt64
	if m.Type != "" && len(m.Type) <= 32 {
		_, err = tx.ExecContext(ctx, "INSERT INTO protocol_message_types(family,code,supported) VALUES('queclink',?,?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)", m.Type, m.Type == "GTSTT" || m.Type == "GTERI" || batteryEvent || crashReport != nil)
		if err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, "SELECT id FROM protocol_message_types WHERE family='queclink' AND code=?", m.Type).Scan(&messageType)
		if err != nil {
			return err
		}
	}
	var rawID int64
	var result sql.Result
	if r.ReceiptID != "" {
		var storedHash []byte
		err = tx.QueryRowContext(ctx, "SELECT id,payload_sha256 FROM device_raw_messages WHERE spool_receipt_id=?", r.ReceiptID).Scan(&rawID, &storedHash)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && string(storedHash) != string(sum[:]) {
			return fmt.Errorf("receipt identity collision")
		}
	}
	if rawID == 0 {
		result, err = tx.ExecContext(ctx, `INSERT INTO device_raw_messages(device_id,message_type_id,protocol_version,payload,payload_sha256,received_at,spool_receipt_id) VALUES(?,?,?,?,?,?,?)`, device, messageType, bounded(m.Version, 32), []byte(r.Payload), sum[:], r.Received.UTC(), nullable(r.ReceiptID))
		if err != nil {
			return err
		}
		rawID, err = result.LastInsertId()
		if err != nil {
			return err
		}
	}
	// Receipt durability is independent from normalized persistence. If a fact
	// fails constraints, the immutable receipt remains queryable and replayable.
	if err = tx.Commit(); err != nil {
		return err
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existing int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM protocol_parse_logs WHERE raw_message_id=? AND parser_version=?", rawID, parserVersion).Scan(&existing)
	if err == nil {
		return tx.Commit()
	}
	if err != sql.ErrNoRows {
		return err
	}
	status := "unsupported"
	processing := "retained"
	if parseErr != nil {
		status = "failed"
	} else if batteryEvent || crashReport != nil || m.Type == "GTSTT" || (m.Type == "GTERI" && (bleErr == nil || eriGPSValid)) {
		status = "success"
		processing = "processed"
	}
	fields, _ := json.Marshal(m.Fields)
	result, err = tx.ExecContext(ctx, `INSERT INTO protocol_parse_logs(raw_message_id,parser_version,parse_status,processing_status,fields,processed_at) VALUES(?,?,?,?,?,?)`, rawID, parserVersion, status, processing, string(fields), time.Now().UTC())
	if err != nil {
		return err
	}
	parseID, _ := result.LastInsertId()
	if parseErr != nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO protocol_parser_errors(parse_log_id,code,detail_redacted) VALUES(?,'invalid_message',?)`, parseID, parseErr.Error())
		if err != nil {
			return err
		}
	}
	if device.Valid {
		_, err = tx.ExecContext(ctx, `INSERT INTO gps_device_status(device_id,last_received_at,connectivity,last_raw_message_id) VALUES(?,?,'observed',?) ON DUPLICATE KEY UPDATE last_raw_message_id=IF(last_received_at IS NULL OR last_received_at<=VALUES(last_received_at),VALUES(last_raw_message_id),last_raw_message_id),last_received_at=GREATEST(COALESCE(last_received_at,VALUES(last_received_at)),VALUES(last_received_at))`, device, r.Received.UTC(), rawID)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(entityChange{Kind: "telemetry", IMEI: m.IMEI})
		if _, err = tx.ExecContext(ctx, `INSERT INTO dashboard_change_outbox(raw_message_id,event_kind,aggregate_id,payload) VALUES(?,'telemetry',?,?) ON DUPLICATE KEY UPDATE raw_message_id=VALUES(raw_message_id)`, rawID, device.Int64, payload); err != nil {
			return err
		}
	}
	if !device.Valid || parseErr != nil || processing != "processed" {
		return tx.Commit()
	}
	// Equal wire payloads keep distinct receipts, but only one set of derived facts.
	result, err = tx.ExecContext(ctx, `INSERT IGNORE INTO message_processing_keys(device_id,message_key,canonical_raw_id,parser_version) VALUES(?,?,?,?)`, device, processingKey[:], rawID, parserVersion)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		if _, err = tx.ExecContext(ctx, "UPDATE protocol_parse_logs SET processing_status='duplicate' WHERE id=?", parseID); err != nil {
			return err
		}
		return tx.Commit()
	}
	if crashReport != nil {
		if err = tx.Commit(); err != nil {
			return err
		}
		normalized, e := (crashsvc.Service{DB: db}).Process(ctx, rawID, device.Int64, r.Received.UTC(), *crashReport)
		if e != nil {
			return e
		}
		if normalized.ResolutionStatus == "RESOLVED" && normalized.SignalEventID > 0 {
			_, _, e = (alertengine.Intake{DB: db}).Enqueue(ctx, alertengine.EventFromCrash(normalized))
		}
		return e
	}
	if batteryEvent {
		decoded, _ := json.Marshal(map[string]any{"backup_battery_voltage": batteryVoltage})
		if _, err = tx.ExecContext(ctx, "INSERT INTO device_status_logs(device_id,raw_message_id,event_at,status_code_raw,decoded_status) VALUES(?,?,?,?,?)", device, rawID, batteryAt, m.Type, string(decoded)); err != nil {
			return err
		}
		return tx.Commit()
	}
	if m.Type == "GTSTT" {
		var satellites any
		if m.Position != nil {
			satellites = m.Position.Satellites
		}
		fix := "no_fix"
		var hdop any
		if m.Position != nil {
			fix = m.Position.FixStatus
			hdop = m.Position.HDOP
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO gps_quality_logs(device_id,raw_message_id,satellite_count,fix_status,event_at,hdop) VALUES(?,?,?,?,?,?)", device, rawID, satellites, fix, eventTime(m, r), hdop); err != nil {
			return err
		}
		// These fixed network fields precede the verified STT append mask.
		if len(m.Fields) >= 12 && (m.Fields[8] != "" || m.Fields[9] != "" || m.Fields[10] != "" || m.Fields[11] != "") {
			if _, err = tx.ExecContext(ctx, "INSERT INTO network_status_logs(device_id,raw_message_id,event_at,mcc,mnc,lac,cell_id) VALUES(?,?,?,?,?,?,?)", device, rawID, eventTime(m, r), nullable(m.Fields[8]), nullable(m.Fields[9]), nullable(m.Fields[10]), nullable(m.Fields[11])); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO device_status_logs(device_id,raw_message_id,event_at,status_code_raw) VALUES(?,?,?,?)`, device, rawID, eventTime(m, r), nullable(m.Status))
		if err != nil {
			return err
		}
	}
	if m.Type == "GTERI" && bleErr == nil {
		if err = persistBLE(ctx, tx, device.Int64, rawID, r.Received.UTC(), eriSent, accessories); err != nil {
			return err
		}
	}
	for sampleIndex, p := range positions {
		var vehicle, assignment sql.NullInt64
		if p.GPSTime != nil {
			// No current association is substituted for missing assignment history.
			rows, e := tx.QueryContext(ctx, `SELECT id,vehicle_id FROM vehicle_assignments WHERE device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC LIMIT 2`, device, p.GPSTime, p.GPSTime)
			if e != nil {
				return e
			}
			count := 0
			for rows.Next() {
				if e = rows.Scan(&assignment, &vehicle); e != nil {
					rows.Close()
					return e
				}
				count++
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return e
			}
			if count > 1 {
				return normalizationError{code: "overlapping_device_assignment"}
			}
		}
		fix := p.FixStatus
		result, err = tx.ExecContext(ctx, `INSERT INTO gps_positions(device_id,vehicle_id,assignment_id,raw_message_id,sample_index,gps_time,received_at,device_send_time,latitude,longitude,speed_kmh,heading_deg,altitude_m,satellite_count,gnss_accuracy_raw,fix_status,validation_state,hdop) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, device, vehicle, assignment, rawID, sampleIndex, p.GPSTime, r.Received.UTC(), p.SendTime, p.Latitude, p.Longitude, p.Speed, p.Heading, p.Altitude, p.Satellites, nullable(p.AccuracyRaw), fix, "validated", p.HDOP)
		if err != nil {
			return err
		}
		positionID, _ := result.LastInsertId()
		if m.Type == "GTERI" {
			if _, err = tx.ExecContext(ctx, "INSERT INTO gps_quality_logs(device_id,position_id,raw_message_id,satellite_count,hdop,fix_status,event_at) VALUES(?,?,?,?,?,?,?)", device, positionID, rawID, p.Satellites, p.HDOP, p.FixStatus, p.GPSTime); err != nil {
				return err
			}
		}
		if vehicle.Valid && fix == "valid" {
			if err = processGeofences(ctx, tx, vehicle.Int64, positionID, p); err != nil {
				return err
			}
			// Lock the vehicle serializes latest-row initialization and out-of-order updates.
			var lock int64
			if err = tx.QueryRowContext(ctx, "SELECT id FROM vehicles WHERE id=? FOR UPDATE", vehicle).Scan(&lock); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO latest_positions(vehicle_id,position_id,gps_time,received_at) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE position_id=IF(VALUES(gps_time)>gps_time OR (VALUES(gps_time)=gps_time AND VALUES(received_at)>=received_at),VALUES(position_id),position_id),received_at=IF(VALUES(gps_time)>gps_time OR (VALUES(gps_time)=gps_time AND VALUES(received_at)>=received_at),VALUES(received_at),received_at),gps_time=GREATEST(gps_time,VALUES(gps_time))`, vehicle, positionID, p.GPSTime, r.Received.UTC())
			if err != nil {
				return err
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if m.Type == "GTERI" && bleErr == nil {
		if err = enqueueFuelAlerts(ctx, db, rawID); err != nil {
			return err
		}
	}
	return nil
}
func eventTime(m Message, r Record) any {
	if m.Position != nil {
		return m.Position.GPSTime
	}
	return nil
}
