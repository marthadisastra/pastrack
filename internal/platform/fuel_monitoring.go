package platform

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// fuelMonitoringOperations serves a read-only, vehicle-centric projection for
// the Fuel Monitoring workspace. It intentionally exposes source availability
// so the UI never turns an absent driver or ignition signal into a fact.
func (s *Server) fuelMonitoringOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "fuel/monitoring/workspace" {
		return false
	}
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return true
	}
	vehicle := int64(0)
	var err error
	if raw := r.URL.Query().Get("vehicle_id"); raw != "" {
		vehicle, err = strconv.ParseInt(raw, 10, 64)
	}
	if err != nil || vehicle < 0 {
		fail(w, http.StatusBadRequest, "invalid_vehicle")
		return true
	}
	from, err := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_period")
		return true
	}
	to, err := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err != nil || !to.After(from) || to.Sub(from) > 31*24*time.Hour {
		fail(w, http.StatusBadRequest, "period_maximum_31_days")
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	from, to = from.UTC(), to.UTC()
	result := map[string]any{
		"generated_at":               time.Now().UTC(),
		"from":                       from,
		"to":                         to,
		"timezone":                   "Asia/Jakarta",
		"vehicle_id":                 vehicle,
		"driver_dimension_available": false,
		"physical_ignition":          "NOT_INSTALLED",
		"basis":                      "fuel received_at; trips and GPS device event time",
	}
	queries := []struct {
		key   string
		query string
		args  []any
	}{
		{"fleet", `SELECT COUNT(DISTINCT v.id) vehicle_count,
		 COALESCE(SUM(CASE WHEN latest.fuel_liters IS NOT NULL AND latest.validity='valid' THEN latest.fuel_liters ELSE 0 END),0) fuel_in_tank_l,
		 AVG(CASE WHEN latest.validity='valid' THEN latest.fuel_percent END) mean_fill_percent,
		 COUNT(DISTINCT CASE WHEN bs.status='online' AND bs.last_received_at>=UTC_TIMESTAMP(3)-INTERVAL 5 MINUTE THEN fs.id END) sensors_fresh,
		 COUNT(DISTINCT fs.id) sensors_total
		 FROM vehicles v
		 LEFT JOIN fuel_tank_profiles tp ON tp.vehicle_id=v.id AND tp.effective_from<=? AND (tp.effective_to IS NULL OR tp.effective_to>?)
		 LEFT JOIN fuel_sensors fs ON fs.tank_profile_id=tp.id
		 LEFT JOIN ble_sensor_status bs ON bs.sensor_id=fs.sensor_id
		 LEFT JOIN latest_fuel_readings lfr ON lfr.fuel_sensor_id=fs.id
		 LEFT JOIN fuel_readings latest ON latest.id=lfr.last_valid_reading_id
		 WHERE v.deleted_at IS NULL AND (?=0 OR v.id=?)`, []any{to, from, vehicle, vehicle}},
		{"consumption", `SELECT COALESCE(SUM(consumed_l),0) consumed_l,COALESCE(SUM(distance_m),0) distance_m,
		 CASE WHEN SUM(consumed_l)>0 THEN SUM(distance_m)/1000/SUM(consumed_l) END efficiency_km_l,
		 AVG(coverage_percent) coverage_percent
		 FROM fuel_consumption_summaries WHERE period_start>=? AND period_end<=? AND (?=0 OR vehicle_id=?)`, []any{from, to, vehicle, vehicle}},
		{"fuel_events", `SELECT event_type,COUNT(*) event_count,COALESCE(SUM(delta_l),0) delta_l
		 FROM fuel_events WHERE event_at>=? AND event_at<? AND (?=0 OR vehicle_id=?) GROUP BY event_type ORDER BY event_type`, []any{from, to, vehicle, vehicle}},
		{"driver_events", `SELECT te.event_kind,COUNT(*) event_count FROM trip_events te JOIN trips t ON t.id=te.trip_id
		 WHERE te.event_at>=? AND te.event_at<? AND (?=0 OR t.vehicle_id=?) GROUP BY te.event_kind ORDER BY te.event_kind`, []any{from, to, vehicle, vehicle}},
		{"weekly", `SELECT FLOOR(TIMESTAMPDIFF(DAY,?,s.period_start)/7)+1 week_number,COALESCE(SUM(s.consumed_l),0) consumed_l,
		 COALESCE(SUM(s.distance_m),0) distance_m FROM fuel_consumption_summaries s
		 WHERE s.period_start>=? AND s.period_end<=? AND (?=0 OR s.vehicle_id=?) GROUP BY week_number ORDER BY week_number`, []any{from, from, to, vehicle, vehicle}},
		{"rankings", `SELECT v.id vehicle_id,v.plate_number,v.vehicle_name,COALESCE(SUM(s.consumed_l),0) consumed_l,
		 COALESCE(SUM(s.distance_m),0) distance_m,AVG(s.coverage_percent) coverage_percent,
		 CASE WHEN SUM(s.consumed_l)>0 THEN SUM(s.distance_m)/1000/SUM(s.consumed_l) END efficiency_km_l,
		 COALESCE((SELECT COUNT(*) FROM trip_events te JOIN trips tr ON tr.id=te.trip_id WHERE tr.vehicle_id=v.id AND te.event_at>=? AND te.event_at<?),0) behavior_event_count,
		 COALESCE((SELECT COUNT(*) FROM fuel_events fe WHERE fe.vehicle_id=v.id AND fe.event_type='fuel_drop' AND fe.event_at>=? AND fe.event_at<?),0) drop_event_count
		 FROM vehicles v LEFT JOIN fuel_consumption_summaries s ON s.vehicle_id=v.id AND s.period_start>=? AND s.period_end<=?
		 WHERE v.deleted_at IS NULL AND (?=0 OR v.id=?) GROUP BY v.id,v.plate_number,v.vehicle_name
		 ORDER BY CASE WHEN SUM(s.consumed_l)>0 THEN 0 ELSE 1 END,
		 SUM(s.distance_m)/NULLIF(SUM(s.consumed_l),0) DESC,v.id LIMIT 130`, []any{from, to, from, to, from, to, vehicle, vehicle}},
		{"heatmap", `SELECT p.vehicle_id,p.latitude,p.longitude,p.speed_kmh,p.gps_time FROM gps_positions p
		 WHERE p.gps_time>=? AND p.gps_time<? AND p.fix_status='valid' AND p.validation_state='validated' AND (?=0 OR p.vehicle_id=?)
		 ORDER BY p.gps_time DESC,p.id DESC LIMIT 1500`, []any{from, to, vehicle, vehicle}},
	}
	for _, q := range queries {
		rows, queryErr := queryRows(ctx, s.DB, q.query, q.args...)
		if queryErr != nil {
			databaseError(w, queryErr)
			return true
		}
		result[q.key] = rows
	}
	respond(w, http.StatusOK, map[string]any{"data": result})
	return true
}
