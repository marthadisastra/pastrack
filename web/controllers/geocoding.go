package controllers

import (
	"context"
	"encoding/json"
	"github.com/beego/beego/v2/server/web"
	"io"
	"math"
	"net/http"
	"pastrack/web/models"
	"time"
)

type TrackingAddress struct{ web.Controller }

func (c *TrackingAddress) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	reply(&c.Controller, 200, map[string]any{"configured": models.GeoReady(), "provider": "Geoapify", "csrf_token": csrfToken(c.Ctx.Request)})
}
func (c *TrackingAddress) Post() {
	if !authorize(&c.Controller, true) {
		return
	}
	var in struct {
		PositionID int64 `json:"position_id"`
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 1024)
	d := json.NewDecoder(c.Ctx.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(&struct{}{}) != io.EOF || in.PositionID < 1 {
		reply(&c.Controller, 400, map[string]string{"error": "Titik GPS tidak valid"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 12*time.Second)
	defer cancel()
	var lat, lon float64
	e := db.QueryRowContext(ctx, `SELECT p.latitude,p.longitude FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id WHERE p.id=? AND d.deleted_at IS NULL`, in.PositionID).Scan(&lat, &lon)
	if e != nil || math.IsNaN(lat) || math.IsNaN(lon) || math.Abs(lat) > 90 || math.Abs(lon) > 180 || (lat == 0 && lon == 0) {
		reply(&c.Controller, 404, map[string]string{"error": "Titik GPS tidak ditemukan atau koordinat tidak valid"})
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": models.ResolveAddress(ctx, lat, lon)})
}
