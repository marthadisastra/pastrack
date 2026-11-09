package platform

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"pastrack/internal/trip"
	"strconv"
	"time"
)

func historyNumber(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	n, e := strconv.ParseFloat(fmt.Sprint(v), 64)
	return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
}

// Derive view-only journey segments; an end-of-range segment is explicitly incomplete.
func historyJourneys(rows []map[string]any) []map[string]any {
	out := []map[string]any{}
	start, last := -1, -1
	var prev *trip.Position
	stopped := time.Time{}
	distance, driving, stopping, maxSpeed := 0.0, 0.0, 0.0, 0.0
	finish := func(reason string) {
		if start >= 0 && last > start && distance >= 100 {
			out = append(out, map[string]any{"id": len(out) + 1, "start_index": start, "end_index": last, "started_at": rows[start]["gps_time"], "ended_at": rows[last]["gps_time"], "distance_m": distance, "driving_seconds": driving, "stopped_seconds": stopping, "max_speed_kmh": maxSpeed, "average_speed_kmh": distance / math.Max(driving, 1) * 3.6, "end_reason": reason})
		}
		start = -1
		last = -1
		prev = nil
		stopped = time.Time{}
		distance = 0
		driving = 0
		stopping = 0
		maxSpeed = 0
	}
	for i, r := range rows {
		lat, a := historyNumber(r["latitude"])
		lon, b := historyNumber(r["longitude"])
		speed, c := historyNumber(r["speed_kmh"])
		at, e := time.Parse(time.RFC3339Nano, fmt.Sprint(r["gps_time"]))
		if !a || !b || !c || e != nil || speed < 0 || speed > 160 || math.Abs(lat) > 90 || math.Abs(lon) > 180 || (lat == 0 && lon == 0) {
			finish("invalid_sample")
			continue
		}
		p := trip.Position{At: at, Lat: lat, Lon: lon, Speed: speed, Valid: true}
		if prev != nil {
			dt := at.Sub(prev.At).Seconds()
			if dt <= 0 {
				continue
			}
			if dt > 300 || trip.Distance(*prev, p)/dt*3.6 > 160 {
				finish("data_gap")
			}
		}
		if start < 0 {
			if speed < 3 {
				prev = &p
				continue
			}
			start = i
			last = i
			maxSpeed = speed
			prev = &p
			continue
		}
		if prev != nil {
			dt := at.Sub(prev.At).Seconds()
			if speed >= 3 || prev.Speed >= 3 {
				distance += trip.Distance(*prev, p)
				driving += dt
			} else {
				stopping += dt
			}
		}
		last = i
		maxSpeed = math.Max(maxSpeed, speed)
		if speed < 3 {
			if stopped.IsZero() {
				stopped = at
			}
			if at.Sub(stopped) >= 3*time.Minute {
				finish("stopped")
				continue
			}
		} else {
			stopped = time.Time{}
		}
		prev = &p
	}
	finish("period_end")
	return out
}
func (s *Server) tripHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed")
		return
	}
	vehicle, e := strconv.ParseInt(r.URL.Query().Get("vehicle_id"), 10, 64)
	if e != nil || vehicle < 1 {
		fail(w, 400, "select_vehicle")
		return
	}
	from, e := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if e != nil {
		fail(w, 400, "invalid_date_range")
		return
	}
	to, e := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if e != nil || !to.After(from) || to.Sub(from) > 31*24*time.Hour {
		fail(w, 400, "period_maximum_31_days")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	vehicles, e := queryRows(ctx, s.DB, `SELECT v.id,v.plate_number,v.vehicle_name,(SELECT MAX(gps_time) FROM gps_positions p WHERE p.vehicle_id=v.id AND p.fix_status='valid' AND p.validation_state='validated') latest_gps_at FROM vehicles v WHERE v.id=? AND v.deleted_at IS NULL`, vehicle)
	if e != nil {
		databaseError(w, e)
		return
	}
	if len(vehicles) == 0 {
		fail(w, 404, "vehicle_not_found")
		return
	}
	// Choose the device with most valid fixes in this period; do not stitch different trackers.
	points, e := queryRows(ctx, s.DB, `SELECT id,device_id,latitude,longitude,speed_kmh,gps_time,fix_status,validation_state FROM gps_positions WHERE vehicle_id=? AND gps_time>=? AND gps_time<=? AND fix_status='valid' AND validation_state='validated' AND NOT(latitude=0 AND longitude=0) ORDER BY gps_time,id LIMIT 20001`, vehicle, from.UTC(), to.UTC())
	if e != nil {
		databaseError(w, e)
		return
	}
	if len(points) > 20000 {
		fail(w, 422, "too_many_records_reduce_period")
		return
	}
	counts := map[string]int{}
	device := ""
	for _, p := range points {
		key := fmt.Sprint(p["device_id"])
		counts[key]++
		if counts[key] > counts[device] {
			device = key
		}
	}
	primary := []map[string]any{}
	last := ""
	for _, p := range points {
		stamp := fmt.Sprint(p["gps_time"])
		if fmt.Sprint(p["device_id"]) == device && stamp != last {
			primary = append(primary, p)
			last = stamp
		}
	}
	readings, e := queryRows(ctx, s.DB, `SELECT f.fuel_sensor_id,f.received_at,f.fuel_liters,f.fuel_percent,f.validity,f.calibration_id,c.media,vi.tank_capacity_l capacity_l,st.fuel_height_raw FROM fuel_readings f LEFT JOIN fuel_calibrations c ON c.id=f.calibration_id LEFT JOIN vehicle_inventory vi ON vi.vehicle_id=f.vehicle_id LEFT JOIN sensor_telemetry st ON st.id=f.telemetry_id WHERE f.vehicle_id=? AND f.received_at>=? AND f.received_at<? ORDER BY f.received_at,f.id LIMIT 20001`, vehicle, from.UTC(), to.UTC())
	if e != nil {
		databaseError(w, e)
		return
	}
	if len(readings) > 20000 {
		fail(w, 422, "too_many_records_reduce_period")
		return
	}
	events, e := queryRows(ctx, s.DB, `SELECT e.id,e.event_type,e.event_at,e.delta_l,p.latitude,p.longitude FROM fuel_events e LEFT JOIN gps_positions p ON p.id=(SELECT pp.id FROM gps_positions pp WHERE pp.vehicle_id=e.vehicle_id AND pp.gps_time<=e.event_at AND pp.gps_time>=e.event_at-INTERVAL 5 MINUTE AND pp.fix_status='valid' ORDER BY pp.gps_time DESC,pp.id DESC LIMIT 1) WHERE e.vehicle_id=? AND e.event_at>=? AND e.event_at<? ORDER BY e.event_at LIMIT 1000`, vehicle, from.UTC(), to.UTC())
	if e != nil {
		databaseError(w, e)
		return
	}
	ignitions, e := s.mapIgnitions(ctx, vehicle, from.UTC(), to.UTC())
	if e != nil {
		databaseError(w, e)
		return
	}
	respond(w, 200, map[string]any{"data": map[string]any{"readings": readings, "events": events, "ignitions": ignitions, "positions": primary, "trips": historyJourneys(primary), "vehicle": vehicles[0], "from": from.UTC(), "to": to.UTC(), "generated_at": time.Now().UTC(), "device_id": device, "devices_in_period": len(counts), "total": len(primary), "rules": "GPS valid; moving >=3 km/h; stopped >=3 minutes; gaps >5 minutes split journeys; distance >=100 m; max 160 km/h; period_end is an incomplete excerpt"}})
}
