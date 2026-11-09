package platform

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type fuelInvestigation struct {
	State         string   `json:"state"`
	Note          string   `json:"note"`
	ReceiptNumber string   `json:"receipt_number"`
	StationName   string   `json:"station_name"`
	ReceiptLiters *float64 `json:"receipt_liters"`
	ReceiptAmount *float64 `json:"receipt_amount"`
	Version       int64    `json:"row_version"`
}

func validFuelInvestigation(p fuelInvestigation) bool {
	if p.State != "open" && p.State != "investigating" && p.State != "resolved" && p.State != "dismissed" {
		return false
	}
	if len(p.Note) > 8000 || len(p.ReceiptNumber) > 100 || len(p.StationName) > 200 || p.Version < 0 {
		return false
	}
	for _, v := range []*float64{p.ReceiptLiters, p.ReceiptAmount} {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 999999999) {
			return false
		}
	}
	return p.State != "resolved" && p.State != "dismissed" || strings.TrimSpace(p.Note) != ""
}

const fuelEventSelect = `SELECT e.id,e.vehicle_id,e.fuel_sensor_id,e.event_type,e.delta_l,e.event_at,e.confidence,e.rule_version,v.plate_number,v.vehicle_name,
 a.received_at start_at,b.received_at end_at,a.fuel_liters start_liters,b.fuel_liters end_liters,a.raw_measurement start_raw,b.raw_measurement end_raw,a.raw_unit,
 TIMESTAMPDIFF(SECOND,a.received_at,b.received_at) duration_seconds,
 COALESCE(i.state,'open') state,COALESCE(i.note,'') note,COALESCE(i.receipt_number,'') receipt_number,COALESCE(i.station_name,'') station_name,i.receipt_liters,i.receipt_amount,COALESCE(i.row_version,0) row_version,i.updated_at,usr.full_name investigator,
 p.latitude,p.longitude,p.speed_kmh,p.gps_time,p.satellite_count,p.hdop,
 st.temperature_c,st.rssi,s.model,
 (SELECT COUNT(*) FROM geofence_events g WHERE g.vehicle_id=e.vehicle_id AND g.event_at BETWEEN a.received_at AND b.received_at) geofence_event_count
 FROM fuel_events e JOIN vehicles v ON v.id=e.vehicle_id JOIN fuel_readings a ON a.id=e.start_reading_id JOIN fuel_readings b ON b.id=e.end_reading_id
 LEFT JOIN fuel_event_investigations i ON i.event_id=e.id LEFT JOIN users usr ON usr.id=i.updated_by
 LEFT JOIN sensor_telemetry st ON st.id=b.telemetry_id LEFT JOIN fuel_sensors fs ON fs.id=e.fuel_sensor_id LEFT JOIN ble_sensors s ON s.id=fs.sensor_id
 LEFT JOIN gps_positions p ON p.id=(SELECT pp.id FROM gps_positions pp WHERE pp.vehicle_id=e.vehicle_id AND pp.gps_time<=e.event_at AND pp.gps_time>=e.event_at-INTERVAL 5 MINUTE AND pp.fix_status='valid' AND pp.validation_state='validated' ORDER BY pp.gps_time DESC,pp.id DESC LIMIT 1) `

