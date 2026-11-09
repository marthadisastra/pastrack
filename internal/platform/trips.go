package platform

import (
	"net/http"
	"pastrack/internal/trip"
	"time"
)

func (s *Server) tripOperations(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	if path == "trips/history" {
		s.tripHistory(w, r)
		return true
	}
	if path == "trips/summary" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		rows, e := queryRows(r.Context(), s.DB, "SELECT COUNT(*) trip_count,SUM(distance_m) distance_m,SUM(driving_seconds) driving_seconds,SUM(stopped_seconds) stopped_seconds,MAX(max_speed_kmh) max_speed_kmh FROM trips")
		if e != nil {
			databaseError(w, e)
			return true
		}
		respond(w, 200, map[string]any{"data": rows[0]})
		return true
	}
	if path != "trips/recalculate" {
		return false
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if u.Role == "viewer" {
		fail(w, 403, "permission_denied")
		return true
	}
	var input struct {
		VehicleID       int64     `json:"vehicle_id"`
		From            time.Time `json:"from"`
		To              time.Time `json:"to"`
		MinimumSpeed    float64   `json:"minimum_speed_kmh"`
		MaximumSpeed    float64   `json:"maximum_speed_kmh"`
		MinimumDistance float64   `json:"minimum_distance_m"`
		MinimumSeconds  int       `json:"minimum_duration_seconds"`
		StopSeconds     int       `json:"stop_duration_seconds"`
		GapSeconds      int       `json:"maximum_gap_seconds"`
	}
	if body(w, r, &input) != nil || input.VehicleID <= 0 || input.From.IsZero() || !input.To.After(input.From) || input.To.Sub(input.From) > 31*24*time.Hour {
		fail(w, 400, "invalid_trip_range")
		return true
	}
	if input.MinimumSpeed == 0 {
		input.MinimumSpeed = 3
	}
	if input.MaximumSpeed == 0 {
		input.MaximumSpeed = 160
	}
	if input.MinimumDistance == 0 {
		input.MinimumDistance = 100
	}
	if input.MinimumSeconds == 0 {
		input.MinimumSeconds = 60
	}
	if input.StopSeconds == 0 {
		input.StopSeconds = 180
	}
	if input.GapSeconds == 0 {
		input.GapSeconds = 300
	}
	if input.MinimumSpeed <= 0 || input.MaximumSpeed <= input.MinimumSpeed || input.MaximumSpeed > 300 || input.MinimumDistance < 0 || input.MinimumSeconds < 1 || input.StopSeconds < 1 || input.GapSeconds < 1 {
		fail(w, 400, "invalid_trip_configuration")
		return true
	}
	rows, e := s.DB.QueryContext(r.Context(), "SELECT id,gps_time,latitude,longitude,speed_kmh FROM gps_positions WHERE vehicle_id=? AND gps_time>=? AND gps_time<=? AND fix_status='valid' AND validation_state='validated' AND speed_kmh IS NOT NULL ORDER BY gps_time,id LIMIT 10001", input.VehicleID, input.From.UTC(), input.To.UTC())
	if e != nil {
		databaseError(w, e)
		return true
	}
	points := []trip.Position{}
	for rows.Next() {
		var p trip.Position
		if e = rows.Scan(&p.ID, &p.At, &p.Lat, &p.Lon, &p.Speed); e != nil {
			rows.Close()
			databaseError(w, e)
			return true
		}
		p.Valid = true
		points = append(points, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		databaseError(w, e)
		return true
	}
	if len(points) > 10000 {
		fail(w, 422, "trip_range_too_large")
		return true
	}
	config := trip.Config{MinimumSpeed: input.MinimumSpeed, MaximumSpeed: input.MaximumSpeed, MinimumDistanceMeters: input.MinimumDistance, MinimumDuration: time.Duration(input.MinimumSeconds) * time.Second, StopDuration: time.Duration(input.StopSeconds) * time.Second, MaximumGap: time.Duration(input.GapSeconds) * time.Second}
	trips := trip.Build(points, config)
	// Configuration is part of the computation identity, so a rule change never
	// silently overwrites a historical result from a different set of thresholds.
	version := tripVersion(config)
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return true
	}
	defer tx.Rollback()
	for _, t := range trips {
		_, e = tx.ExecContext(r.Context(), `INSERT INTO trips(vehicle_id,start_position_id,end_position_id,started_at,ended_at,distance_m,driving_seconds,stopped_seconds,max_speed_kmh,average_speed_kmh,calculation_version,confidence) VALUES(?,?,?,?,?,?,?,?,?,?,?,'gps_validated') ON DUPLICATE KEY UPDATE id=id`, input.VehicleID, t.StartID, t.EndID, t.StartedAt, t.EndedAt, t.DistanceMeters, int(t.DrivingSeconds), int(t.StoppedSeconds), t.MaxSpeed, t.AverageSpeed, version)
		if e != nil {
			databaseError(w, e)
			return true
		}
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return true
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"calculated": len(trips), "version": version, "positions_examined": len(points)}})
	return true
}
