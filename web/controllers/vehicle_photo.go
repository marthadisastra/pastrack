package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
)

type VehiclePhoto struct{ web.Controller }

func (c *VehiclePhoto) vehicleID() (int64, bool) {
	id, err := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id < 1 {
		reply(&c.Controller, 400, map[string]string{"error": "ID kendaraan tidak valid."})
		return 0, false
	}
	return id, true
}
func vehiclePhotoPath(id int64) string {
	return filepath.Join(env("VEHICLE_PHOTO_DIR", "/etc/pastrack/vehicle-photos"), fmt.Sprintf("%d.img", id))
}
func (c *VehiclePhoto) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	id, ok := c.vehicleID()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	var mime sql.NullString
	err := db.QueryRowContext(ctx, `SELECT a.photo_mime FROM vehicle_assets a JOIN vehicles v ON v.id=a.vehicle_id WHERE a.vehicle_id=? AND v.deleted_at IS NULL`, id).Scan(&mime)
	if err != nil || !mime.Valid {
		http.NotFound(c.Ctx.ResponseWriter, c.Ctx.Request)
		return
	}
	f, err := os.Open(vehiclePhotoPath(id))
	if err != nil {
		http.NotFound(c.Ctx.ResponseWriter, c.Ctx.Request)
		return
	}
	defer f.Close()
	c.Ctx.Output.Header("Content-Type", mime.String)
	c.Ctx.Output.Header("Cache-Control", "private, max-age=300")
	c.Ctx.Output.Header("X-Content-Type-Options", "nosniff")
	io.Copy(c.Ctx.ResponseWriter, f)
}
func (c *VehiclePhoto) Post() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id, ok := c.vehicleID()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 10*time.Second)
	defer cancel()
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT 1 FROM vehicles WHERE id=? AND deleted_at IS NULL`, id).Scan(&exists); err != nil {
		reply(&c.Controller, 404, map[string]string{"error": "Kendaraan tidak ditemukan."})
		return
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 5<<20)
	f, _, err := c.Ctx.Request.FormFile("photo")
	if err != nil {
		reply(&c.Controller, 400, map[string]string{"error": "Pilih foto maksimal 5 MB."})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (5<<20)+1))
	if err != nil || len(data) == 0 || len(data) > 5<<20 {
		reply(&c.Controller, 400, map[string]string{"error": "Foto maksimal 5 MB."})
		return
	}
	mime := http.DetectContentType(data)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		reply(&c.Controller, 422, map[string]string{"error": "Gunakan foto JPG, PNG, atau WebP."})
		return
	}
	dir := filepath.Dir(vehiclePhotoPath(id))
	if err = os.MkdirAll(dir, 0700); err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Penyimpanan foto tidak tersedia."})
		return
	}
	tmp, err := os.CreateTemp(dir, ".vehicle-*.tmp")
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Foto gagal disimpan."})
		return
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(tmp.Name(), vehiclePhotoPath(id))
	}
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Foto gagal disimpan."})
		return
	}
	_, err = db.ExecContext(ctx, `INSERT INTO vehicle_assets (vehicle_id,photo_mime,photo_updated_at,updated_at) VALUES (?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE photo_mime=VALUES(photo_mime),photo_updated_at=VALUES(photo_updated_at),updated_at=VALUES(updated_at)`, id, mime)
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Metadata foto gagal disimpan."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"message": "Foto kendaraan berhasil disimpan."})
}
func (c *VehiclePhoto) Delete() {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id, ok := c.vehicleID()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `UPDATE vehicle_assets SET photo_mime=NULL,photo_updated_at=NULL,updated_at=UTC_TIMESTAMP(3) WHERE vehicle_id=?`, id)
	if err != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Foto gagal dihapus."})
		return
	}
	if err = os.Remove(vehiclePhotoPath(id)); err != nil && !os.IsNotExist(err) {
		reply(&c.Controller, 503, map[string]string{"error": "File foto gagal dihapus."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"message": "Foto kendaraan dihapus."})
}
