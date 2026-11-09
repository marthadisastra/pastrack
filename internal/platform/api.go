package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type resource struct {
	Table, Select, Join, Search, Vehicle, Time string
	Admin                                      bool
	Write                                      []string
	Required                                   []string
}

var resources = map[string]resource{
	"vehicles":              {Table: "vehicles", Select: vehicleTelemetrySelect, Join: vehicleTelemetryJoin, Search: "CONCAT_WS(' ',a.plate_number,a.vehicle_name,a.fleet_code,(SELECT vi.vehicle_code FROM vehicle_inventory vi WHERE vi.vehicle_id=a.id))", Vehicle: "a.id", Write: []string{"plate_number", "vehicle_name", "fleet_code"}, Required: []string{"plate_number", "vehicle_name"}},
	"devices":               {Table: "gps_devices", Select: "a.id,CONCAT('••••',RIGHT(a.imei,4)) AS identifier,a.model_id,m.manufacturer,m.model,a.firmware_version,a.protocol_version,a.is_active,assignment.vehicle_id,linked_vehicle.plate_number,linked_vehicle.vehicle_name,ds.last_received_at,CASE WHEN ds.last_received_at IS NULL THEN 'UNKNOWN' WHEN ds.last_received_at<=UTC_TIMESTAMP(3)-INTERVAL 5 MINUTE OR ds.last_received_at>UTC_TIMESTAMP(3)+INTERVAL 1 MINUTE THEN 'DEVICE_OFFLINE' ELSE 'ONLINE' END AS connectivity,gq.fix_status,gq.satellite_count,gq.hdop,gq.event_at gps_quality_at,net.mcc,net.mnc,net.lac,net.cell_id,net.rssi,net.event_at network_at,bat.status_code_raw battery_event,bat.event_at battery_event_at,CAST(JSON_UNQUOTE(JSON_EXTRACT(volts.decoded_status,'$.backup_battery_voltage')) AS DECIMAL(4,2)) battery_voltage,volts.event_at battery_voltage_at", Join: " LEFT JOIN gps_device_models m ON m.id=a.model_id LEFT JOIN vehicle_assignments assignment ON assignment.id=(SELECT va.id FROM vehicle_assignments va WHERE va.device_id=a.id AND va.starts_at<=UTC_TIMESTAMP(3) AND (va.ends_at IS NULL OR va.ends_at>UTC_TIMESTAMP(3)) ORDER BY va.starts_at DESC,va.id DESC LIMIT 1) LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=assignment.vehicle_id LEFT JOIN gps_device_status ds ON ds.device_id=a.id LEFT JOIN gps_quality_logs gq ON gq.id=(SELECT q.id FROM gps_quality_logs q WHERE q.device_id=a.id ORDER BY COALESCE(q.event_at,q.created_at) DESC,q.id DESC LIMIT 1) LEFT JOIN network_status_logs net ON net.id=(SELECT n.id FROM network_status_logs n WHERE n.device_id=a.id AND (NULLIF(n.mcc,'') IS NOT NULL OR NULLIF(n.mnc,'') IS NOT NULL OR NULLIF(n.lac,'') IS NOT NULL OR NULLIF(n.cell_id,'') IS NOT NULL OR n.rssi IS NOT NULL) ORDER BY COALESCE(n.event_at,n.created_at) DESC,n.id DESC LIMIT 1) LEFT JOIN device_status_logs bat ON bat.id=(SELECT bs.id FROM device_status_logs bs WHERE bs.device_id=a.id AND bs.decoded_status IS NOT NULL AND bs.status_code_raw IN ('GTBPL','GTBTC') ORDER BY bs.event_at DESC,bs.id DESC LIMIT 1) LEFT JOIN device_status_logs volts ON volts.id=(SELECT vs.id FROM device_status_logs vs WHERE vs.device_id=a.id AND vs.decoded_status IS NOT NULL AND vs.status_code_raw='GTBPL' ORDER BY vs.event_at DESC,vs.id DESC LIMIT 1)", Search: "m.model", Write: []string{"imei", "model_id", "firmware_version", "protocol_version", "is_active"}, Required: []string{"imei", "model_id", "is_active"}},
	"device-models":         {Table: "gps_device_models", Select: "a.id,a.manufacturer,a.model,a.protocol_family", Search: "a.model", Write: []string{"manufacturer", "model", "protocol_family"}, Required: []string{"manufacturer", "model", "protocol_family"}},
	"sensors":               {Table: "ble_sensors", Select: "a.id,CONCAT('••••',RIGHT(a.sensor_identifier,4)) AS identifier,a.model,srn.reported_name AS sensor_name,a.is_active,d.id AS device_id,CONCAT('••••',RIGHT(d.imei,4)) AS device_identifier,source_raw.protocol_version AS device_protocol_version,v.id AS vehicle_id,v.plate_number,v.vehicle_name," + strings.ReplaceAll(vehicleCapacitySQL, "a.id", "v.id") + " AS capacity_l,CASE WHEN t.vehicle_id IS NOT NULL AND (v.id IS NULL OR t.vehicle_id<>v.id) THEN 'uncalibrated' ELSE f.decoding_state END AS decoding_state,st.volume_raw,st.fuel_height_raw,st.received_at,(d.id IS NOT NULL AND v.id IS NOT NULL AND a.is_active=1) AS pairing_active", Join: " LEFT JOIN sensor_reported_names srn ON srn.sensor_id=a.id LEFT JOIN fuel_sensors f ON f.sensor_id=a.id LEFT JOIN fuel_tank_profiles t ON t.id=f.tank_profile_id LEFT JOIN sensor_telemetry latest_sensor ON latest_sensor.id=(SELECT latest.id FROM sensor_telemetry latest JOIN device_raw_messages raw ON raw.id=latest.raw_message_id AND raw.device_id=latest.device_id WHERE latest.sensor_id=a.id ORDER BY latest.received_at DESC,latest.id DESC LIMIT 1) LEFT JOIN gps_devices d ON d.id=latest_sensor.device_id AND d.is_active=1 AND d.deleted_at IS NULL LEFT JOIN vehicle_assignments va ON va.id=(SELECT CASE WHEN COUNT(*)=1 THEN MAX(va2.id) END FROM vehicle_assignments va2 WHERE va2.device_id=d.id AND va2.starts_at<=UTC_TIMESTAMP(3) AND (va2.ends_at IS NULL OR va2.ends_at>UTC_TIMESTAMP(3))) LEFT JOIN vehicles v ON v.id=va.vehicle_id AND v.deleted_at IS NULL LEFT JOIN sensor_telemetry st ON st.id=latest_sensor.id LEFT JOIN device_raw_messages source_raw ON source_raw.id=st.raw_message_id", Search: "CONCAT_WS(' ',srn.reported_name,a.sensor_identifier,a.mac,a.model,d.imei,v.plate_number,v.vehicle_name)", Write: []string{"sensor_identifier", "mac", "model", "is_active"}, Required: []string{"sensor_identifier", "model", "is_active"}},
	"fuel/sensors":          {Table: "fuel_sensors", Select: "a.id,a.sensor_id,a.tank_profile_id,a.measurement_unit,a.decoding_state", Write: []string{"sensor_id", "tank_profile_id", "measurement_unit", "decoding_state"}, Required: []string{"sensor_id", "decoding_state"}},
	"fuel/readings":         {Table: "fuel_readings", Select: "a.id,a.vehicle_id,a.fuel_sensor_id,a.reading_at,a.received_at,a.raw_measurement,a.raw_unit,a.fuel_liters,a.fuel_percent,a.validity,a.invalid_reason,a.calibration_id,st.fuel_height_raw,st.volume_raw,st.tilt_raw,st.temperature_c,st.battery_percent,st.device_id,linked_vehicle.plate_number,linked_vehicle.vehicle_name,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier,CONCAT('••••',RIGHT(linked_sensor.sensor_identifier,4)) AS sensor_identifier,(SELECT nr.reported_name FROM sensor_reported_names nr WHERE nr.sensor_id=linked_sensor.id) AS sensor_name", Join: " LEFT JOIN sensor_telemetry st ON st.id=a.telemetry_id LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN gps_devices linked_device ON linked_device.id=st.device_id LEFT JOIN fuel_sensors linked_fuel_sensor ON linked_fuel_sensor.id=a.fuel_sensor_id LEFT JOIN ble_sensors linked_sensor ON linked_sensor.id=linked_fuel_sensor.sensor_id", Vehicle: "a.vehicle_id", Time: "a.received_at"},
	"fuel/events":           {Table: "fuel_events", Select: "a.id,a.vehicle_id,a.fuel_sensor_id,a.event_type,a.delta_l,a.event_at,a.confidence,linked_vehicle.plate_number,linked_vehicle.vehicle_name,CONCAT('••••',RIGHT(linked_sensor.sensor_identifier,4)) AS sensor_identifier,(SELECT nr.reported_name FROM sensor_reported_names nr WHERE nr.sensor_id=linked_sensor.id) AS sensor_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN fuel_sensors linked_fuel_sensor ON linked_fuel_sensor.id=a.fuel_sensor_id LEFT JOIN ble_sensors linked_sensor ON linked_sensor.id=linked_fuel_sensor.sensor_id", Vehicle: "a.vehicle_id", Time: "a.event_at"},
	"fuel/analytics":        {Table: "fuel_consumption_summaries", Select: "a.id,a.vehicle_id,a.period_start,a.period_end,a.consumed_l,a.distance_m,a.coverage_percent,linked_vehicle.plate_number,linked_vehicle.vehicle_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id", Vehicle: "a.vehicle_id", Time: "a.period_start"},
	"gps/positions":         {Table: "gps_positions", Select: "a.id,a.vehicle_id,a.device_id,a.latitude,a.longitude,a.speed_kmh,a.heading_deg,a.altitude_m,a.gps_time,a.received_at,a.satellite_count,a.hdop,a.fix_status,a.validation_state,linked_vehicle.plate_number,linked_vehicle.vehicle_name,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Vehicle: "a.vehicle_id", Time: "a.gps_time"},
	"gps/quality":           {Table: "gps_quality_logs", Select: "a.id,a.device_id,a.position_id,a.satellite_count,a.hdop,a.fix_status,a.event_at,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Time: "a.event_at"},
	"trips":                 {Table: "trips", Select: "a.id,a.vehicle_id,a.started_at,a.ended_at,a.distance_m,a.driving_seconds,a.stopped_seconds,a.max_speed_kmh,a.average_speed_kmh,a.confidence,linked_vehicle.plate_number,linked_vehicle.vehicle_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id", Vehicle: "a.vehicle_id", Time: "a.started_at"},
	"devices/health":        {Table: "gps_device_status", Select: "a.id,a.device_id,a.last_received_at,CASE WHEN a.last_received_at IS NULL THEN 'UNKNOWN' WHEN a.last_received_at<UTC_TIMESTAMP()-INTERVAL 5 MINUTE THEN 'DEVICE_OFFLINE' ELSE 'ONLINE' END AS connectivity,a.last_raw_message_id,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Time: "a.last_received_at"},
	"devices/connectivity":  {Table: "network_status_logs", Select: "a.id,a.device_id,a.event_at,a.mcc,a.mnc,a.lac,a.cell_id,a.rssi,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Time: "a.event_at"},
	"devices/data-activity": {Table: "protocol_parse_logs", Select: "a.id,a.raw_message_id,a.parser_version,a.parse_status,a.processing_status,a.processed_at,message.device_id,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN device_raw_messages message ON message.id=a.raw_message_id LEFT JOIN gps_devices linked_device ON linked_device.id=message.device_id", Time: "a.processed_at"},
	"devices/raw-messages":  {Table: "device_raw_messages", Select: "a.id,a.device_id,a.protocol_version,HEX(a.payload_sha256) AS payload_sha256,a.received_at,mt.code AS message_type,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN protocol_message_types mt ON mt.id=a.message_type_id LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Time: "a.received_at", Admin: true},
	"devices/parse-errors":  {Table: "protocol_parser_errors", Select: "a.id,a.parse_log_id,a.code,a.field_path,a.detail_redacted,a.created_at", Time: "a.created_at", Admin: true},
	"events":                {Table: "device_events", Select: "a.id,a.device_id,a.vehicle_id,a.event_at,a.received_at,a.raw_code,a.raw_value,a.verification_state,linked_vehicle.plate_number,linked_vehicle.vehicle_name,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id", Vehicle: "a.vehicle_id", Time: "a.received_at"},
	"geofences":             {Table: "geofences", Select: "a.id,a.name,a.shape,a.center_lat,a.center_lon,a.radius_m,a.polygon_geojson,a.is_active,a.row_version", Search: "a.name", Write: []string{"name", "shape", "center_lat", "center_lon", "radius_m", "polygon_geojson", "is_active"}, Required: []string{"name", "shape", "is_active"}},
	"geofence-events":       {Table: "geofence_events", Select: "a.id,a.geofence_id,a.vehicle_id,a.position_id,a.transition,a.event_at,linked_vehicle.plate_number,linked_vehicle.vehicle_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id", Vehicle: "a.vehicle_id", Time: "a.event_at"},
	"alerts/rules":          {Table: "alert_rules", Select: "a.id,a.name,a.event_type_id,a.severity_id,a.configuration,a.version,a.enabled", Search: "a.name", Write: []string{"name", "event_type_id", "severity_id", "configuration", "version", "enabled"}, Required: []string{"name", "severity_id", "configuration", "version", "enabled"}},
	"alerts/events":         {Table: "alert_events", Select: "a.id,a.vehicle_id,a.rule_code,a.severity,a.title,a.description,a.state,a.raised_at,a.last_detected_at,a.acknowledged_at,a.resolved_at,a.latitude,a.longitude,a.rule_id,linked_vehicle.plate_number,linked_vehicle.vehicle_name,crash_set.crash_counter,crash_set.primary_message_type,crash_set.evidence_status,crash_set.expected_frames,crash_set.received_frames,crash_set.sequence_key", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN crash_evidence_sets crash_set ON a.rule_code='CRASH' AND a.scope_key=CONCAT('crash:',crash_set.sequence_key)", Search: "a.rule_code", Vehicle: "a.vehicle_id", Time: "a.raised_at"},
	"crashes/history":       {Table: "crash_evidence_sets", Select: "a.id,a.vehicle_id,a.device_id,a.crash_counter,a.anchor_at,a.primary_message_type,a.evidence_status,a.expected_frames,a.received_frames,a.sequence_key,linked_vehicle.plate_number,linked_vehicle.vehicle_name,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier,alert.id AS alert_id,alert.state AS alert_state,alert.severity,alert.title", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id LEFT JOIN alert_events alert ON alert.rule_code='CRASH' AND alert.scope_key=CONCAT('crash:',a.sequence_key)", Vehicle: "a.vehicle_id", Time: "a.anchor_at"},
	"sensor-telemetry":      {Table: "sensor_telemetry", Select: "a.id,a.sensor_id,a.device_id,a.sensor_time,a.received_at,a.tilt_raw,a.fuel_height_raw,a.volume_raw,a.battery_mv,a.battery_percent,a.temperature_c,a.rssi,a.validation_state,CONCAT('••••',RIGHT(linked_device.imei,4)) AS device_identifier,CONCAT('••••',RIGHT(linked_sensor.sensor_identifier,4)) AS sensor_identifier,(SELECT nr.reported_name FROM sensor_reported_names nr WHERE nr.sensor_id=linked_sensor.id) AS sensor_name", Join: " LEFT JOIN gps_devices linked_device ON linked_device.id=a.device_id LEFT JOIN ble_sensors linked_sensor ON linked_sensor.id=a.sensor_id", Time: "a.received_at"},
	"calibrations":          {Table: "fuel_calibrations", Select: "a.id,a.fuel_sensor_id,a.tank_profile_id,a.revision,a.media,a.input_quantity,a.input_unit,a.output_unit,a.approval_state,CONCAT('••••',RIGHT(linked_sensor.sensor_identifier,4)) AS sensor_identifier,(SELECT nr.reported_name FROM sensor_reported_names nr WHERE nr.sensor_id=linked_sensor.id) AS sensor_name", Join: " LEFT JOIN fuel_sensors linked_fuel_sensor ON linked_fuel_sensor.id=a.fuel_sensor_id LEFT JOIN ble_sensors linked_sensor ON linked_sensor.id=linked_fuel_sensor.sensor_id", Write: []string{"fuel_sensor_id", "tank_profile_id", "revision", "media", "input_quantity", "input_unit", "output_unit"}, Required: []string{"fuel_sensor_id", "revision", "media", "input_quantity", "output_unit"}},
	"tank-profiles":         {Table: "fuel_tank_profiles", Select: "a.id,a.vehicle_id,a.name,inventory.tank_capacity_l AS capacity_l,a.revision,a.effective_from,a.effective_to,linked_vehicle.plate_number,linked_vehicle.vehicle_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id LEFT JOIN vehicle_inventory inventory ON inventory.vehicle_id=a.vehicle_id", Vehicle: "a.vehicle_id", Write: []string{"vehicle_id", "name", "revision", "effective_from", "effective_to"}, Required: []string{"name", "revision", "effective_from"}},
	"users":                 {Table: "users", Select: "a.id,a.username,a.full_name,a.email,a.is_active,a.row_version,a.last_login_at,COALESCE((SELECT r.code FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=a.id LIMIT 1),'viewer') AS role", Search: "a.username", Admin: true},
	"notifications":         {Table: "notifications", Select: "a.id,a.title,a.body,a.target,a.read_at,a.created_at", Time: "a.created_at"},
	"audit":                 {Table: "audit_log", Select: "a.id,a.actor_id,a.action,a.entity,a.entity_id,a.occurred_at", Time: "a.occurred_at", Admin: true},
	"reports":               {Table: "report_jobs", Select: "a.id,a.report_type,a.state,a.created_at,a.expires_at", Time: "a.created_at"},
	"event-types":           {Table: "event_types", Select: "a.id,a.code,a.description", Write: []string{"code", "description"}, Required: []string{"code", "description"}, Admin: true},
	"event-severities":      {Table: "event_severities", Select: "a.id,a.code,a.rank", Write: []string{"code", "rank"}, Required: []string{"code", "rank"}, Admin: true},
	"geofence-assignments":  {Table: "geofence_vehicle_assignments", Select: "a.id,a.geofence_id,a.vehicle_id,a.starts_at,a.ends_at,linked_vehicle.plate_number,linked_vehicle.vehicle_name", Join: " LEFT JOIN vehicles linked_vehicle ON linked_vehicle.id=a.vehicle_id", Vehicle: "a.vehicle_id", Write: []string{"geofence_id", "vehicle_id", "starts_at", "ends_at"}, Required: []string{"geofence_id", "vehicle_id", "starts_at"}},
}

func (s *Server) api(w http.ResponseWriter, r *http.Request, u *Identity) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	if s.maintenance(w, r, u, path) {
		return
	}
	if path == "dashboard/snapshot" {
		s.dashboardSnapshot(w, r)
		return
	}
	if path == "dashboard/fuel-history" {
		s.dashboardFuelHistory(w, r)
		return
	}
	if path == "dashboard/scopes" {
		s.dashboardScopes(w, r, u)
		return
	}
	if path == "notifications/read-all" || (strings.HasPrefix(path, "notifications/") && strings.HasSuffix(path, "/read")) {
		s.notificationReadState(w, r, u, path)
		return
	}
	if path == "vehicles/options" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return
		}
		rows, err := queryRows(r.Context(), s.DB, "SELECT DISTINCT fleet_code FROM vehicles WHERE deleted_at IS NULL AND fleet_code IS NOT NULL AND fleet_code<>'' ORDER BY fleet_code")
		if err != nil {
			databaseError(w, err)
			return
		}
		respond(w, 200, map[string]any{"data": rows})
		return
	}
	if s.fuelEventOperations(w, r, u, path) {
		return
	}
	if s.fuelMonitoringOperations(w, r, path) {
		return
	}
	if path == "sensors" && r.Method == "POST" {
		fail(w, 405, "Sensor ditambahkan otomatis dari kiriman GPS.")
		return
	}
	if s.vehicleInventory(w, r, u, path) {
		return
	}
	if s.remoteOutput(w, r, u, path) {
		return
	}
	if s.deviceDiscovery(w, r, u, path) {
		return
	}
	if s.automaticCalibration(w, r, u, path) {
		return
	}
	if s.simpleSetup(w, r, u, path) {
		return
	}
	if s.bleRegistration(w, r, u, path) {
		return
	}
	if s.reportOperations(w, r, u, path) {
		return
	}
	if s.userMutation(w, r, u, path) {
		return
	}
	if s.tripOperations(w, r, u, path) {
		return
	}
	if s.calibration(w, r, u, path) {
		return
	}
	if s.assignment(w, r, u, path) {
		return
	}
	if path == "fleet/overview" || path == "fleet/statistics" || path == "fuel/overview" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return
		}
		s.overview(w, r)
		return
	}
	if path == "fleet/live" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return
		}
		s.live(w, r)
		return
	}
	if path == "fleet/activity" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return
		}
		s.fleetActivity(w, r)
		return
	}
	if path == "gps/history" {
		path = "gps/positions"
	}
	if path == "devices/gps-quality" {
		path = "gps/quality"
	}
	if strings.HasPrefix(path, "alerts/events/") && strings.HasSuffix(path, "/acknowledge") {
		s.acknowledge(w, r, u, path)
		return
	}
	if strings.HasPrefix(path, "alerts/events/") && strings.HasSuffix(path, "/resolve") {
		s.resolveAlert(w, r, u, path)
		return
	}
	spec, ok := resources[path]
	var id int64
	if !ok {
		parts := strings.Split(path, "/")
		last := parts[len(parts)-1]
		if n, e := parseID(last); e == nil {
			id = n
			path = strings.Join(parts[:len(parts)-1], "/")
			spec, ok = resources[path]
		}
		if !ok && len(parts) == 3 && parts[0] == "vehicles" {
			vehicle, e := parseID(parts[1])
			if e != nil {
				fail(w, 400, "invalid_vehicle")
				return
			}
			target := map[string]string{"telemetry": "gps/positions", "positions": "gps/positions", "trips": "trips", "events": "events", "fuel": "fuel/readings", "devices": "devices", "calibrations": "calibrations"}[parts[2]]
			if target != "" {
				spec, ok = resources[target]
				if target == "devices" {
					spec.Vehicle = "assignment.vehicle_id"
				}
				if target == "calibrations" {
					spec.Join += " JOIN fuel_tank_profiles tp ON tp.id=a.tank_profile_id"
					spec.Vehicle = "tp.vehicle_id"
				}
				r.URL.Query()
				q := r.URL.Query()
				q.Set("vehicle_id", strconv.FormatInt(vehicle, 10))
				r.URL.RawQuery = q.Encode()
				id = 0
				path = target
			}
		}
	}
	if !ok {
		fail(w, 404, "endpoint_not_found")
		return
	}
	if spec.Admin && u.Role != "admin" {
		fail(w, 403, "permission_denied")
		return
	}
	if r.Method == http.MethodDelete && (spec.Table == "vehicles" || spec.Table == "gps_devices") {
		s.deleteInventory(w, r, u, spec.Table, id)
		return
	}
	if r.Method != "GET" {
		s.mutate(w, r, u, spec, id)
		return
	}
	s.list(w, r, u, spec, id)
}
func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	rows, e := s.fleetRows(r.Context())
	if e != nil {
		databaseError(w, e)
		return
	}
	summary := fleetSummary(rows)
	var expires time.Time
	now := time.Now().UTC()
	for _, row := range rows {
		for _, field := range []string{"gps_time", "device_last_received_at", "fuel_received_at", "fuel_measured_at", "sensor_last_received_at"} {
			at := timestamp(row[field])
			if at.IsZero() {
				continue
			}
			at = at.Add(5 * time.Minute)
			if at.After(now) && (expires.IsZero() || at.Before(expires)) {
				expires = at
			}
		}
	}
	if !expires.IsZero() {
		summary["next_freshness_change_at"] = expires
	}

	counters, err := queryRows(r.Context(), s.DB, `SELECT (SELECT COUNT(*) FROM device_raw_messages WHERE received_at>=UTC_DATE()) messages_today,(SELECT COUNT(*) FROM protocol_parse_logs WHERE parse_status='failed' AND processed_at>=UTC_DATE()) parse_errors_today,(SELECT COUNT(*) FROM fuel_readings WHERE validity='valid') valid_fuel_readings`)
	if err != nil {
		databaseError(w, err)
		return
	}
	for key, value := range counters[0] {
		summary[key] = value
	}
	respond(w, 200, map[string]any{"data": summary, "meta": map[string]any{"as_of": time.Now().UTC(), "physical_ignition_state": "NOT_INSTALLED"}})
}
func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	rows, e := s.fleetRows(r.Context())
	if e != nil {
		databaseError(w, e)
		return
	}
	respond(w, 200, map[string]any{"data": rows, "meta": map[string]any{"as_of": time.Now().UTC(), "bounded": false}})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request, u *Identity, spec resource, id int64) {
	if spec.Table == "ble_sensors" && (u.Role == "admin" || u.Role == "operator") {
		spec.Select += ",d.imei AS gps_imei,a.mac AS sensor_mac,st.raw_message_id AS source_raw_id"
	}

	q := r.URL.Query()
	limit := 50
	if q.Has("limit") {
		n, e := strconv.Atoi(q.Get("limit"))
		if e != nil || n < 1 || n > 200 {
			fail(w, 400, "invalid_limit")
			return
		}
		limit = n
	}
	where := []string{"1=1"}
	args := []any{}
	if id > 0 {
		where = append(where, "a.id=?")
		args = append(args, id)
	}
	if q.Get("cursor") != "" {
		n, e := parseID(q.Get("cursor"))
		if e != nil {
			fail(w, 400, "invalid_cursor")
			return
		}
		where = append(where, "a.id<?")
		args = append(args, n)
	}
	if q.Get("vehicle_id") != "" {
		n, e := parseID(q.Get("vehicle_id"))
		if e != nil || spec.Vehicle == "" {
			fail(w, 400, "invalid_vehicle_filter")
			return
		}
		where = append(where, spec.Vehicle+"=?")
		args = append(args, n)
	}
	if q.Get("search") != "" {
		if spec.Search == "" {
			fail(w, 400, "search_not_supported")
			return
		}
		if len(q.Get("search")) > 100 {
			fail(w, 400, "invalid_search")
			return
		}
		where = append(where, spec.Search+" LIKE ?")
		args = append(args, "%"+q.Get("search")+"%")
	}
	if spec.Table == "vehicles" {
		if code := q.Get("fleet_code"); code != "" {
			if len(code) > 100 {
				fail(w, 400, "invalid_fleet_code")
				return
			}
			where = append(where, "a.fleet_code=?")
			args = append(args, code)
		}
		if status := q.Get("movement_status"); status != "" {
			if !map[string]bool{"MOVING": true, "STOPPED": true, "OFFLINE": true, "STALE": true, "GPS_NO_FIX": true, "UNKNOWN_SPEED": true, "UNASSIGNED": true}[status] {
				fail(w, 400, "invalid_movement_status")
				return
			}
			where = append(where, "("+vehicleMovementSQL+")=?")
			args = append(args, status)
		}
	}
	var from, to time.Time
	for _, name := range []string{"from", "to"} {
		if value := q.Get(name); value != "" {
			t, e := time.Parse(time.RFC3339, value)
			if e != nil || spec.Time == "" {
				fail(w, 400, "invalid_date_range")
				return
			}
			op := ">="
			if name == "to" {
				op = "<="
				to = t
			} else {
				from = t
			}
			where = append(where, spec.Time+op+"?")
			args = append(args, t.UTC())
		}
	}
	if !from.IsZero() && !to.IsZero() && (to.Before(from) || to.Sub(from) > 366*24*time.Hour) {
		fail(w, 400, "invalid_date_range")
		return
	}
	if spec.Table == "vehicles" || spec.Table == "gps_devices" || spec.Table == "ble_sensors" || spec.Table == "users" {
		where = append(where, "a.deleted_at IS NULL")
	}
	if spec.Table == "report_jobs" && u.Role != "admin" {
		where = append(where, "a.requested_by=?")
		args = append(args, u.ID)
	}
	if spec.Table == "notifications" {
		where = append(where, "a.user_id=?")
		args = append(args, u.ID)
	}
	args = append(args, limit+1)
	rows, e := queryRows(r.Context(), s.DB, "SELECT "+spec.Select+" FROM "+spec.Table+" a"+spec.Join+" WHERE "+strings.Join(where, " AND ")+" ORDER BY a.id DESC LIMIT ?", args...)
	if e != nil {
		databaseError(w, e)
		return
	}
	if spec.Table == "vehicles" {
		now := time.Now().UTC()
		for _, row := range rows {
			classifyFleet(row, now)
		}
	}
	if id > 0 && spec.Table == "device_raw_messages" && len(rows) > 0 {
		var payload []byte
		if e = s.DB.QueryRowContext(r.Context(), "SELECT payload FROM device_raw_messages WHERE id=?", id).Scan(&payload); e != nil {
			databaseError(w, e)
			return
		}
		rows[0]["payload_redacted"] = regexp.MustCompile(`[0-9]{15}`).ReplaceAllString(string(payload), "[device-redacted]")
	}
	if id > 0 {
		if len(rows) == 0 {
			fail(w, 404, "record_not_found")
			return
		}
		respond(w, 200, map[string]any{"data": rows[0]})
		return
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	var cursor any
	if more {
		cursor = fmt.Sprint(rows[len(rows)-1]["id"])
	}
	respond(w, 200, map[string]any{"data": rows, "page": map[string]any{"has_more": more, "next_cursor": cursor}, "meta": map[string]any{"as_of": time.Now().UTC(), "timezone": "Asia/Jakarta"}})
}
func (s *Server) mutate(w http.ResponseWriter, r *http.Request, u *Identity, spec resource, id int64) {
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return
	}
	if len(spec.Write) == 0 {
		fail(w, 405, "read_only_resource")
		return
	}
	if r.Method != "POST" && r.Method != "PUT" {
		fail(w, 405, "method_not_allowed")
		return
	}
	if (r.Method == "PUT") != (id > 0) {
		fail(w, 400, "invalid_mutation_target")
		return
	}
	var input map[string]any
	if body(w, r, &input) != nil {
		fail(w, 400, "invalid_json")
		return
	}
	if spec.Table == "ble_sensors" && r.Method == "PUT" {
		spec.Write = []string{"is_active"}
	}
	if spec.Table == "gps_devices" && r.Method == "PUT" {
		spec.Write = []string{"model_id", "is_active"}
	}
	allowed := map[string]bool{}
	for _, k := range spec.Write {
		allowed[k] = true
	}
	for k, v := range input {
		if !allowed[k] {
			fail(w, 400, "unknown_field")
			return
		}
		if text, ok := v.(string); ok && len(text) > 1000 {
			fail(w, 400, "field_too_long")
			return
		}
	}
	if spec.Table == "vehicles" {
		for _, key := range []string{"plate_number", "vehicle_name", "fleet_code"} {
			if value, ok := input[key]; ok && value != nil {
				text, ok := value.(string)
				if !ok {
					fail(w, 400, "invalid_vehicle_field")
					return
				}
				text = strings.TrimSpace(text)
				max := 100
				if key == "plate_number" {
					max = 32
				}
				if len(text) > max || (key != "fleet_code" && text == "") {
					fail(w, 400, "invalid_vehicle_field")
					return
				}
				input[key] = text
			}
		}
	}
	if r.Method == "POST" {
		for _, k := range spec.Required {
			v, exists := input[k]
			if !exists || v == nil || v == "" {
				fail(w, 400, "required_field_missing")
				return
			}
		}
	}
	if spec.Table == "gps_devices" {
		if v, ok := input["imei"]; ok {
			value, ok := v.(string)
			if !ok || len(value) != 15 || strings.Trim(value, "0123456789") != "" {
				fail(w, 400, "invalid_device_identifier")
				return
			}
		}
	}
	if spec.Table == "geofences" {
		if err := validateGeofence(input); err != nil {
			fail(w, 400, "invalid_geofence")
			return
		}
	}
	if spec.Table == "vehicles" && r.Method == "POST" {
		input["row_version"] = 1
	}
	if spec.Table == "geofences" && r.Method == "POST" {
		input["row_version"] = 1
	}
	if spec.Table == "fuel_calibrations" {
		input["approval_state"] = "unverified"
	}
	keys := []string{}
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		fail(w, 400, "empty_update")
		return
	}
	values := []any{}
	placeholders := []string{}
	sets := []string{}
	for _, k := range keys {
		v := input[k]
		if _, ok := v.(map[string]any); ok {
			b, _ := json.Marshal(v)
			v = string(b)
		}
		values = append(values, v)
		placeholders = append(placeholders, "?")
		sets = append(sets, "`"+k+"`=?")
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return
	}
	defer tx.Rollback()
	if spec.Table == "fuel_calibrations" && id > 0 {
		var state string
		if e = tx.QueryRowContext(r.Context(), "SELECT approval_state FROM fuel_calibrations WHERE id=? FOR UPDATE", id).Scan(&state); e != nil {
			fail(w, 404, "record_not_found")
			return
		}
		if state == "approved" {
			fail(w, 409, "approved_calibration_is_immutable")
			return
		}
	}
	statement := "INSERT INTO " + spec.Table + " (`" + strings.Join(keys, "`,`") + "`) VALUES(" + strings.Join(placeholders, ",") + ")"
	if id > 0 {
		if spec.Table == "vehicles" {
			sets = append(sets, "row_version=row_version+1")
		}
		statement = "UPDATE " + spec.Table + " SET " + strings.Join(sets, ",") + " WHERE id=?"
		if spec.Table == "vehicles" || spec.Table == "gps_devices" {
			statement += " AND deleted_at IS NULL"
		}
		values = append(values, id)
	}
	result, e := tx.ExecContext(r.Context(), statement, values...)
	if e != nil {
		fail(w, 422, "invalid_or_conflicting_record")
		return
	}
	if id == 0 {
		id, _ = result.LastInsertId()
	} else {
		var exists int
		where := " WHERE id=?"
		if spec.Table == "vehicles" || spec.Table == "gps_devices" {
			where += " AND deleted_at IS NULL"
		}
		if e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+spec.Table+where, id).Scan(&exists); e != nil {
			databaseError(w, e)
			return
		}
		if exists == 0 {
			fail(w, 404, "record_not_found")
			return
		}
	}
	// Sensitive inputs such as IMEI are never copied to audit JSON.
	_, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(3))", u.ID, r.Method, spec.Table, id, token())
	if e != nil {
		databaseError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return
	}
	s.Notify()
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	respond(w, status, map[string]any{"data": map[string]any{"id": id}})
}
func validateGeofence(v map[string]any) error {
	shape, _ := v["shape"].(string)
	if shape == "circle" {
		lat, a := v["center_lat"].(float64)
		lon, b := v["center_lon"].(float64)
		radius, c := v["radius_m"].(float64)
		if !a || !b || !c || lat < -90 || lat > 90 || lon < -180 || lon > 180 || radius <= 0 {
			return fmt.Errorf("invalid circle")
		}
		return nil
	}
	if shape == "polygon" {
		g, ok := v["polygon_geojson"].(map[string]any)
		if !ok || g["type"] != "Polygon" {
			return fmt.Errorf("invalid polygon")
		}
		coords, ok := g["coordinates"].([]any)
		if !ok || len(coords) == 0 {
			return fmt.Errorf("invalid polygon")
		}
		ring, ok := coords[0].([]any)
		if !ok || len(ring) < 4 || len(ring) > 1000 {
			return fmt.Errorf("invalid ring")
		}
		for _, p := range ring {
			pair, ok := p.([]any)
			if !ok || len(pair) != 2 {
				return fmt.Errorf("invalid coordinate")
			}
			lon, a := pair[0].(float64)
			lat, b := pair[1].(float64)
			if !a || !b || lon < -180 || lon > 180 || lat < -90 || lat > 90 {
				return fmt.Errorf("invalid bounds")
			}
		}
		first, _ := json.Marshal(ring[0])
		last, _ := json.Marshal(ring[len(ring)-1])
		if string(first) != string(last) {
			return fmt.Errorf("ring must close")
		}
		return nil
	}
	return fmt.Errorf("shape required")
}
func (s *Server) acknowledge(w http.ResponseWriter, r *http.Request, u *Identity, path string) {
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return
	}
	if u.Role == "viewer" {
		fail(w, 403, "permission_denied")
		return
	}
	parts := strings.Split(path, "/")
	id, e := parseID(parts[2])
	if e != nil {
		fail(w, 400, "invalid_id")
		return
	}
	result, e := s.DB.ExecContext(r.Context(), "UPDATE alert_events SET state='acknowledged',acknowledged_by=?,acknowledged_at=UTC_TIMESTAMP(3) WHERE id=? AND state NOT IN ('resolved','closed','acknowledged')", u.ID, id)
	if e != nil {
		databaseError(w, e)
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 409, "alert_not_actionable")
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"id": id, "state": "acknowledged"}})
}

