package platform

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Dashboard v1 is shared by REST snapshots and the forthcoming WebSocket projection.
// Nullable measurements remain nullable; absence is never a zero-valued sensor fact.
type dashboardGPS struct {
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Speed      *float64 `json:"speedKmh"`
	Heading    *float64 `json:"headingDeg"`
	Satellites *float64 `json:"satellites"`
	HDOP       *float64 `json:"hdop"`
	Fix        string   `json:"fixStatus"`
	Fresh      bool     `json:"fresh"`
	MeasuredAt *string  `json:"measuredAt"`
	ReceivedAt *string  `json:"receivedAt"`
}
type dashboardFuel struct {
	SampleID      *string  `json:"sampleId"`
	Liters        *float64 `json:"liters"`
	Percent       *float64 `json:"percent"`
	Capacity      *float64 `json:"capacityLiters"`
	CalibrationID *string  `json:"calibrationId"`
	Validity      string   `json:"validity"`
	Fresh         bool     `json:"fresh"`
	MeasuredAt    *string  `json:"measuredAt"`
	ReceivedAt    *string  `json:"receivedAt"`
}
type dashboardIgnition struct {
	Status        string  `json:"status"`
	Source        *string `json:"source"`
	ReportedAt    *string `json:"reportedAt"`
	PhysicalState string  `json:"physicalState"`
}
type dashboardVehicle struct {
	ID                string            `json:"id"`
	Code              string            `json:"code"`
	Plate             string            `json:"plate"`
	Name              string            `json:"name"`
	DeviceID          *string           `json:"deviceId"`
	DeviceIdentifier  *string           `json:"deviceIdentifier"`
	Monitored         bool              `json:"monitored"`
	Movement          string            `json:"movementStatus"`
	Connectivity      string            `json:"connectivity"`
	GatewayReceivedAt *string           `json:"gatewayReceivedAt"`
	GPS               dashboardGPS      `json:"gps"`
	Fuel              dashboardFuel     `json:"fuel"`
	Ignition          dashboardIgnition `json:"ignition"`
	SensorStatus      *string           `json:"sensorStatus"`
	SensorReceivedAt  *string           `json:"sensorReceivedAt"`
	ActiveAlerts      int               `json:"activeAlerts"`
	AlertCodes        []string          `json:"alertCodes"`
}
type dashboardSummary struct {
	Total                int            `json:"totalVehicles"`
	Monitored            int            `json:"monitoredVehicles"`
	States               map[string]int `json:"movementCounts"`
	DevicesOnline        int            `json:"devicesOnline"`
	ActiveAlerts         int            `json:"activeAlerts"`
	GPSFresh             int            `json:"gpsFreshVehicles"`
	FuelLiters           *float64       `json:"fuelLiters"`
	FuelCapacity         *float64       `json:"fuelCapacityLiters"`
	FuelPercent          *float64       `json:"fuelFillPercent"`
	FuelValidFresh       int            `json:"fuelValidFreshVehicles"`
	FuelCapacityEligible int            `json:"fuelCapacityEligibleVehicles"`
	CalibrationUnknown   int            `json:"calibrationUnknownVehicles"`
	AverageHDOP          *float64       `json:"averageHdop"`
	HDOPSamples          int            `json:"hdopSamples"`
}
type dashboardFuelPoint struct {
	ID            string   `json:"sampleId"`
	At            string   `json:"at"`
	Liters        *float64 `json:"liters"`
	Percent       *float64 `json:"percent"`
	Validity      string   `json:"validity"`
	CalibrationID *string  `json:"calibrationId"`
}

