package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"strings"
	"time"
)

type entityChange struct {
	Kind     string `json:"kind"`
	IMEI     string `json:"imei"`
	SensorID string `json:"sensor_id,omitempty"`
}

var realtimeChanges = make(chan []byte, 2048)
var spoolReady = make(chan struct{}, 1)

func queueRealtime(payload []byte) {
	select {
	case realtimeChanges <- payload:
	default:
	}
}
func signalRealtime() { queueRealtime([]byte("positions")) }
func signalFrameRealtime(payload string, loc *time.Location) {
	m, err := parseMessage(payload, loc)
	if err != nil || m.IMEI == "" {
		return
	}
	kind := "gps"
	if m.Position == nil {
		kind = "report"
	}
	if b, e := json.Marshal(entityChange{Kind: kind, IMEI: m.IMEI}); e == nil {
		queueRealtime(b)
	}
	if m.Type != "GTERI" {
		return
	}
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
		if b, e := json.Marshal(entityChange{Kind: "fuel", IMEI: m.IMEI, SensorID: id}); e == nil {
			queueRealtime(b)
		}
	}
}
func sendRealtime(path string, payload []byte) error {
	conn, e := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: path, Net: "unixgram"})
	if e != nil {
		return e
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, e = conn.Write(payload)
	return e
}
func drainDashboardOutbox(ctx context.Context, db *sql.DB, path string) (bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT id,payload FROM dashboard_change_outbox WHERE delivered_at IS NULL ORDER BY id LIMIT 256")
	if err != nil {
		return false, err
	}
	type event struct {
		id      int64
		payload []byte
	}
	events := make([]event, 0, 256)
	for rows.Next() {
		var value event
		if err = rows.Scan(&value.id, &value.payload); err != nil {
			rows.Close()
			return false, err
		}
		events = append(events, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(events) == 0 {
		if err == nil {
			_, _ = db.ExecContext(ctx, "DELETE FROM dashboard_change_outbox WHERE delivered_at<UTC_TIMESTAMP(3)-INTERVAL 7 DAY LIMIT 1000")
		}
		return false, err
	}
	for _, value := range events {
		if err = sendRealtime(path, value.payload); err != nil {
			_, _ = db.ExecContext(ctx, "UPDATE dashboard_change_outbox SET attempts=attempts+1 WHERE id=? AND delivered_at IS NULL", value.id)
			return true, err
		}
		if _, err = db.ExecContext(ctx, "UPDATE dashboard_change_outbox SET delivered_at=UTC_TIMESTAMP(3),attempts=attempts+1 WHERE id=? AND delivered_at IS NULL", value.id); err != nil {
			return true, err
		}
	}
	return true, nil
}

func publishChanges(ctx context.Context, db *sql.DB, path string) {
	if path == "off" {
		return
	}
	signalRealtime()
	for {
		var payload []byte
		select {
		case <-ctx.Done():
			return
		case payload = <-realtimeChanges:
		}
		delay := time.Second
		for {
			delivered := false
			var err error
			if db != nil {
				for {
					var batch bool
					batch, err = drainDashboardOutbox(ctx, db, path)
					if err != nil || !batch {
						break
					}
					delivered = true
				}
			}
			if err == nil && !delivered {
				err = sendRealtime(path, payload)
			}
			if err == nil {
				break
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if delay < 30*time.Second {
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
		}
	}
}
