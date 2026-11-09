package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"github.com/beego/beego/v2/server/web"
	"log"
	"net/http"
	"os"
	"pastrack/web/models"
	"pastrack/web/ui"
	"strconv"
	"strings"
	"sync"
	"time"
)

var db *sql.DB
var assets = ui.Assets
var sessionSecret string

func env(k, d string) string { return models.Env(k, d) }
func Configure(database *sql.DB) {
	db = database
	sessionSecret = os.Getenv("SESSION_SECRET")
	if len(sessionSecret) < 32 {
		log.Fatal("SESSION_SECRET must be configured")
	}
}
func StartBackground()                        { initPush(); startPushWorker(); startRealtime() }
func Security(next http.Handler) http.Handler { return security(next) }

var query = models.Query

func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://tile.openstreetmap.org https://server.arcgisonline.com; frame-src https://www.openstreetmap.org; connect-src 'self' https://*.googleapis.com https://*.firebaseinstallations.googleapis.com https://*.google-analytics.com https://*.analytics.google.com; worker-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		if !strings.HasPrefix(r.URL.Path, "/assets/") && r.URL.Path != "/healthz" {
			if u := resolveUser(r); u != nil {
				r = r.WithContext(context.WithValue(r.Context(), userContextKey{}, u))
			}
		}
		if r.URL.Path == "/api/realtime" {
			RealtimeStream(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func signature(s string) string {
	h := hmac.New(sha256.New, []byte(sessionSecret))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
func sendAsset(c *web.Controller, name string) {
	if e := ui.Render(c.Ctx.ResponseWriter, name); e != nil {
		c.Abort("500")
	}
}

type Dashboard struct{ web.Controller }

func (c *Dashboard) Get() {
	if !authenticated(c.Ctx.Request) {
		c.Redirect("/login", 302)
		return
	}
	sendAsset(&c.Controller, "index.html")
}

var attempts = struct {
	sync.Mutex
	entries map[string][]time.Time
}{entries: make(map[string][]time.Time)}

func allowLogin(ip string) bool {
	attempts.Lock()
	defer attempts.Unlock()
	now := time.Now()
	for k, v := range attempts.entries {
		if len(v) == 0 || now.Sub(v[len(v)-1]) > 10*time.Minute {
			delete(attempts.entries, k)
		}
	}
	v := attempts.entries[ip]
	var recent []time.Time
	for _, t := range v {
		if now.Sub(t) < 10*time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 15 || len(attempts.entries) > 2048 {
		return false
	}
	attempts.entries[ip] = append(recent, now)
	return true
}

type Login struct{ web.Controller }

func (c *Login) Get() {
	if authenticated(c.Ctx.Request) {
		c.Redirect("/", 302)
		return
	}
	sendAsset(&c.Controller, "login.html")
}

type Logout struct{ web.Controller }

type Health struct{ web.Controller }

func (c *Health) Get() {
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 2*time.Second)
	defer cancel()
	if db.PingContext(ctx) != nil {
		c.Ctx.Output.SetStatus(503)
		c.Ctx.Output.Body([]byte("unavailable"))
		return
	}
	c.Ctx.Output.Body([]byte("ok"))
}
func integer(s string, def, min, max int) int {
	v, e := strconv.Atoi(s)
	if e != nil || v < min || v > max {
		return def
	}
	return v
}

type API struct{ web.Controller }

func (c *API) Get() {
	if !authenticated(c.Ctx.Request) {
		c.Ctx.Output.SetStatus(401)
		c.Data["json"] = map[string]string{"error": "Sesi berakhir. Silakan masuk kembali."}
		c.ServeJSON()
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	device := integer(c.GetString("device"), 0, 0, 2147483647)
	hours := integer(c.GetString("hours"), 24, 1, 720)
	page := integer(c.GetString("page"), 1, 1, 10000)
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	kind := c.Ctx.Input.Param(":kind")
	var result any
	var e error
	switch kind {
	case "fuel-sensors":
		result, e = models.FuelSensorStatus(ctx, int64(device))
	case "vehicle-status":
		result, e = query(ctx, `SELECT v.id,v.plate_number,v.vehicle_name,v.vehicle_type,v.fleet_code,v.is_active,a.make_model,a.production_year,a.fuel_type,a.ownership,a.owner_name,a.color,a.transmission,a.seats,a.vin,a.engine_number,a.insurance_provider,a.insurance_policy,a.insurance_expiry,a.tax_expiry,a.registration_expiry,a.tank_capacity_l,a.notes,a.photo_updated_at,
			(SELECT COUNT(*) FROM gps_devices d WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL) gps_count,
			(SELECT COUNT(*) FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL AND s.deleted_at IS NULL) sensor_count,
			(SELECT GROUP_CONCAT(DISTINCT s.sensor_id ORDER BY s.sensor_id SEPARATOR ', ') FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL AND s.deleted_at IS NULL) sensor_ids,
			(SELECT MAX(s.last_seen_at) FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL AND s.deleted_at IS NULL) sensor_seen_at,
			(SELECT MAX(d.last_seen_at) FROM gps_devices d WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL) gps_report_at,
			p.latitude,p.longitude,p.speed_kmh,p.gps_time,dp.imei position_imei
			FROM vehicles v LEFT JOIN vehicle_assets a ON a.vehicle_id=v.id LEFT JOIN gps_positions p ON p.id=(SELECT pp.id FROM gps_positions pp JOIN gps_devices dd ON dd.id=pp.device_id WHERE dd.vehicle_id=v.id AND dd.deleted_at IS NULL AND pp.latitude BETWEEN -90 AND 90 AND pp.longitude BETWEEN -180 AND 180 AND NOT(pp.latitude=0 AND pp.longitude=0) ORDER BY pp.gps_time DESC,pp.id DESC LIMIT 1)
			LEFT JOIN gps_devices dp ON dp.id=p.device_id WHERE v.deleted_at IS NULL ORDER BY v.plate_number LIMIT 500`)
	case "summary":
		loc := time.FixedZone("WIB", 7*3600)
		now := time.Now().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC()
		result, e = query(ctx, `SELECT (SELECT COUNT(*) FROM gps_devices WHERE deleted_at IS NULL) AS devices,(SELECT COUNT(*) FROM gps_devices WHERE deleted_at IS NULL AND is_active=1 AND last_seen_at>=?) AS online,(SELECT COUNT(*) FROM device_raw_messages WHERE received_at>=?) AS messages_today,(SELECT COUNT(*) FROM device_raw_messages WHERE received_at>=? AND parse_status='failed') AS failed_today,(SELECT MAX(received_at) FROM device_raw_messages) AS latest_message`, time.Now().UTC().Add(-5*time.Minute), today, today)
	case "devices":
		result, e = query(ctx, `SELECT d.id,d.vehicle_id,d.imei,d.model,d.protocol_version,d.firmware_version,d.last_seen_at,d.is_active,v.plate_number,v.vehicle_name,p.latitude,p.longitude,p.speed_kmh,p.gps_time,p.device_send_time,p.heading_deg,(SELECT COUNT(*) FROM device_fuel_sensors s WHERE s.device_id=d.id AND s.deleted_at IS NULL) fuel_sensor_count,(SELECT COUNT(*) FROM device_fuel_sensors s WHERE s.device_id=d.id AND s.deleted_at IS NULL AND EXISTS (SELECT 1 FROM fuel_calibration_points z WHERE z.sensor_pk=s.id AND z.level_percent=0) AND EXISTS (SELECT 1 FROM fuel_calibration_points z WHERE z.sensor_pk=s.id AND z.level_percent=100)) fuel_ready_count,(SELECT COUNT(*) FROM device_fuel_sensors s WHERE s.device_id=d.id AND s.deleted_at IS NULL AND s.calibration_media='water_test') fuel_water_test_count,(SELECT GROUP_CONCAT(s.sensor_id ORDER BY s.sensor_id SEPARATOR ', ') FROM device_fuel_sensors s WHERE s.device_id=d.id AND s.deleted_at IS NULL) fuel_sensor_ids,(SELECT MAX(s.last_seen_at) FROM device_fuel_sensors s WHERE s.device_id=d.id AND s.deleted_at IS NULL) fuel_last_seen_at FROM gps_devices d LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN gps_positions p ON p.id=(SELECT pp.id FROM gps_positions pp WHERE pp.device_id=d.id ORDER BY pp.gps_time DESC,pp.id DESC LIMIT 1) WHERE d.deleted_at IS NULL AND (?=0 OR d.id=?) ORDER BY d.id LIMIT 500`, device, device)
	case "positions":
		where := "p.created_at>=?"
		args := []any{since}
		if device > 0 {
			where += " AND p.device_id=?"
			args = append(args, device)
		}
		args = append(args, (page-1)*50)
		result, e = query(ctx, `SELECT p.id,d.imei,v.plate_number,p.latitude,p.longitude,p.speed_kmh,p.heading_deg,p.altitude_m,p.gps_time,p.device_send_time,p.created_at FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id WHERE `+where+` ORDER BY p.id DESC LIMIT 51 OFFSET ?`, args...)
	case "messages":
		where := "r.received_at>=?"
		args := []any{since}
		if device > 0 {
			where += " AND r.device_id=?"
			args = append(args, device)
		}
		if s := c.GetString("search"); s != "" {
			if len(s) > 100 {
				s = s[:100]
			}
			where += " AND (r.raw_payload LIKE ? OR r.imei LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%")
		}
		if status := c.GetString("status"); status == "success" || status == "partial" || status == "failed" || status == "pending" {
			where += " AND r.parse_status=?"
			args = append(args, status)
		}
		args = append(args, (page-1)*50)
		result, e = query(ctx, `SELECT r.id,r.imei,r.message_type,r.raw_payload,r.parsed_payload,r.received_at,r.parse_status,r.parse_error FROM device_raw_messages r WHERE `+where+` ORDER BY r.id DESC LIMIT 51 OFFSET ?`, args...)
	case "events":
		where := "r.received_at>=?"
		args := []any{since}
		if device > 0 {
			where += " AND e.device_id=?"
			args = append(args, device)
		}
		args = append(args, (page-1)*50)
		result, e = query(ctx, `SELECT e.id,d.imei,e.message_type,e.event_code,e.event_value,e.event_time,r.received_at FROM device_events e JOIN gps_devices d ON d.id=e.device_id LEFT JOIN device_raw_messages r ON r.id=e.raw_message_id WHERE `+where+` ORDER BY e.id DESC LIMIT 51 OFFSET ?`, args...)
	case "fuel":
		where := "received_at>=? AND (message_type IN ('GTBAR','GTERI','GTBAA') OR raw_payload LIKE '%E04E7A648774%')"
		args := []any{since}
		if device > 0 {
			where += " AND device_id=?"
			args = append(args, device)
		}
		result, e = query(ctx, `SELECT id,imei,message_type,received_at,raw_payload,parse_status FROM device_raw_messages WHERE `+where+` ORDER BY id DESC LIMIT 50`, args...)
	default:
		c.Ctx.Output.SetStatus(404)
		c.Data["json"] = map[string]string{"error": "Halaman API tidak ditemukan"}
		c.ServeJSON()
		return
	}
	if e != nil {
		log.Printf("query %s: %v", kind, e)
		c.Ctx.Output.SetStatus(503)
		c.Data["json"] = map[string]string{"error": "Data belum dapat dimuat. Coba lagi sebentar."}
	} else {
		c.Data["json"] = map[string]any{"data": result, "page": page, "updated_at": time.Now().UTC().Format(time.RFC3339)}
	}
	c.ServeJSON()
}