func (s *Server) fuelEventOperations(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	if path == "fuel/events/workspace" {
		s.fuelEventWorkspace(w, r)
		return true
	}
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[0] != "fuel" || parts[1] != "events" || (parts[3] != "investigation" && parts[3] != "bap") {
		return false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_event")
		return true
	}
	if parts[3] == "bap" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		rows, e := queryRows(r.Context(), s.DB, fuelEventSelect+" WHERE e.id=?", id)
		if e != nil {
			databaseError(w, e)
			return true
		}
		if len(rows) == 0 {
			fail(w, 404, "event_not_found")
			return true
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="BAP-Fuel-EVT-%d.html"`, id))
		bapTemplate.Execute(w, map[string]any{"Event": rows[0], "Generated": time.Now().In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006 15:04:05 WIB"), "Actor": u.Name})
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	var p fuelInvestigation
	if body(w, r, &p) != nil || !validFuelInvestigation(p) {
		fail(w, 422, "invalid_investigation_or_receipt")
		return true
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return true
	}
	defer tx.Rollback()
	var eventType string
	if e = tx.QueryRowContext(r.Context(), "SELECT event_type FROM fuel_events WHERE id=? FOR UPDATE", id).Scan(&eventType); e == sql.ErrNoRows {
		fail(w, 404, "event_not_found")
		return true
	} else if e != nil {
		databaseError(w, e)
		return true
	}
	if eventType != "refueling" && (p.ReceiptNumber != "" || p.StationName != "" || p.ReceiptLiters != nil || p.ReceiptAmount != nil) {
		fail(w, 422, "receipt_requires_refueling_event")
		return true
	}
	var version int64
	e = tx.QueryRowContext(r.Context(), "SELECT row_version FROM fuel_event_investigations WHERE event_id=?", id).Scan(&version)
	if e != nil && e != sql.ErrNoRows {
		databaseError(w, e)
		return true
	}
	if version != p.Version {
		fail(w, 409, "investigation_changed_reload")
		return true
	}
	_, e = tx.ExecContext(r.Context(), `INSERT INTO fuel_event_investigations(event_id,state,note,receipt_number,station_name,receipt_liters,receipt_amount,row_version,updated_by) VALUES(?,?,?,?,?,?,?,1,?) ON DUPLICATE KEY UPDATE state=VALUES(state),note=VALUES(note),receipt_number=VALUES(receipt_number),station_name=VALUES(station_name),receipt_liters=VALUES(receipt_liters),receipt_amount=VALUES(receipt_amount),row_version=row_version+1,updated_by=VALUES(updated_by),updated_at=UTC_TIMESTAMP(3)`, id, p.State, p.Note, p.ReceiptNumber, p.StationName, p.ReceiptLiters, p.ReceiptAmount, u.ID)
	if e == nil {
		_, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'investigate','fuel_events',?,?,UTC_TIMESTAMP(3))", u.ID, id, token())
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		databaseError(w, e)
		return true
	}
	respond(w, 200, map[string]any{"data": map[string]any{"row_version": version + 1}})
	return true
}
func (s *Server) fuelEventWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed")
		return
	}
	vehicle, e := strconv.ParseInt(r.URL.Query().Get("vehicle_id"), 10, 64)
	if r.URL.Query().Get("vehicle_id") == "" {
		vehicle = 0
		e = nil
	}
	if e != nil || vehicle < 0 {
		fail(w, 400, "select_vehicle")
		return
	}
	from, e := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if e != nil {
		fail(w, 400, "invalid_period")
		return
	}
	to, e := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if e != nil || !to.After(from) || to.Sub(from) > 31*24*time.Hour {
		fail(w, 400, "period_maximum_31_days")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result := map[string]any{"generated_at": time.Now().UTC(), "from": from, "to": to, "basis": "received_at; event_at; GPS time (WIB)", "idle_available": false, "all_vehicles": vehicle == 0}
	if vehicle > 0 {
		ignition, err := s.mapIgnitions(ctx, vehicle, from.UTC(), to.UTC())
		if err != nil {
			databaseError(w, err)
			return
		}
		result["ignitions"] = ignition
	}
	queries := []struct {
		key, query string
		args       []any
		limit      int
	}{
		{"events", fuelEventSelect + " WHERE (?=0 OR e.vehicle_id=?) AND e.event_at>=? AND e.event_at<? ORDER BY e.event_at DESC,e.id DESC LIMIT 1001", []any{vehicle, vehicle, from.UTC(), to.UTC()}, 1000},
		{"readings", `SELECT f.id,f.vehicle_id,f.fuel_sensor_id,f.received_at,f.reading_at,f.fuel_liters,f.fuel_percent,f.validity,f.invalid_reason,f.raw_measurement,f.raw_unit,f.calibration_id,c.media,vi.tank_capacity_l capacity_l,st.fuel_height_raw,st.temperature_c,st.rssi FROM fuel_readings f LEFT JOIN fuel_calibrations c ON c.id=f.calibration_id LEFT JOIN vehicle_inventory vi ON vi.vehicle_id=f.vehicle_id LEFT JOIN sensor_telemetry st ON st.id=f.telemetry_id WHERE f.vehicle_id=? AND f.received_at>=? AND f.received_at<? ORDER BY f.received_at,f.id LIMIT 20001`, []any{vehicle, from.UTC(), to.UTC()}, 20000},
		{"positions", `SELECT id,vehicle_id,latitude,longitude,speed_kmh,gps_time,satellite_count,hdop,device_id FROM gps_positions WHERE vehicle_id=? AND gps_time>=? AND gps_time<? AND fix_status='valid' AND validation_state='validated' ORDER BY gps_time,id LIMIT 20001`, []any{vehicle, from.UTC(), to.UTC()}, 20000},
		{"sensors", `SELECT DISTINCT f.id fuel_sensor_id,b.model,n.reported_name AS sensor_name,t.name tank_name,vi.tank_capacity_l capacity_l,f.decoding_state FROM fuel_sensors f JOIN ble_sensors b ON b.id=f.sensor_id LEFT JOIN sensor_reported_names n ON n.sensor_id=b.id LEFT JOIN fuel_tank_profiles t ON t.id=f.tank_profile_id LEFT JOIN vehicle_inventory vi ON vi.vehicle_id=? WHERE t.vehicle_id=? OR EXISTS(SELECT 1 FROM fuel_readings r WHERE r.fuel_sensor_id=f.id AND r.vehicle_id=?)`, []any{vehicle, vehicle, vehicle}, 0},
		{"geofences", `SELECT id,name,shape,center_lat,center_lon,radius_m,polygon_geojson FROM geofences WHERE is_active=1 LIMIT 1001`, nil, 1000},
		{"rules", `SELECT r.id,r.name,r.version,r.configuration,t.code FROM alert_rules r JOIN event_types t ON t.id=r.event_type_id WHERE r.enabled=1 AND t.code IN ('fuel_drop','refueling') LIMIT 100`, nil, 0},
		{"analytics", `SELECT consumed_l,distance_m,coverage_percent,period_start,period_end FROM fuel_consumption_summaries WHERE vehicle_id=? AND period_start>=? AND period_end<=? ORDER BY period_start LIMIT 1000`, []any{vehicle, from.UTC(), to.UTC()}, 0},
	}
	for _, q := range queries {
		if vehicle == 0 && q.key != "events" && q.key != "rules" {
			result[q.key] = []map[string]any{}
			continue
		}
		rows, e := queryRows(ctx, s.DB, q.query, q.args...)
		if e != nil {
			databaseError(w, e)
			return
		}
		if q.limit > 0 && len(rows) > q.limit {
			fail(w, 422, "too_many_records_reduce_period")
			return
		}
		result[q.key] = rows
	}
	respond(w, 200, map[string]any{"data": result})
}

var bapTemplate = template.Must(template.New("bap").Parse(`<!doctype html><html lang="id"><meta charset="utf-8"><title>BAP Fuel Event {{.Event.id}}</title><style>body{font:14px Arial;max-width:850px;margin:40px auto;color:#172334}h1{color:#174fc5}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ccd4df;padding:10px;text-align:left}p{white-space:pre-wrap}@media print{body{margin:12mm}}</style><h1>PasTrack · BAP Investigasi BBM</h1><p>EVT-{{.Event.id}} · Dibuat {{.Generated}} oleh {{.Actor}}</p><table><tr><th>Kendaraan</th><td>{{.Event.plate_number}}</td></tr><tr><th>Kejadian (UTC)</th><td>{{.Event.event_at}} · {{.Event.event_type}}</td></tr><tr><th>Perubahan BBM</th><td>{{.Event.delta_l}} L</td></tr><tr><th>Periode sumber (UTC)</th><td>{{.Event.start_at}} — {{.Event.end_at}}</td></tr><tr><th>Liter awal / akhir</th><td>{{.Event.start_liters}} / {{.Event.end_liters}}</td></tr><tr><th>Koordinat GPS terdekat ≤5 menit</th><td>{{.Event.latitude}}, {{.Event.longitude}}</td></tr><tr><th>Aturan / confidence</th><td>{{.Event.rule_version}} / {{.Event.confidence}}</td></tr><tr><th>Status / revisi</th><td>{{.Event.state}} / {{.Event.row_version}}</td></tr><tr><th>Petugas terakhir</th><td>{{.Event.investigator}}</td></tr><tr><th>Struk manual / SPBU</th><td>{{.Event.receipt_number}} / {{.Event.station_name}}</td></tr><tr><th>Liter / nominal struk</th><td>{{.Event.receipt_liters}} L / Rp {{.Event.receipt_amount}}</td></tr></table><h2>Catatan investigasi</h2><p>{{.Event.note}}</p><p>Perubahan sensor merupakan indikasi anomali, bukan bukti pencurian. Struk merupakan input manual, bukan hasil verifikasi SPBU. Dokumen ini adalah snapshot pada waktu unduhan; buka HTML di browser dan gunakan Cetak / Simpan PDF.</p><p>Petugas: ____________________     Pemeriksa: ____________________</p></html>`))
