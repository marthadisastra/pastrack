package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"pastrack/web/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
)

type FuelCalibration struct{ web.Controller }
type fuelPointInput struct {
	Percent float64 `json:"level_percent"`
	Raw     int     `json:"raw_value"`
	Note    string  `json:"note"`
	Media   string  `json:"media"`
}

func (c *FuelCalibration) sensorID() (int64, bool) {
	id, e := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	if e != nil || id < 1 {
		reply(&c.Controller, 400, map[string]string{"error": "Sensor tidak valid."})
		return 0, false
	}
	return id, true
}
func (c *FuelCalibration) readInput() (fuelPointInput, bool) {
	var p fuelPointInput
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 4096)
	d := json.NewDecoder(c.Ctx.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(&struct{}{}) != io.EOF || math.IsNaN(p.Percent) || math.IsInf(p.Percent, 0) || p.Percent < 0 || p.Percent > 100 || p.Raw < 0 || p.Raw > 65535 || len(p.Note) > 200 {
		reply(&c.Controller, 422, map[string]string{"error": "Titik kalibrasi tidak valid."})
		return p, false
	}
	p.Note = strings.TrimSpace(p.Note)
	if p.Media != "" && p.Media != "water_test" && p.Media != "fuel_tank" {
		reply(&c.Controller, 422, map[string]string{"error": "Media kalibrasi tidak valid."})
		return p, false
	}
	return p, true
}

func (c *FuelCalibration) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	id, ok := c.sensorID()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	rows, e := models.FuelSensorStatus(ctx, 0)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Sensor belum dapat dimuat."})
		return
	}
	for _, r := range rows {
		if fmt.Sprint(r["sensor_pk"]) == strconv.FormatInt(id, 10) {
			reply(&c.Controller, 200, map[string]any{"data": r, "csrf_token": csrfToken(c.Ctx.Request)})
			return
		}
	}
	reply(&c.Controller, 404, map[string]string{"error": "Sensor tidak ditemukan."})
}

func (c *FuelCalibration) Post() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id, ok := c.sensorID()
	if !ok {
		return
	}
	p, ok := c.readInput()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Database tidak tersedia."})
		return
	}
	defer tx.Rollback()
	var media string
	e = tx.QueryRowContext(ctx, `SELECT s.calibration_media FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE s.id=? AND s.deleted_at IS NULL AND d.deleted_at IS NULL FOR UPDATE`, id).Scan(&media)
	if e == sql.ErrNoRows {
		reply(&c.Controller, 404, map[string]string{"error": "Sensor tidak ditemukan."})
		return
	}
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Sensor belum dapat dikunci."})
		return
	}
	if p.Media != "" && p.Media != media {
		var count int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM fuel_calibration_points WHERE sensor_pk=?`, id).Scan(&count); e != nil {
			reply(&c.Controller, 503, map[string]string{"error": "Profil kalibrasi belum dapat diperiksa."})
			return
		}
		if count > 0 {
			reply(&c.Controller, 409, map[string]string{"error": "Hapus titik pada profil lama sebelum mengganti media kalibrasi."})
			return
		}
		media = p.Media
	}
	if media == "unverified" {
		media = "water_test"
	}
	if _, e = tx.ExecContext(ctx, `UPDATE device_fuel_sensors SET calibration_media=? WHERE id=?`, media, id); e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Media gagal disimpan."})
		return
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO fuel_calibration_points (sensor_pk,level_percent,raw_value,note,created_at,updated_at) VALUES (?,?,?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE raw_value=VALUES(raw_value),note=VALUES(note),updated_at=VALUES(updated_at)`, id, p.Percent, p.Raw, p.Note)
	if e != nil {
		reply(&c.Controller, 422, map[string]string{"error": "Nilai raw sudah dipakai atau titik tidak valid."})
		return
	}
	points, e := models.FuelPoints(ctx, tx, id)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Titik belum dapat dibaca."})
		return
	}
	if len(points) >= 2 {
		for i := 1; i < len(points); i++ {
			if points[i].Raw <= points[i-1].Raw || points[i].Percent <= points[i-1].Percent {
				reply(&c.Controller, 422, map[string]string{"error": "Nilai raw harus naik searah persentase. Periksa titik kalibrasi."})
				return
			}
		}
	}
	if e = tx.Commit(); e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Kalibrasi gagal disimpan."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"data": points, "media": media})
}
func (c *FuelCalibration) Delete() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id, ok := c.sensorID()
	if !ok {
		return
	}
	point, e := strconv.ParseInt(c.Ctx.Input.Param(":point"), 10, 64)
	if e != nil || point < 1 {
		reply(&c.Controller, 400, map[string]string{"error": "Titik tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	result, e := db.ExecContext(ctx, `DELETE FROM fuel_calibration_points WHERE id=? AND sensor_pk=?`, point, id)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Titik gagal dihapus."})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		reply(&c.Controller, 404, map[string]string{"error": "Titik tidak ditemukan."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]string{"message": "Titik dihapus."})
}
