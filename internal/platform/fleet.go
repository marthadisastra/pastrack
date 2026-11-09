package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"pastrack/internal/protocol"
	"pastrack/internal/remoteoutput"
	"strconv"
	"strings"
	"time"
)

const vehicleCapacitySQL = `(SELECT vi.tank_capacity_l FROM vehicle_inventory vi WHERE vi.vehicle_id=a.id)`

const vehicleInventorySelect = `(SELECT vi.vehicle_code FROM vehicle_inventory vi WHERE vi.vehicle_id=a.id) vehicle_code,` + vehicleCapacitySQL + ` tank_capacity_l,(SELECT vi.photo_updated_at FROM vehicle_inventory vi WHERE vi.vehicle_id=a.id) photo_updated_at,`
const vehicleOutputSelect = `(SELECT CONVERT(ro.payload USING utf8mb4) FROM device_raw_messages ro WHERE ro.device_id=d.id AND (ro.payload LIKE '+RESP:GTDOS,%' OR ro.payload LIKE '+BUFF:GTDOS,%') AND ro.received_at>=va.starts_at ORDER BY SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(ro.payload USING utf8mb4),',',-2),',',1) DESC,ro.id DESC LIMIT 1) remote_output_payload,(SELECT JSON_OBJECT('state',rc.state,'desired_active',rc.desired_active,'created_at',DATE_FORMAT(rc.created_at,'%Y-%m-%dT%H:%i:%s.%fZ')) FROM remote_output_commands rc WHERE rc.device_id=d.id AND rc.created_at>=va.starts_at ORDER BY rc.id DESC LIMIT 1) remote_last_command_json,`

const vehicleTelemetrySelect = vehicleInventorySelect + vehicleOutputSelect + `(SELECT CONVERT(ir.payload USING utf8mb4) FROM device_raw_messages ir WHERE ir.device_id=d.id AND (ir.payload LIKE '+%:GTIGL,%' OR ir.payload LIKE '+%:GTVGL,%' OR ir.payload LIKE '+%:GTIGN,%' OR ir.payload LIKE '+%:GTVGN,%' OR ir.payload LIKE '+%:GTIGF,%' OR ir.payload LIKE '+%:GTVGF,%') AND ir.received_at>=va.starts_at AND SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(ir.payload USING utf8mb4),',',-2),',',1) REGEXP '^[0-9]{14}$' ORDER BY SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(ir.payload USING utf8mb4),',',-2),',',1) DESC,ir.received_at DESC,ir.id DESC LIMIT 1) ignition_payload,a.id,a.plate_number,a.vehicle_name,a.fleet_code,a.physical_ignition_state,d.id device_id,CONCAT('••••',RIGHT(d.imei,4)) device_identifier,m.model device_model,ds.last_received_at device_last_received_at,p.latitude,p.longitude,p.speed_kmh,p.heading_deg,p.gps_time,p.received_at,p.satellite_count,p.hdop,p.fix_status,fr.id fuel_sample_id,fr.fuel_liters,fr.fuel_percent,fr.raw_measurement fuel_raw,fr.raw_unit fuel_raw_unit,fr.calibration_id fuel_calibration_id,(SELECT MIN(cp.input_value) FROM fuel_calibration_points cp WHERE cp.calibration_id=fr.calibration_id) fuel_raw_min,(SELECT MAX(cp.input_value) FROM fuel_calibration_points cp WHERE cp.calibration_id=fr.calibration_id) fuel_raw_max,fr.validity fuel_validity,fr.received_at fuel_received_at,fr.reading_at fuel_reading_at,(SELECT COALESCE(t.sensor_time,r.device_time,(SELECT MAX(g.gps_time) FROM gps_positions g WHERE g.raw_message_id=t.raw_message_id)) FROM sensor_telemetry t JOIN device_raw_messages r ON r.id=t.raw_message_id WHERE t.id=fr.telemetry_id) fuel_measured_at,st.status sensor_status,st.last_received_at sensor_last_received_at,(SELECT COUNT(*) FROM alert_events ae WHERE ae.vehicle_id=a.id AND ae.state NOT IN ('resolved','closed','false_positive')) active_alerts,a.row_version`
const vehicleTelemetryJoin = ` LEFT JOIN vehicle_assignments va ON va.id=(SELECT va2.id FROM vehicle_assignments va2 WHERE va2.vehicle_id=a.id AND va2.starts_at<=UTC_TIMESTAMP(3) AND (va2.ends_at IS NULL OR va2.ends_at>UTC_TIMESTAMP(3)) ORDER BY va2.starts_at DESC,va2.id DESC LIMIT 1) LEFT JOIN gps_devices d ON d.id=va.device_id AND d.deleted_at IS NULL AND d.is_active=1 LEFT JOIN gps_device_models m ON m.id=d.model_id LEFT JOIN gps_device_status ds ON ds.device_id=d.id LEFT JOIN latest_positions lp ON lp.vehicle_id=a.id LEFT JOIN gps_positions p ON p.id=lp.position_id AND p.device_id=d.id LEFT JOIN fuel_readings fr ON fr.id=(SELECT f.id FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id WHERE f.vehicle_id=a.id AND t.device_id=d.id ORDER BY f.received_at DESC,f.id DESC LIMIT 1) LEFT JOIN fuel_sensors fs ON fs.id=fr.fuel_sensor_id LEFT JOIN ble_sensor_status st ON st.sensor_id=fs.sensor_id`
const vehicleMovementSQL = `CASE WHEN d.id IS NULL THEN 'UNASSIGNED' WHEN ds.last_received_at IS NULL OR ds.last_received_at<=UTC_TIMESTAMP(3)-INTERVAL 5 MINUTE OR ds.last_received_at>UTC_TIMESTAMP(3)+INTERVAL 1 MINUTE THEN 'OFFLINE' WHEN p.gps_time IS NULL OR p.latitude IS NULL OR p.fix_status<>'valid' THEN 'GPS_NO_FIX' WHEN p.gps_time<=UTC_TIMESTAMP(3)-INTERVAL 5 MINUTE OR p.gps_time>UTC_TIMESTAMP(3)+INTERVAL 1 MINUTE THEN 'STALE' WHEN p.speed_kmh IS NULL THEN 'UNKNOWN_SPEED' WHEN p.speed_kmh>3 THEN 'MOVING' ELSE 'STOPPED' END`

