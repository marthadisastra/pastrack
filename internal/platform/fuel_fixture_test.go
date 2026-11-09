package platform

import (
	"math"
	"time"
)

// Deterministic, isolated demonstration data. Never inserted into operational telemetry.
func fuelSimulation(scenario string) map[string]any {
	type point struct{ lat, lon float64 }
	route := []point{{-6.1754, 106.8272}, {-6.208, 106.875}, {-6.245, 106.94}, {-6.278, 107.055}, {-6.313, 107.15}, {-6.353, 107.29}, {-6.389, 107.43}, {-6.434, 107.56}, {-6.474, 107.69}, {-6.492, 107.84}, {-6.522, 108.00}, {-6.553, 108.15}, {-6.59, 108.30}, {-6.64, 108.45}, {-6.706, 108.557}}
	distances := []float64{0}
	total := 0.0
	for i := 1; i < len(route); i++ {
		a, b := route[i-1], route[i]
		rad := math.Pi / 180
		h := math.Pow(math.Sin((b.lat-a.lat)*rad/2), 2) + math.Cos(a.lat*rad)*math.Cos(b.lat*rad)*math.Pow(math.Sin((b.lon-a.lon)*rad/2), 2)
		total += 6371 * 2 * math.Asin(math.Sqrt(h))
		distances = append(distances, total)
	}
	weights := make([]float64, 361)
	sum := 0.0
	for i := 1; i <= 360; i++ {
		if i < 175 || i > 185 {
			weights[i] = .85 + .15*math.Sin(float64(i)/19)
			sum += weights[i]
		}
	}
	start := time.Date(2026, 10, 31, 6, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	stamp := func(i int) string { return start.Add(time.Duration(i) * time.Minute).UTC().Format(time.RFC3339) }
	positions, readings, events := []map[string]any{}, []map[string]any{}, []map[string]any{}
	distance := 0.0
	consumed := 0.0
	for i := 0; i <= 360; i++ {
		step := weights[i] / sum * total
		distance += step
		consumed = distance / 3.5
		segment := 1
		for segment < len(distances)-1 && distances[segment] < distance {
			segment++
		}
		a, b := route[segment-1], route[segment]
		fraction := (distance - distances[segment-1]) / (distances[segment] - distances[segment-1])
		fraction = math.Max(0, math.Min(1, fraction))
		lat, lon := a.lat+(b.lat-a.lat)*fraction, a.lon+(b.lon-a.lon)*fraction
		drop := 0.0
		if scenario == "anomaly" && i > 180 {
			drop = math.Min(40, float64(i-180)*20)
		}
		liters := math.Round((320-consumed-drop)*1000) / 1000
		ignition := "ON"
		if i >= 175 && i <= 185 {
			ignition = "OFF"
		}
		positions = append(positions, map[string]any{"id": i + 1, "latitude": lat, "longitude": lon, "speed_kmh": step * 60, "gps_time": stamp(i), "device_id": "sim-gps", "satellite_count": 14, "hdop": .8, "ignition_state": ignition})
		readings = append(readings, map[string]any{"id": i + 1, "fuel_sensor_id": "sim-tank", "received_at": stamp(i), "reading_at": stamp(i), "fuel_liters": liters, "fuel_percent": liters / 4, "validity": "valid", "media": "fuel_tank", "calibration_id": "sim-calibration", "raw_measurement": math.Round(1000 + liters*10), "raw_unit": "simulated_raw", "temperature_c": 29.4, "rssi": -64, "ignition_state": ignition})
	}
	if scenario == "anomaly" {
		a, b, p := readings[180], readings[182], positions[182]
		events = append(events, map[string]any{"id": "SIM-DROP-1031", "vehicle_id": "sim-anomaly", "fuel_sensor_id": "sim-tank", "event_type": "fuel_drop", "delta_l": -40, "event_at": stamp(182), "start_at": stamp(180), "end_at": stamp(182), "start_liters": a["fuel_liters"], "end_liters": b["fuel_liters"], "start_raw": a["raw_measurement"], "end_raw": b["raw_measurement"], "raw_unit": "simulated_raw", "duration_seconds": 120, "latitude": p["latitude"], "longitude": p["longitude"], "speed_kmh": 0, "gps_time": stamp(182), "ignition_state": "OFF", "state": "open", "row_version": 0, "note": "SIMULASI: penurunan 40 L dalam 2 menit pada titik berhenti.", "confidence": "simulation", "rule_version": "simulation-v1"})
	}
	name := "SIMULASI NORMAL"
	if scenario == "anomaly" {
		name = "SIMULASI ANOMALI"
	}
	return map[string]any{"simulation": true, "scenario": scenario, "simulation_label": name + " · 31 Oktober 2026", "description": "Data sintetis Jakarta–Cirebon; koordinat rute representatif, bukan rekaman GPS atau hasil routing jalan. Konsumsi normal 3,5 km/L; berhenti 08.55–09.05 WIB.", "vehicle": map[string]any{"plate_number": name, "vehicle_name": "Perjalanan Jakarta–Cirebon"}, "generated_at": time.Now().UTC(), "from": stamp(0), "to": stamp(360), "basis": "synthetic one-minute samples, WIB", "idle_available": false, "positions": positions, "trips": historyJourneys(positions), "readings": readings, "events": events, "sensors": []map[string]any{{"fuel_sensor_id": "sim-tank", "model": "CFS300B (simulasi)", "tank_name": "Tangki simulasi 400 L", "capacity_l": 400}}, "geofences": []map[string]any{}, "rules": []map[string]any{{"name": "Skenario sintetis; bukan aturan produksi", "code": "fuel_drop", "version": 1, "configuration": `{"drop_liters":40,"drop_minutes":2,"at":"09:00–09:02 WIB","production_rule":false}`}}, "analytics": []map[string]any{{"consumed_l": consumed, "distance_m": total * 1000, "coverage_percent": 100, "period_start": stamp(0), "period_end": stamp(360)}}}
}
