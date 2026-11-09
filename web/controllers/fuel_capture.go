package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
)

type FuelCapture struct{ web.Controller }

func (c *FuelCapture) Post() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id, err := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	level := c.Ctx.Input.Param(":level")
	if err != nil || id < 1 || (level != "empty" && level != "full") {
		reply(&c.Controller, 400, map[string]string{"error": "Sensor atau kondisi kalibrasi tidak valid."})
		return
	}
	var request struct {
		Media string `json:"media"`
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 1024)
	decoder := json.NewDecoder(c.Ctx.Request.Body)
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(&request)
	if decodeErr != io.EOF && decodeErr != nil {
		reply(&c.Controller, 422, map[string]string{"error": "Media kalibrasi tidak valid."})
		return
	}
	if decodeErr == nil && decoder.Decode(&struct{}{}) != io.EOF {
		reply(&c.Controller, 422, map[string]string{"error": "Permintaan kalibrasi tidak valid."})
		return
	}
	if request.Media != "" && request.Media != "water_test" && request.Media != "fuel_tank" {
		reply(&c.Controller, 422, map[string]string{"error": "Media kalibrasi tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Database tidak tersedia."})
		return
	}
	defer tx.Rollback()
	var raw sql.NullInt64
	var at sql.NullTime
	var media string
	err = tx.QueryRowContext(ctx, `SELECT s.observed_raw_value,s.observed_at,s.calibration_media FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE s.id=? AND s.deleted_at IS NULL AND s.is_active=1 AND s.reported_device_id=s.device_id AND d.deleted_at IS NULL AND d.is_active=1 FOR UPDATE`, id).Scan(&raw, &at, &media)
	if err == sql.ErrNoRows {
		reply(&c.Controller, 404, map[string]string{"error": "Sensor aktif tidak ditemukan."})
		return
	}
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Pembacaan sensor belum dapat diperiksa."})
		return
	}
	if !raw.Valid || !at.Valid || time.Since(at.Time.UTC()) > 60*time.Second || time.Since(at.Time.UTC()) < -10*time.Second {
		reply(&c.Controller, 422, map[string]string{"error": "Pembacaan sensor tidak tersedia atau lebih dari 60 detik. Tunggu laporan baru."})
		return
	}
	if request.Media != "" && request.Media != media {
		var pointCount int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM fuel_calibration_points WHERE sensor_pk=?`, id).Scan(&pointCount); err != nil {
			reply(&c.Controller, 503, map[string]string{"error": "Profil kalibrasi belum dapat diperiksa."})
			return
		}
		if pointCount > 0 {
			reply(&c.Controller, 409, map[string]string{"error": "Hapus titik profil lama sebelum mengganti media kalibrasi."})
			return
		}
		media = request.Media
	}
	if media == "unverified" {
		media = "water_test"
	}
	percent := 0
	if level == "full" {
		percent = 100
	}
	var other sql.NullInt64
	otherPercent := 100 - percent
	err = tx.QueryRowContext(ctx, `SELECT raw_value FROM fuel_calibration_points WHERE sensor_pk=? AND level_percent=?`, id, otherPercent).Scan(&other)
	if err != nil && err != sql.ErrNoRows {
		reply(&c.Controller, 503, map[string]string{"error": "Titik kalibrasi belum dapat diperiksa."})
		return
	}
	if other.Valid && ((percent == 0 && raw.Int64 >= other.Int64) || (percent == 100 && raw.Int64 <= other.Int64)) {
		reply(&c.Controller, 422, map[string]string{"error": "Nilai penuh harus lebih tinggi dari kosong. Pertahankan sensor diam dan ulangi saat stabil."})
		return
	}
	var intermediate sql.NullInt64
	query := `SELECT MIN(raw_value) FROM fuel_calibration_points WHERE sensor_pk=? AND level_percent>0 AND level_percent<100`
	if percent == 100 {
		query = `SELECT MAX(raw_value) FROM fuel_calibration_points WHERE sensor_pk=? AND level_percent>0 AND level_percent<100`
	}
	if err = tx.QueryRowContext(ctx, query, id).Scan(&intermediate); err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Titik antara belum dapat diperiksa."})
		return
	}
	if intermediate.Valid && ((percent == 0 && raw.Int64 >= intermediate.Int64) || (percent == 100 && raw.Int64 <= intermediate.Int64)) {
		reply(&c.Controller, 422, map[string]string{"error": "Nilai ini bertabrakan dengan titik antara. Periksa tabel kalibrasi."})
		return
	}
	note := fmt.Sprintf("Ditandai %s dari laporan sensor %s WIB", map[string]string{"empty": "kosong", "full": "penuh"}[level], at.Time.UTC().Add(7*time.Hour).Format("02-01-2006 15:04:05"))
	_, err = tx.ExecContext(ctx, `INSERT INTO fuel_calibration_points(sensor_pk,level_percent,raw_value,note,created_at,updated_at) VALUES(?,?,?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE raw_value=VALUES(raw_value),note=VALUES(note),updated_at=VALUES(updated_at)`, id, percent, raw.Int64, note)
	if err != nil {
		reply(&c.Controller, 422, map[string]string{"error": "Nilai raw bentrok dengan titik lain. Periksa tabel kalibrasi."})
		return
	}
	_, err = tx.ExecContext(ctx, `UPDATE device_fuel_sensors SET calibration_media=? WHERE id=?`, media, id)
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Profil sensor gagal disimpan."})
		return
	}
	if err = tx.Commit(); err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Titik kalibrasi gagal disimpan."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"level_percent": percent, "raw_value": raw.Int64, "observed_at": at.Time.UTC(), "media": media})
}
