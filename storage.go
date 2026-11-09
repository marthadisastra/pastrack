package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Record struct {
	PayloadBytes []byte    `json:"payload_bytes,omitempty"`
	ReceiptID    string    `json:"receipt_id,omitempty"`
	Payload      string    `json:"payload"`
	Received     time.Time `json:"received_at"`
	Peer         string    `json:"peer"`
}

func save(ctx context.Context, db *sql.DB, r Record, loc *time.Location) error {
	if os.Getenv("PASTRACK_SCHEMA") == "v2" {
		return saveV2(ctx, db, r)
	}
	m, parseErr := parseMessage(r.Payload, loc)
	status := "partial"
	var note any = "Message retained; report layout has not been verified"
	if parseErr != nil {
		status = "failed"
		note = parseErr.Error()
	} else if m.Position != nil {
		status = "success"
		note = nil
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var device sql.NullInt64
	if m.IMEI != "" {
		var id int64
		e = tx.QueryRowContext(ctx, "SELECT id FROM gps_devices WHERE imei=? AND is_active=1 LIMIT 1", m.IMEI).Scan(&id)
		if e == nil {
			device = sql.NullInt64{Int64: id, Valid: true}
		} else if e != sql.ErrNoRows {
			return e
		}
	}
	if !device.Valid && parseErr == nil {
		status = "partial"
		note = "IMEI is not registered or inactive; raw payload retained"
	}
	parsed, e := json.Marshal(m)
	if e != nil {
		return e
	}
	result, e := tx.ExecContext(ctx, `INSERT INTO device_raw_messages (device_id,imei,message_type,protocol_version,raw_payload,parsed_payload,received_at,parse_status,parse_error) VALUES (?,?,?,?,?,?,?,?,?)`, device, bounded(m.IMEI, 20), bounded(m.Type, 30), bounded(m.Version, 30), r.Payload, string(parsed), r.Received.UTC(), status, note)
	if e != nil {
		return e
	}
	rawID, e := result.LastInsertId()
	if e != nil {
		return e
	}
	if device.Valid && m.Type == "GTERI" {
		seen := map[string]bool{}
		for i := 0; i+1 < len(m.Fields); i++ {
			if m.Fields[i] != "CFS300B" {
				continue
			}
			id := strings.ToUpper(m.Fields[i+1])
			if len(id) != 12 || strings.Trim(id, "0123456789ABCDEF") != "" || seen[id] {
				continue
			}
			seen[id] = true
			var observed any
			if value, ok := observedFuelValue(m, i); ok {
				observed = value
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO device_fuel_sensors (device_id,reported_device_id,sensor_id,last_raw_message_id,last_seen_at,observed_raw_value,observed_at) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE device_id=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(device_id),device_id),reported_device_id=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(reported_device_id),reported_device_id),last_raw_message_id=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(last_raw_message_id),last_raw_message_id),observed_raw_value=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(observed_raw_value),observed_raw_value),observed_at=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(observed_at),observed_at),last_seen_at=IF(last_seen_at IS NULL OR last_seen_at<=VALUES(last_seen_at),VALUES(last_seen_at),last_seen_at)`, device.Int64, device.Int64, id, rawID, r.Received.UTC(), observed, r.Received.UTC())
			if e != nil {
				return e
			}
		}
	}
	if device.Valid && parseErr == nil {
		_, e = tx.ExecContext(ctx, "UPDATE gps_devices SET last_seen_at=GREATEST(COALESCE(last_seen_at,?),?) WHERE id=?", r.Received.UTC(), r.Received.UTC(), device.Int64)
		if e != nil {
			return e
		}
		if p := m.Position; p != nil {
			_, e = tx.ExecContext(ctx, `INSERT INTO gps_positions (device_id,raw_message_id,gps_time,device_send_time,latitude,longitude,speed_kmh,heading_deg,altitude_m,gps_accuracy,satellites) VALUES (?,?,?,?,?,?,?,?,?,NULL,NULL)`, device.Int64, rawID, p.GPSTime, p.SendTime, p.Latitude, p.Longitude, p.Speed, p.Heading, p.Altitude)
			if e != nil {
				return e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO device_events (device_id,raw_message_id,message_type,event_code,event_value,event_time) VALUES (?,?,?,?,?,?)`, device.Int64, rawID, m.Type, m.Status, "Unverified protocol status code", p.GPSTime)
			if e != nil {
				return e
			}
		}
	}
	if e = tx.Commit(); e != nil {
		return fmt.Errorf("commit: %w", e)
	}
	return nil
}
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func bounded(s string, n int) any {
	if len(s) > n {
		return nil
	}
	return nullable(s)
}
