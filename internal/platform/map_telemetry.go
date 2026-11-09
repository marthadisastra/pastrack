package platform

import (
	"context"
	"fmt"
	"pastrack/internal/protocol"
	"time"
)

func (s *Server) mapIgnitions(ctx context.Context, vehicle int64, from, to time.Time) ([]map[string]any, error) {
	rows, err := queryRows(ctx, s.DB, `SELECT r.device_id,a.starts_at,a.ends_at,CONVERT(r.payload USING utf8mb4) payload FROM device_raw_messages r JOIN vehicle_assignments a ON a.device_id=r.device_id AND a.vehicle_id=? AND a.starts_at<? AND (a.ends_at IS NULL OR a.ends_at>?)
 WHERE r.received_at>=? AND SUBSTRING(r.payload,7,5) IN ('GTIGL','GTVGL','GTIGN','GTIGF','GTVGN','GTVGF')

 ORDER BY r.received_at DESC,r.id DESC LIMIT 10001`, vehicle, to, from, from.Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	if len(rows) > 10000 {
		return nil, fmt.Errorf("Riwayat ignition terlalu banyak. Persempit periode.")
	}
	out := []map[string]any{}
	for _, row := range rows {
		report, err := protocol.DecodeIgnition(stringValue(row["payload"]))
		if err != nil || report.At.After(to) {
			continue
		}
		start, end := timestamp(row["starts_at"]), timestamp(row["ends_at"])
		if !report.At.Before(start) && (end.IsZero() || report.At.Before(end)) {
			out = append(out, map[string]any{"device_id": row["device_id"], "event_at": report.At, "ignition_status": report.State, "source": report.Source})
		}
	}
	return out, nil
}