// A GPS assignment is the sole owner of gateway and accessory data on the dashboard.
func (s *Server) fleetRows(ctx context.Context) ([]map[string]any, error) {
	return fleetRowsFrom(ctx, s.DB, time.Now().UTC())
}

func fleetRowsFrom(ctx context.Context, db rowQuerier, now time.Time) ([]map[string]any, error) {
	rows, e := queryRows(ctx, db, `SELECT `+strings.ReplaceAll(vehicleInventorySelect, "a.id", "v.id")+vehicleOutputSelect+`(SELECT CONVERT(ir.payload USING utf8mb4) FROM device_raw_messages ir WHERE ir.device_id=d.id AND (ir.payload LIKE '+%:GTIGL,%' OR ir.payload LIKE '+%:GTVGL,%' OR ir.payload LIKE '+%:GTIGN,%' OR ir.payload LIKE '+%:GTVGN,%' OR ir.payload LIKE '+%:GTIGF,%' OR ir.payload LIKE '+%:GTVGF,%') AND ir.received_at>=va.starts_at AND SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(ir.payload USING utf8mb4),',',-2),',',1) REGEXP '^[0-9]{14}$' ORDER BY SUBSTRING_INDEX(SUBSTRING_INDEX(CONVERT(ir.payload USING utf8mb4),',',-2),',',1) DESC,ir.received_at DESC,ir.id DESC LIMIT 1) ignition_payload,v.id,v.plate_number,v.vehicle_name,v.fleet_code,v.physical_ignition_state,d.id device_id,CONCAT('••••',RIGHT(d.imei,4)) device_identifier,m.model device_model,ds.last_received_at device_last_received_at,p.latitude,p.longitude,p.speed_kmh,p.heading_deg,p.gps_time,p.received_at,p.satellite_count,p.hdop,p.fix_status,fr.id fuel_sample_id,fr.fuel_liters,fr.fuel_percent,fr.raw_measurement fuel_raw,fr.raw_unit fuel_raw_unit,fr.calibration_id fuel_calibration_id,(SELECT MIN(cp.input_value) FROM fuel_calibration_points cp WHERE cp.calibration_id=fr.calibration_id) fuel_raw_min,(SELECT MAX(cp.input_value) FROM fuel_calibration_points cp WHERE cp.calibration_id=fr.calibration_id) fuel_raw_max,fr.validity fuel_validity,fr.received_at fuel_received_at,fr.reading_at fuel_reading_at,(SELECT COALESCE(t.sensor_time,r.device_time,(SELECT MAX(g.gps_time) FROM gps_positions g WHERE g.raw_message_id=t.raw_message_id)) FROM sensor_telemetry t JOIN device_raw_messages r ON r.id=t.raw_message_id WHERE t.id=fr.telemetry_id) fuel_measured_at,st.status sensor_status,st.last_received_at sensor_last_received_at,(SELECT COUNT(*) FROM alert_events ae WHERE ae.vehicle_id=v.id AND ae.state NOT IN ('resolved','closed','false_positive')) active_alerts FROM vehicles v LEFT JOIN vehicle_assignments va ON va.id=(SELECT va2.id FROM vehicle_assignments va2 WHERE va2.vehicle_id=v.id AND va2.starts_at<=UTC_TIMESTAMP(3) AND (va2.ends_at IS NULL OR va2.ends_at>UTC_TIMESTAMP(3)) ORDER BY va2.starts_at DESC,va2.id DESC LIMIT 1) LEFT JOIN gps_devices d ON d.id=va.device_id AND d.deleted_at IS NULL AND d.is_active=1 LEFT JOIN gps_device_models m ON m.id=d.model_id LEFT JOIN gps_device_status ds ON ds.device_id=d.id LEFT JOIN latest_positions lp ON lp.vehicle_id=v.id LEFT JOIN gps_positions p ON p.id=lp.position_id AND p.device_id=d.id LEFT JOIN fuel_readings fr ON fr.id=(SELECT f.id FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id WHERE f.vehicle_id=v.id AND t.device_id=d.id ORDER BY f.received_at DESC,f.id DESC LIMIT 1) LEFT JOIN fuel_sensors fs ON fs.id=fr.fuel_sensor_id LEFT JOIN ble_sensor_status st ON st.sensor_id=fs.sensor_id WHERE v.deleted_at IS NULL ORDER BY v.id`)
	if e != nil {
		return nil, e
	}
	for _, row := range rows {
		if vehicle, ok := row["id"].(int64); ok {
			codes, err := queryRows(ctx, db, "SELECT DISTINCT rule_code FROM alert_events WHERE vehicle_id=? AND state NOT IN ('resolved','closed','false_positive')", vehicle)
			if err != nil {
				return nil, err
			}
			labels := make([]string, 0, len(codes))
			for _, code := range codes {
				if value, ok := code["rule_code"].(string); ok {
					labels = append(labels, value)
				}
			}
			row["active_alert_codes"] = strings.Join(labels, ",")
		}
		classifyFleet(row, now)
	}
	return rows, nil
}
func timestamp(v any) time.Time { t, _ := time.Parse(time.RFC3339Nano, stringValue(v)); return t }
func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	case time.Time:
		return value.UTC().Format(time.RFC3339Nano)
	default:
		return ""
	}
}
func numeric(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}
func freshAt(v any, now time.Time) bool {
	t := timestamp(v)
	return !t.IsZero() && !t.After(now.Add(time.Minute)) && now.Sub(t) < 5*time.Minute
}
func classifyFleet(row map[string]any, now time.Time) {
	row["remote_output_active"] = nil
	row["remote_output_reported_at"] = nil
	if report, err := remoteoutput.Decode(stringValue(row["remote_output_payload"])); err == nil && report.Active != nil {
		row["remote_output_active"] = *report.Active
		row["remote_output_reported_at"] = report.At.Format(time.RFC3339)
	}
	delete(row, "remote_output_payload")
	var command map[string]any
	if json.Unmarshal([]byte(stringValue(row["remote_last_command_json"])), &command) == nil && command != nil {
		command["desired_active"] = numeric(command["desired_active"]) == 1
		row["remote_last_command"] = command
	}
	delete(row, "remote_last_command_json")
	row["ignition_status"] = "UNKNOWN"
	row["ignition_source"] = nil
	row["ignition_reported_at"] = nil
	if report, err := protocol.DecodeIgnition(stringValue(row["ignition_payload"])); err == nil {
		row["ignition_status"] = report.State
		row["ignition_source"] = report.Source
		row["ignition_reported_at"] = report.At.Format(time.RFC3339)
		row["ignition_state_policy"] = "last_report_until_next_event"
		row["ignition_gps_time"] = report.GPSTime.Format(time.RFC3339)
	}
	delete(row, "ignition_payload")

	state := "OFFLINE"
	connectivity := "OFFLINE"
	if row["device_id"] == nil {
		state = "UNASSIGNED"
		connectivity = "UNASSIGNED"
	} else if freshAt(row["device_last_received_at"], now) {
		connectivity = "ONLINE"
		switch {
		case row["gps_time"] == nil || row["latitude"] == nil || row["fix_status"] != "valid":
			state = "GPS_NO_FIX"
		case !freshAt(row["gps_time"], now):
			state = "STALE"
		case row["speed_kmh"] == nil:
			state = "UNKNOWN_SPEED"
		case numeric(row["speed_kmh"]) > 3:
			state = "MOVING"
		default:
			state = "STOPPED"
		}
	}
	row["movement_status"] = state
	row["connectivity"] = connectivity
	row["fuel_fresh"] = freshAt(row["fuel_received_at"], now) && freshAt(row["fuel_measured_at"], now)
	row["sensor_last_status"] = row["sensor_status"]
	if row["sensor_last_received_at"] != nil && !freshAt(row["sensor_last_received_at"], now) {
		row["sensor_status"] = "STALE"
	}
	row["gps_fresh"] = connectivity == "ONLINE" && row["fix_status"] == "valid" && row["latitude"] != nil && freshAt(row["gps_time"], now)
}
func fleetSummary(rows []map[string]any) map[string]any {
	summary := map[string]any{"total_fleet": len(rows), "moving": 0, "stopped": 0, "vehicles_offline": 0, "unassigned": 0, "gps_stale": 0, "gps_no_fix": 0, "unknown_speed": 0, "devices_online": 0, "active_alerts": 0, "gps_coverage": 0, "fuel_coverage": 0}
	keys := map[string]string{"MOVING": "moving", "STOPPED": "stopped", "OFFLINE": "vehicles_offline", "UNASSIGNED": "unassigned", "STALE": "gps_stale", "GPS_NO_FIX": "gps_no_fix", "UNKNOWN_SPEED": "unknown_speed"}
	for _, row := range rows {
		if k := keys[stringValue(row["movement_status"])]; k != "" {
			summary[k] = summary[k].(int) + 1
		}
		if row["connectivity"] == "ONLINE" {
			summary["devices_online"] = summary["devices_online"].(int) + 1
		}
		summary["active_alerts"] = summary["active_alerts"].(int) + int(numeric(row["active_alerts"]))
		if row["gps_fresh"] == true {
			summary["gps_coverage"] = summary["gps_coverage"].(int) + 1
		}
		if row["fuel_fresh"] == true && row["fuel_validity"] == "valid" {
			summary["fuel_coverage"] = summary["fuel_coverage"].(int) + 1
		}
	}
	return summary
}

// Fleet activity is an alarm stream grouped by vehicle and alert type.
func (s *Server) fleetActivity(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 50 {
			limit = n
		}
	}
	rows, e := queryRows(r.Context(), s.DB, `SELECT CONCAT('alert-',a.vehicle_id,'-',a.rule_code) id,'alert' kind,a.vehicle_id,v.plate_number,a.rule_code,a.severity,COALESCE(a.title,a.rule_code) title,MAX(a.description) description,COUNT(*) alert_count,MAX(a.raised_at) received_at,MAX(a.last_detected_at) event_at,MAX(a.state) state FROM alert_events a JOIN vehicles v ON v.id=a.vehicle_id WHERE a.state NOT IN ('resolved','closed','false_positive') GROUP BY a.vehicle_id,a.rule_code,v.plate_number,a.severity,a.title ORDER BY MAX(a.raised_at) DESC LIMIT `+strconv.Itoa(limit))
	if e != nil {
		databaseError(w, e)
		return
	}
	respond(w, 200, map[string]any{"data": rows})
}
