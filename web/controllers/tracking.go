package controllers

import (
	"context"
	"database/sql"
	"errors"
	"github.com/beego/beego/v2/server/web"
	"log"
	"pastrack/web/models"
	"strconv"
	"time"
)

type Tracking struct{ web.Controller }

func trackingRange(start, end string) (time.Time, time.Time, error) {
	a, e := time.Parse(time.RFC3339, start)
	if e != nil {
		return a, a, errors.New("Waktu mulai tidak valid")
	}
	b, e := time.Parse(time.RFC3339, end)
	if e != nil {
		return a, b, errors.New("Waktu akhir tidak valid")
	}
	if !b.After(a) || b.Sub(a) > 31*24*time.Hour {
		return a, b, errors.New("Rentang waktu harus berurutan dan maksimal 31 hari")
	}
	return a.UTC(), b.UTC(), nil
}
func (c *Tracking) getDevice() {
	fail := func(status int, message string) {
		c.Ctx.Output.SetStatus(status)
		c.Data["json"] = map[string]string{"error": message}
		c.ServeJSON()
	}
	if !authenticated(c.Ctx.Request) {
		fail(401, "Sesi berakhir")
		return
	}
	id, e := strconv.ParseInt(c.GetString("device"), 10, 64)
	if e != nil || id < 1 {
		fail(400, "Pilih satu perangkat")
		return
	}
	start, end, e := trackingRange(c.GetString("start"), c.GetString("end"))
	if e != nil {
		fail(400, e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 15*time.Second)
	defer cancel()
	var imei, model string
	var plate sql.NullString
	e = db.QueryRowContext(ctx, `SELECT d.imei,d.model,v.plate_number FROM gps_devices d LEFT JOIN vehicles v ON v.id=d.vehicle_id WHERE d.id=? AND d.deleted_at IS NULL`, id).Scan(&imei, &model, &plate)
	if e == sql.ErrNoRows {
		fail(404, "Perangkat tidak ditemukan")
		return
	}
	if e != nil {
		log.Printf("tracking device: %v", e)
		fail(500, "Data perangkat tidak dapat dimuat")
		return
	}
	rows, e := query(ctx, `SELECT id,latitude,longitude,speed_kmh,heading_deg,altitude_m,gps_time,device_send_time,created_at FROM gps_positions WHERE device_id=? AND gps_time>=? AND gps_time<=? ORDER BY gps_time,id LIMIT 20001`, id, start, end)
	if e != nil {
		log.Printf("tracking positions: %v", e)
		fail(500, "Riwayat tidak dapat dimuat")
		return
	}
	if len(rows) > 20000 {
		fail(422, "Lebih dari 20.000 titik. Persempit rentang waktu untuk menampilkan rute lengkap.")
		return
	}
	c.Data["json"] = map[string]any{"data": rows, "device": map[string]any{"id": id, "imei": imei, "model": model, "plate_number": plate.String}, "start": start, "end": end, "basis": "gps_time", "total": len(rows), "generated_at": time.Now().UTC(), "stops": models.DetectStops(rows), "stop_rules": map[string]int{"radius_m": models.StopRadiusMeters, "min_seconds": models.StopMinimumSeconds, "max_gap_seconds": models.StopMaximumGapSeconds}, "geocoding_configured": models.GeoReady()}
	c.ServeJSON()
}

// Vehicle-based tracking is the UI entry point; device queries remain compatible with existing integrations.
func (c *Tracking) Get() {
	if c.GetString("device") != "" {
		c.getDevice()
		return
	}
	if !authorize(&c.Controller, false) {
		return
	}
	vehicle := int64(0)
	if value := c.GetString("vehicle"); value != "" {
		var e error
		vehicle, e = strconv.ParseInt(value, 10, 64)
		if e != nil || vehicle < 0 || vehicle > 2147483647 {
			reply(&c.Controller, 400, map[string]string{"error": "Kendaraan tidak valid"})
			return
		}
	}
	options := c.GetString("options") == "1"
	var start, end time.Time
	var e error
	if !options {
		start, end, e = trackingRange(c.GetString("start"), c.GetString("end"))
		if e != nil {
			reply(&c.Controller, 400, map[string]string{"error": e.Error()})
			return
		}
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 20*time.Second)
	defer cancel()
	data, e := models.LoadFleetTracking(ctx, vehicle, start, end, options)
	if errors.Is(e, models.ErrTrackingFleetLimit) {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	if errors.Is(e, sql.ErrNoRows) {
		reply(&c.Controller, 404, map[string]string{"error": "Kendaraan tidak ditemukan"})
		return
	}
	if e != nil {
		log.Printf("tracking fleet: %v", e)
		reply(&c.Controller, 500, map[string]string{"error": "Rute kendaraan tidak dapat dimuat"})
		return
	}
	reply(&c.Controller, 200, data)
}
