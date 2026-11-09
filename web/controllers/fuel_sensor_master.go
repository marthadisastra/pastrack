package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beego/beego/v2/server/web"
	"github.com/go-sql-driver/mysql"
)

type FuelSensorMaster struct{ web.Controller }
type fuelSensorInput struct {
	SensorID   string `json:"sensor_id"`
	Model      string `json:"model"`
	Active     bool   `json:"is_active"`
	RowVersion int64  `json:"row_version"`
}

var sensorMAC = regexp.MustCompile(`^[0-9A-F]{12}$`)

func (c *FuelSensorMaster) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	archived := c.GetString("archived") == "1"
	where := "s.deleted_at IS NULL"
	if archived {
		where = "s.deleted_at IS NOT NULL"
	}
	args := []any{}
	if search := strings.TrimSpace(c.GetString("search")); search != "" {
		if len(search) > 100 {
			search = search[:100]
		}
		where += " AND (s.sensor_id LIKE ? OR s.model LIKE ? OR d.imei LIKE ? OR v.plate_number LIKE ?)"
		for i := 0; i < 4; i++ {
			args = append(args, "%"+search+"%")
		}
	}
	page := integer(c.GetString("page"), 1, 1, 10000)
	args = append(args, (page-1)*50)
	rows, e := query(ctx, `SELECT s.id,s.sensor_id,s.model,s.device_id,s.reported_device_id,s.last_seen_at,s.observed_raw_value,s.observed_at,s.calibration_media,s.is_active,s.deleted_at,s.row_version,d.imei,v.id vehicle_id,v.plate_number,rd.imei reported_imei,(SELECT COUNT(*) FROM fuel_calibration_points p WHERE p.sensor_pk=s.id) point_count FROM device_fuel_sensors s LEFT JOIN gps_devices d ON d.id=s.device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN gps_devices rd ON rd.id=s.reported_device_id WHERE `+where+` ORDER BY s.id DESC LIMIT 51 OFFSET ?`, args...)
	if e != nil {
		log.Printf("sensor master list: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "Daftar sensor belum dapat dimuat."})
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": rows, "page": page, "csrf_token": csrfToken(c.Ctx.Request)})
}
func sensorSnapshot(ctx context.Context, tx *sql.Tx, id int64) (map[string]any, error) {
	var b []byte
	e := tx.QueryRowContext(ctx, `SELECT JSON_OBJECT('id',id,'sensor_id',sensor_id,'model',model,'device_id',device_id,'reported_device_id',reported_device_id,'is_active',is_active,'deleted_at',deleted_at,'row_version',row_version,'calibration_media',calibration_media) FROM device_fuel_sensors WHERE id=? FOR UPDATE`, id).Scan(&b)
	if e != nil {
		return nil, e
	}
	var v map[string]any
	e = json.Unmarshal(b, &v)
	return v, e
}
func (c *FuelSensorMaster) Post()    { c.mutate("create") }
func (c *FuelSensorMaster) Put()     { c.mutate("update") }
func (c *FuelSensorMaster) Delete()  { c.mutate("archive") }
func (c *FuelSensorMaster) Restore() { c.mutate("restore") }
func (c *FuelSensorMaster) mutate(action string) {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	id := int64(integer(c.Ctx.Input.Param(":id"), 0, 0, 2147483647))
	if (action == "create" && id != 0) || (action != "create" && id == 0) {
		reply(&c.Controller, 400, map[string]string{"error": "ID sensor tidak valid."})
		return
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 4096)
	decoder := json.NewDecoder(c.Ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var input fuelSensorInput
	if decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		reply(&c.Controller, 400, map[string]string{"error": "Form sensor tidak valid."})
		return
	}
	input.SensorID = strings.ToUpper(strings.TrimSpace(input.SensorID))
	input.Model = strings.TrimSpace(input.Model)
	if action == "create" || action == "update" {
		if !sensorMAC.MatchString(input.SensorID) || input.Model == "" || utf8.RuneCountInString(input.Model) > 32 || strings.ContainsAny(input.Model, "\x00\r\n") {
			reply(&c.Controller, 422, map[string]string{"error": "MAC harus 12 digit heksadesimal; model dan GPS harus valid."})
			return
		}
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 10*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Database tidak tersedia."})
		return
	}
	defer tx.Rollback()
	var before map[string]any
	if action != "create" {
		before, e = sensorSnapshot(ctx, tx, id)
		if e == sql.ErrNoRows {
			reply(&c.Controller, 404, map[string]string{"error": "Sensor tidak ditemukan."})
			return
		}
		if e != nil {
			reply(&c.Controller, 503, map[string]string{"error": "Sensor belum dapat dikunci."})
			return
		}
		if input.RowVersion != int64(before["row_version"].(float64)) {
			reply(&c.Controller, 409, map[string]string{"error": "Data sensor berubah. Muat ulang dahulu."})
			return
		}
		if (action == "restore") != (before["deleted_at"] != nil) {
			reply(&c.Controller, 409, map[string]string{"error": "Status arsip sudah berubah."})
			return
		}
	}
	if action == "create" || action == "update" {
		if action == "update" && before["sensor_id"] != input.SensorID {
			var n int
			if e = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM fuel_calibration_points WHERE sensor_pk=?)+(SELECT COUNT(*) FROM device_raw_messages WHERE id=(SELECT last_raw_message_id FROM device_fuel_sensors WHERE id=?))`, id, id).Scan(&n); e != nil {
				reply(&c.Controller, 503, map[string]string{"error": "Riwayat sensor belum dapat diperiksa."})
				return
			}
			if n > 0 {
				reply(&c.Controller, 409, map[string]string{"error": "MAC sensor yang sudah memiliki pembacaan atau kalibrasi tidak dapat diganti."})
				return
			}
		}
	}
	switch action {
	case "create":
		var result sql.Result
		result, e = tx.ExecContext(ctx, `INSERT INTO device_fuel_sensors(sensor_id,model,is_active,last_seen_at) VALUES(?,?,?,NULL)`, input.SensorID, input.Model, input.Active)
		if e == nil {
			id, e = result.LastInsertId()
		}
	case "update":
		_, e = tx.ExecContext(ctx, `UPDATE device_fuel_sensors SET sensor_id=?,model=?,is_active=?,row_version=row_version+1 WHERE id=?`, input.SensorID, input.Model, input.Active, id)
	case "archive":
		_, e = tx.ExecContext(ctx, `UPDATE device_fuel_sensors SET deleted_at=UTC_TIMESTAMP(3),is_active=0,row_version=row_version+1 WHERE id=?`, id)
	case "restore":
		_, e = tx.ExecContext(ctx, `UPDATE device_fuel_sensors SET deleted_at=NULL,is_active=0,row_version=row_version+1 WHERE id=?`, id)
	}
	if e != nil {
		var me *mysql.MySQLError
		if errors.As(e, &me) && me.Number == 1062 {
			reply(&c.Controller, 409, map[string]string{"error": "MAC sensor sudah terdaftar."})
			return
		}
		log.Printf("sensor master %s: %v", action, e)
		reply(&c.Controller, 503, map[string]string{"error": "Sensor gagal disimpan."})
		return
	}
	after, e := sensorSnapshot(ctx, tx, id)
	req := requestID()
	if e == nil {
		e = audit(ctx, tx, action, "fuel_sensors", id, before, after, req)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		log.Printf("sensor master commit: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "Perubahan sensor belum dapat dikonfirmasi."})
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"data": after, "request_id": req})
}