func (s *Server) resolveAlert(w http.ResponseWriter, r *http.Request, u *Identity, path string) {
	if r.Method != http.MethodPost {
		fail(w, 405, "method_not_allowed")
		return
	}
	if u.Role == "viewer" || s.ReadOnly {
		fail(w, 403, "permission_denied")
		return
	}
	parts := strings.Split(path, "/")
	id, err := parseID(parts[2])
	if err != nil {
		fail(w, 400, "invalid_id")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		databaseError(w, err)
		return
	}
	defer tx.Rollback()
	var previous string
	if err = tx.QueryRowContext(r.Context(), "SELECT state FROM alert_events WHERE id=? FOR UPDATE", id).Scan(&previous); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 404, "not_found")
		} else {
			databaseError(w, err)
		}
		return
	}
	if previous != "active" && previous != "acknowledged" {
		fail(w, 409, "alert_not_actionable")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE alert_events SET state='resolved',resolved_at=UTC_TIMESTAMP(3),row_version=row_version+1 WHERE id=?", id); err != nil {
		databaseError(w, err)
		return
	}
	_, _ = tx.ExecContext(r.Context(), "UPDATE alert_rule_states SET current_alert_id=NULL,row_version=row_version+1 WHERE current_alert_id=?", id)
	_, err = tx.ExecContext(r.Context(), `INSERT INTO alert_audit_trail(alert_id,action,previous_state,new_state,actor_id,actor_type,reason_code,request_id,occurred_at) VALUES(?,?,?,?,?,'USER','USER_RESOLVED',?,UTC_TIMESTAMP(3))`, id, "ALERT_RESOLVED", previous, "resolved", u.ID, fmt.Sprintf("resolve-%d", time.Now().UnixNano()))
	if err != nil {
		databaseError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"id": id, "state": "resolved"}})
}
