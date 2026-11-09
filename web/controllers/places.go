package controllers

import (
	"context"
	"encoding/json"
	"github.com/beego/beego/v2/server/web"
	"io"
	"net/http"
	"pastrack/web/models"
	"time"
)

type TrackingPlaces struct{ web.Controller }

func (c *TrackingPlaces) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	rows, e := models.Places(ctx)
	if e != nil {
		reply(&c.Controller, 500, map[string]string{"error": "Daftar lokasi tidak dapat dimuat"})
		return
	}
	u := currentUser(ctx)
	reply(&c.Controller, 200, map[string]any{"data": rows, "csrf_token": csrfToken(c.Ctx.Request), "can_manage": u.Role == "admin" || u.Role == "operator", "geocoding_configured": models.GeoReady()})
}
func (c *TrackingPlaces) Post() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	var p models.Place
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 4096)
	d := json.NewDecoder(c.Ctx.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(&struct{}{}) != io.EOF {
		reply(&c.Controller, 400, map[string]string{"error": "Data lokasi tidak valid"})
		return
	}
	if e := models.ValidatePlace(&p); e != nil {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		reply(&c.Controller, 500, map[string]string{"error": "Lokasi belum dapat disimpan"})
		return
	}
	defer tx.Rollback()
	var before any
	action := "create"
	if p.ID > 0 {
		action = "update"
		var raw []byte
		if e = tx.QueryRowContext(ctx, `SELECT JSON_OBJECT('id',id,'name',name,'latitude',latitude,'longitude',longitude,'radius_m',radius_m,'row_version',row_version) FROM tracking_places WHERE id=? FOR UPDATE`, p.ID).Scan(&raw); e != nil {
			reply(&c.Controller, 404, map[string]string{"error": "Lokasi tidak ditemukan"})
			return
		}
		_ = json.Unmarshal(raw, &before)
	}
	if e = models.SavePlace(ctx, tx, &p); e != nil {
		reply(&c.Controller, 409, map[string]string{"error": e.Error()})
		return
	}
	if e = audit(ctx, tx, action, "tracking_places", p.ID, before, p, "tracking-place"); e == nil {
		e = tx.Commit()
	}
	if e != nil {
		reply(&c.Controller, 500, map[string]string{"error": "Lokasi belum dapat disimpan"})
		return
	}
	publishRealtime("places")
	reply(&c.Controller, 200, map[string]any{"data": p, "message": "Lokasi disimpan"})
}