func dashboardNumber(v any) *float64 {
	if v == nil {
		return nil
	}
	var n float64
	switch x := v.(type) {
	case float64:
		n = x
	case int64:
		n = float64(x)
	case int:
		n = float64(x)
	case string:
		var err error
		n, err = strconv.ParseFloat(x, 64)
		if err != nil {
			return nil
		}
	case []byte:
		return dashboardNumber(string(x))
	default:
		return nil
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return nil
	}
	return &n
}
func dashboardText(v any) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprint(v)
	if s == "" {
		return nil
	}
	return &s
}
func dashboardString(v any, fallback string) string {
	if s := dashboardText(v); s != nil {
		return *s
	}
	return fallback
}
func dashboardProject(rows []map[string]any) ([]dashboardVehicle, dashboardSummary) {
	vehicles := make([]dashboardVehicle, 0, len(rows))
	summary := dashboardSummary{Total: len(rows), States: map[string]int{"MOVING": 0, "STOPPED": 0, "OFFLINE": 0, "UNASSIGNED": 0, "STALE": 0, "GPS_NO_FIX": 0, "UNKNOWN_SPEED": 0}}
	var liters, capacity, eligibleLiters, hdop float64
	for _, r := range rows {
		id := dashboardString(r["id"], "")
		code := dashboardString(r["vehicle_code"], dashboardString(r["plate_number"], "Vehicle "+id))
		v := dashboardVehicle{ID: id, Code: code, Plate: dashboardString(r["plate_number"], ""), Name: dashboardString(r["vehicle_name"], ""), DeviceID: dashboardText(r["device_id"]), DeviceIdentifier: dashboardText(r["device_identifier"]), Monitored: r["device_id"] != nil, Movement: dashboardString(r["movement_status"], "UNKNOWN_SPEED"), Connectivity: dashboardString(r["connectivity"], "OFFLINE"), GatewayReceivedAt: dashboardText(r["device_last_received_at"]), SensorStatus: dashboardText(r["sensor_status"]), SensorReceivedAt: dashboardText(r["sensor_last_received_at"]), AlertCodes: []string{},
			GPS:      dashboardGPS{Latitude: dashboardNumber(r["latitude"]), Longitude: dashboardNumber(r["longitude"]), Speed: dashboardNumber(r["speed_kmh"]), Heading: dashboardNumber(r["heading_deg"]), Satellites: dashboardNumber(r["satellite_count"]), HDOP: dashboardNumber(r["hdop"]), Fix: dashboardString(r["fix_status"], "unknown"), Fresh: r["gps_fresh"] == true, MeasuredAt: dashboardText(r["gps_time"]), ReceivedAt: dashboardText(r["received_at"])},
			Fuel:     dashboardFuel{SampleID: dashboardText(r["fuel_sample_id"]), Liters: dashboardNumber(r["fuel_liters"]), Percent: dashboardNumber(r["fuel_percent"]), Capacity: dashboardNumber(r["tank_capacity_l"]), CalibrationID: dashboardText(r["fuel_calibration_id"]), Validity: dashboardString(r["fuel_validity"], "unknown"), Fresh: r["fuel_fresh"] == true, MeasuredAt: dashboardText(r["fuel_measured_at"]), ReceivedAt: dashboardText(r["fuel_received_at"])},
			Ignition: dashboardIgnition{Status: dashboardString(r["ignition_status"], "UNKNOWN"), Source: dashboardText(r["ignition_source"]), ReportedAt: dashboardText(r["ignition_reported_at"]), PhysicalState: dashboardString(r["physical_ignition_state"], "UNKNOWN")},
		}
		if n := dashboardNumber(r["active_alerts"]); n != nil && *n > 0 {
			v.ActiveAlerts = int(*n)
		}
		for _, code := range strings.Split(dashboardString(r["active_alert_codes"], ""), ",") {
			if code != "" {
				v.AlertCodes = append(v.AlertCodes, code)
			}
		}
		if v.Monitored {
			summary.Monitored++
			if v.Fuel.CalibrationID == nil {
				summary.CalibrationUnknown++
			}
		}
		summary.States[v.Movement]++
		if v.Connectivity == "ONLINE" {
			summary.DevicesOnline++
		}
		summary.ActiveAlerts += v.ActiveAlerts
		if v.GPS.Fresh {
			summary.GPSFresh++
			if v.GPS.HDOP != nil && *v.GPS.HDOP >= 0 {
				hdop += *v.GPS.HDOP
				summary.HDOPSamples++
			}
		}
		if v.Monitored && v.Fuel.Validity == "valid" && v.Fuel.Fresh && v.Fuel.Liters != nil && *v.Fuel.Liters >= 0 {
			summary.FuelValidFresh++
			liters += *v.Fuel.Liters
			if v.Fuel.Capacity != nil && *v.Fuel.Capacity > 0 {
				summary.FuelCapacityEligible++
				capacity += *v.Fuel.Capacity
				eligibleLiters += *v.Fuel.Liters
			}
		}
		vehicles = append(vehicles, v)
	}
	if summary.FuelValidFresh > 0 {
		summary.FuelLiters = &liters
	}
	if summary.FuelCapacityEligible > 0 {
		summary.FuelCapacity = &capacity
		percent := eligibleLiters / capacity * 100
		summary.FuelPercent = &percent
	}
	if summary.HDOPSamples > 0 {
		average := hdop / float64(summary.HDOPSamples)
		summary.AverageHDOP = &average
	}
	return vehicles, summary
}
func (s *Server) dashboardSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		fail(w, 405, "method_not_allowed")
		return
	}
	vehicles, summary, asOf, err := s.dashboardSnapshotData(r.Context())
	if err != nil {
		databaseError(w, err)
		return
	}
	respond(w, 200, map[string]any{"data": map[string]any{"vehicles": vehicles, "summary": summary}, "meta": map[string]any{"schemaVersion": 1, "asOf": asOf, "mode": "snapshot", "realtime": false, "scope": "all_active_vehicles", "fuelBasis": "valid_fresh_readings; fill percent weighted over known positive capacity"}})
}

