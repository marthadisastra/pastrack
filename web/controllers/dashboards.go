package controllers

import (
	"context"
	"errors"
	"github.com/beego/beego/v2/server/web"
	"log"
	"pastrack/web/models"
	"time"
)

type AnalyticsDashboard struct{ web.Controller }

func (c *AnalyticsDashboard) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	kind := c.Ctx.Input.Param(":kind")
	switch kind {
	case "vehicles", "gps", "activity", "fuel":
	default:
		reply(&c.Controller, 404, map[string]string{"error": "Dashboard tidak ditemukan"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 12*time.Second)
	defer cancel()
	data, e := models.Analytics(ctx, kind, integer(c.GetString("hours"), 24, 1, 720))
	if errors.Is(e, models.ErrVehicleAnalysisLimit) {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	if e != nil {
		log.Printf("Dashboard %s: %v", kind, e)
		reply(&c.Controller, 500, map[string]string{"error": "Dashboard tidak dapat dimuat"})
		return
	}
	reply(&c.Controller, 200, data)
}