func (s *Server) dashboardSnapshotData(ctx context.Context) ([]dashboardVehicle, dashboardSummary, time.Time, error) {
	asOf := time.Now().UTC()
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, dashboardSummary{}, time.Time{}, err
	}
	defer tx.Rollback()
	rows, err := fleetRowsFrom(ctx, tx, asOf)
	if err != nil {
		return nil, dashboardSummary{}, time.Time{}, err
	}
	if identity, ok := ctx.Value(key{}).(*Identity); ok && identity != nil && identity.Role != "admin" {
		allowed, e := queryRows(ctx, tx, "SELECT vehicle_id FROM dashboard_vehicle_scopes WHERE user_id=?", identity.ID)
		scopeAvailable := true
		if e != nil {
			// Optional scope migration may not yet be installed on an older live DB.
			// Keep backward-compatible all-fleet behavior until it is applied.
			if strings.Contains(e.Error(), "doesn't exist") {
				allowed = nil
				scopeAvailable = false
			} else {
				return nil, dashboardSummary{}, time.Time{}, e
			}
		}
		if !scopeAvailable {
			goto scopeDone
		}
		ids := make(map[string]struct{}, len(allowed))
		for _, row := range allowed {
			ids[dashboardString(row["vehicle_id"], "")] = struct{}{}
		}
		filteredRows := rows[:0]
		for _, row := range rows {
			if _, ok := ids[dashboardString(row["id"], "")]; ok {
				filteredRows = append(filteredRows, row)
			}
		}
		rows = filteredRows
	}

scopeDone:
	vehicles, summary := dashboardProject(rows)
	if err = tx.Commit(); err != nil {
		return nil, dashboardSummary{}, time.Time{}, err
	}
	return vehicles, summary, asOf, nil
}

func dashboardHistoryDuration(value string) (time.Duration, bool) {
	switch value {
	case "12h":
		return 12 * time.Hour, true
	case "24h":
		return 24 * time.Hour, true
	case "7d":
		return 7 * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func (s *Server) dashboardFuelHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	vehicleID, err := strconv.ParseInt(r.URL.Query().Get("vehicleId"), 10, 64)
	duration, validRange := dashboardHistoryDuration(r.URL.Query().Get("range"))
	if err != nil || vehicleID <= 0 || !validRange {
		fail(w, http.StatusBadRequest, "invalid_dashboard_history_query")
		return
	}
	to := time.Now().UTC()
	from := to.Add(-duration)
	rows, err := queryRows(r.Context(), s.DB, `SELECT f.id,COALESCE(f.reading_at,f.received_at) sample_at,f.fuel_liters,f.fuel_percent,f.validity,f.calibration_id FROM fuel_readings f JOIN vehicles v ON v.id=f.vehicle_id AND v.deleted_at IS NULL WHERE f.vehicle_id=? AND COALESCE(f.reading_at,f.received_at)>=? AND COALESCE(f.reading_at,f.received_at)<=? ORDER BY COALESCE(f.reading_at,f.received_at),f.id LIMIT 100001`, vehicleID, from, to.Add(time.Minute))
	if err != nil {
		databaseError(w, err)
		return
	}
	if len(rows) > 100000 {
		fail(w, http.StatusRequestEntityTooLarge, "dashboard_history_too_dense")
		return
	}
	sourceCount := len(rows)
	stride := max(1, (sourceCount+3999)/4000)
	points := make([]dashboardFuelPoint, 0, min(sourceCount, 5000))
	for index, row := range rows {
		last := index == sourceCount-1
		validityChanged := index > 0 && dashboardString(row["validity"], "unknown") != dashboardString(rows[index-1]["validity"], "unknown")
		if index%stride != 0 && !last && !validityChanged {
			continue
		}
		if len(points) == 4999 && !last {
			continue
		}
		at := dashboardString(row["sample_at"], "")
		if at == "" {
			continue
		}
		points = append(points, dashboardFuelPoint{ID: dashboardString(row["id"], ""), At: at, Liters: dashboardNumber(row["fuel_liters"]), Percent: dashboardNumber(row["fuel_percent"]), Validity: dashboardString(row["validity"], "unknown"), CalibrationID: dashboardText(row["calibration_id"])})
	}
	respond(w, http.StatusOK, map[string]any{"data": map[string]any{"vehicleId": strconv.FormatInt(vehicleID, 10), "range": r.URL.Query().Get("range"), "points": points}, "meta": map[string]any{"schemaVersion": 1, "from": from, "to": to, "limit": 5000, "sourceCount": sourceCount, "sampled": sourceCount > len(points)}})
}
