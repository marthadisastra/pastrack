package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beego/beego/v2/server/web"
	"github.com/go-sql-driver/mysql"
)

var imeiPattern = regexp.MustCompile(`^[0-9]{15}$`)

type Master struct{ web.Controller }
type MasterInput struct {
	Plate      string            `json:"plate_number"`
	Name       string            `json:"vehicle_name"`
	Type       string            `json:"vehicle_type"`
	Fleet      string            `json:"fleet_code"`
	Active     bool              `json:"is_active"`
	IMEI       string            `json:"imei"`
	Model      string            `json:"model"`
	Protocol   string            `json:"protocol_name"`
	Version    string            `json:"protocol_version"`
	Firmware   string            `json:"firmware_version"`
	VehicleID  *int64            `json:"vehicle_id"`
	Metadata   json.RawMessage   `json:"metadata"`
	RowVersion int64             `json:"row_version"`
	Asset      VehicleAssetInput `json:"asset"`
}

type VehicleAssetInput struct {
	MakeModel          string   `json:"make_model"`
	Year               *int     `json:"production_year"`
	FuelType           string   `json:"fuel_type"`
	Ownership          string   `json:"ownership"`
	OwnerName          string   `json:"owner_name"`
	Color              string   `json:"color"`
	Transmission       string   `json:"transmission"`
	Seats              *int     `json:"seats"`
	VIN                string   `json:"vin"`
	EngineNumber       string   `json:"engine_number"`
	InsuranceProvider  string   `json:"insurance_provider"`
	InsurancePolicy    string   `json:"insurance_policy"`
	InsuranceExpiry    string   `json:"insurance_expiry"`
	TaxExpiry          string   `json:"tax_expiry"`
	RegistrationExpiry string   `json:"registration_expiry"`
	TankCapacity       *float64 `json:"tank_capacity_l"`
	Notes              string   `json:"notes"`
}

func requestID() string {
	var b [12]byte
	_, e := rand.Read(b[:])
	if e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func csrfToken(r *http.Request) string {
	c, e := r.Cookie("pastrack_session")
	if e != nil {
		return ""
	}
	return signature("csrf:" + c.Value)
}
func authorize(c *web.Controller, mutation bool) bool {
	if !authenticated(c.Ctx.Request) {
		reply(c, 401, map[string]string{"error": "Sesi berakhir. Silakan masuk kembali."})
		return false
	}
	if mutation {
		r := c.Ctx.Request
		if !hmac.Equal([]byte(r.Header.Get("X-CSRF-Token")), []byte(csrfToken(r))) || r.Header.Get("X-CSRF-Token") == "" {
			reply(c, 403, map[string]string{"error": "Token keamanan tidak valid. Muat ulang halaman."})
			return false
		}
		expected := "https://" + r.Host
		if env("COOKIE_SECURE", "false") != "true" {
			expected = "http://" + r.Host
		}
		if r.Header.Get("Origin") != expected {
			reply(c, 403, map[string]string{"error": "Origin permintaan tidak valid."})
			return false
		}
	}
	return true
}
func reply(c *web.Controller, status int, value any) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = value
	c.ServeJSON()
}
func entityTable(s string) (string, error) {
	switch s {
	case "vehicles":
		return "vehicles", nil
	case "devices":
		return "gps_devices", nil
	}
	return "", fmt.Errorf("master tidak ditemukan")
}
func (c *Master) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	table, e := entityTable(c.Ctx.Input.Param(":entity"))
	if e != nil {
		reply(&c.Controller, 404, map[string]string{"error": e.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	archived := c.GetString("archived") == "1"
	where := "m.deleted_at IS NULL"
	if archived {
		where = "m.deleted_at IS NOT NULL"
	}
	args := []any{}
	if s := strings.TrimSpace(c.GetString("search")); s != "" {
		if len(s) > 100 {
			s = s[:100]
		}
		if table == "vehicles" {
			where += " AND (m.plate_number LIKE ? OR m.vehicle_name LIKE ? OR m.fleet_code LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%", "%"+s+"%")
		} else {
			where += " AND (m.imei LIKE ? OR m.model LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%")
		}
	}
	page := integer(c.GetString("page"), 1, 1, 10000)
	args = append(args, (page-1)*50)
	columns := "m.*"
	join := ""
	if table == "gps_devices" {
		columns += ",v.plate_number,v.vehicle_name,(SELECT GROUP_CONCAT(s.sensor_id ORDER BY s.sensor_id SEPARATOR ', ') FROM device_fuel_sensors s WHERE s.device_id=m.id AND s.deleted_at IS NULL) sensor_ids,(SELECT COUNT(*) FROM device_fuel_sensors s WHERE s.device_id=m.id AND s.deleted_at IS NULL) sensor_count"
		join = " LEFT JOIN vehicles v ON v.id=m.vehicle_id"
	} else {
		columns += ",(SELECT COUNT(*) FROM gps_devices d WHERE d.vehicle_id=m.id AND d.deleted_at IS NULL) AS device_count,a.make_model,a.production_year,a.fuel_type,a.ownership,a.owner_name,a.color,a.transmission,a.seats,a.vin,a.engine_number,a.insurance_provider,a.insurance_policy,a.insurance_expiry,a.tax_expiry,a.registration_expiry,a.tank_capacity_l,a.notes,a.photo_mime,a.photo_updated_at"
		join = " LEFT JOIN vehicle_assets a ON a.vehicle_id=m.id"
	}
	rows, e := query(ctx, "SELECT "+columns+" FROM "+table+" m"+join+" WHERE "+where+" ORDER BY m.id DESC LIMIT 51 OFFSET ?", args...)
	if e != nil {
		log.Printf("master list: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "Master data belum dapat dimuat."})
		return
	}
	options, e := query(ctx, "SELECT id,plate_number,vehicle_name,is_active FROM vehicles WHERE deleted_at IS NULL ORDER BY plate_number LIMIT 2000")
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Daftar kendaraan belum dapat dimuat."})
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": rows, "vehicles": options, "page": page, "csrf_token": csrfToken(c.Ctx.Request)})
}
func validateMaster(entity string, m *MasterInput) error {
	m.Plate = strings.ToUpper(strings.Join(strings.Fields(m.Plate), " "))
	m.Name = strings.TrimSpace(m.Name)
	m.Type = strings.TrimSpace(m.Type)
	m.Fleet = strings.TrimSpace(m.Fleet)
	m.IMEI = strings.TrimSpace(m.IMEI)
	m.Model = strings.TrimSpace(m.Model)
	m.Protocol = strings.TrimSpace(m.Protocol)
	m.Version = strings.TrimSpace(m.Version)
	m.Firmware = strings.TrimSpace(m.Firmware)
	for _, p := range []struct {
		s string
		n int
	}{{m.Plate, 20}, {m.Name, 100}, {m.Type, 50}, {m.Fleet, 50}, {m.Model, 50}, {m.Protocol, 30}, {m.Version, 30}, {m.Firmware, 50}} {
		if utf8.RuneCountInString(p.s) > p.n || strings.ContainsAny(p.s, "\x00\r\n") {
			return errors.New("Nilai terlalu panjang atau berisi karakter yang tidak diizinkan.")
		}
	}
	if entity == "vehicles" {
		if m.Plate == "" || m.Name == "" || m.Type == "" {
			return errors.New("Nomor polisi, nama, dan jenis kendaraan wajib diisi.")
		}
		if err := validateAsset(&m.Asset); err != nil {
			return err
		}
	} else {
		if !imeiPattern.MatchString(m.IMEI) {
			return errors.New("IMEI wajib berupa tepat 15 digit angka.")
		}
		if m.Model == "" || m.Protocol == "" {
			return errors.New("Model dan protokol perangkat wajib diisi.")
		}
		if m.VehicleID != nil && *m.VehicleID <= 0 {
			return errors.New("Kendaraan tidak valid.")
		}
		if len(m.Metadata) > 0 && string(m.Metadata) != "null" {
			var v map[string]any
			if json.Unmarshal(m.Metadata, &v) != nil || v == nil {
				return errors.New("Metadata wajib berupa JSON object.")
			}
		}
	}
	return nil
}
func validateAsset(a *VehicleAssetInput) error {
	for _, field := range []struct {
		value *string
		limit int
	}{
		{&a.MakeModel, 120}, {&a.FuelType, 40}, {&a.Ownership, 80}, {&a.OwnerName, 120}, {&a.Color, 60}, {&a.Transmission, 60}, {&a.VIN, 60}, {&a.EngineNumber, 60}, {&a.InsuranceProvider, 120}, {&a.InsurancePolicy, 100}, {&a.Notes, 2000},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if utf8.RuneCountInString(*field.value) > field.limit || strings.ContainsAny(*field.value, "\x00") {
			return errors.New("Data aset terlalu panjang atau tidak valid.")
		}
	}
	if a.Year != nil && (*a.Year < 1900 || *a.Year > time.Now().Year()+1) {
		return errors.New("Tahun kendaraan tidak valid.")
	}
	if a.Seats != nil && (*a.Seats < 1 || *a.Seats > 200) {
		return errors.New("Jumlah kursi tidak valid.")
	}
	if a.TankCapacity != nil && (*a.TankCapacity <= 0 || *a.TankCapacity > 100000 || math.IsNaN(*a.TankCapacity) || math.IsInf(*a.TankCapacity, 0)) {
		return errors.New("Kapasitas tangki tidak valid.")
	}
	for _, value := range []string{a.InsuranceExpiry, a.TaxExpiry, a.RegistrationExpiry} {
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return errors.New("Tanggal asuransi, pajak, atau STNK tidak valid.")
		}
	}
	return nil
}
func audit(ctx context.Context, tx *sql.Tx, action, entity string, id any, before, after any, req string) error {
	encode := func(v any) any {
		if v == nil {
			return nil
		}
		b, e := json.Marshal(v)
		if e != nil {
			return nil
		}
		return string(b)
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO pastrack_audit_log (occurred_at,actor,action,entity,entity_id,before_data,after_data,request_id) VALUES (?,?,?,?,?,?,?,?)`, time.Now().UTC(), actorName(ctx), action, entity, id, encode(before), encode(after), req)
	return e
}
func rowSnapshot(ctx context.Context, tx *sql.Tx, table string, id int64) (map[string]any, error) {
	r := tx.QueryRowContext(ctx, "SELECT JSON_OBJECT('id',id,'is_active',is_active,'row_version',row_version,'deleted_at',deleted_at,"+func() string {
		if table == "vehicles" {
			return "'plate_number',plate_number,'vehicle_name',vehicle_name,'vehicle_type',vehicle_type,'fleet_code',fleet_code"
		}
		return "'imei',imei,'model',model,'protocol_name',protocol_name,'protocol_version',protocol_version,'firmware_version',firmware_version,'vehicle_id',vehicle_id,'metadata',metadata"
	}()+") FROM "+table+" WHERE id=? FOR UPDATE", id)
	var b []byte
	if e := r.Scan(&b); e != nil {
		return nil, e
	}
	var v map[string]any
	e := json.Unmarshal(b, &v)
	if e == nil && table == "vehicles" {
		var asset sql.NullString
		e = tx.QueryRowContext(ctx, `SELECT JSON_OBJECT('make_model',make_model,'production_year',production_year,'fuel_type',fuel_type,'ownership',ownership,'owner_name',owner_name,'color',color,'transmission',transmission,'seats',seats,'vin',vin,'engine_number',engine_number,'insurance_provider',insurance_provider,'insurance_policy',insurance_policy,'insurance_expiry',insurance_expiry,'tax_expiry',tax_expiry,'registration_expiry',registration_expiry,'tank_capacity_l',tank_capacity_l,'notes',notes,'photo_mime',photo_mime,'photo_updated_at',photo_updated_at) FROM vehicle_assets WHERE vehicle_id=?`, id).Scan(&asset)
		if e == sql.ErrNoRows {
			e = nil
		} else if e == nil && asset.Valid {
			var detail map[string]any
			e = json.Unmarshal([]byte(asset.String), &detail)
			v["asset"] = detail
		}
	}
	return v, e
}
func (c *Master) Post()    { c.mutate("create") }
func (c *Master) Put()     { c.mutate("update") }
func (c *Master) Delete()  { c.mutate("archive") }
func (c *Master) Restore() { c.mutate("restore") }
func (c *Master) mutate(action string) {
	if !authorize(&c.Controller, true) {
		return
	}
	if !allowRole(&c.Controller, "admin", "operator") {
		return
	}
	entity := c.Ctx.Input.Param(":entity")
	table, e := entityTable(entity)
	if e != nil {
		reply(&c.Controller, 404, map[string]string{"error": e.Error()})
		return
	}
	id := int64(integer(c.Ctx.Input.Param(":id"), 0, 0, 2147483647))
	if action == "create" && id != 0 {
		reply(&c.Controller, 405, map[string]string{"error": "Gunakan PUT untuk memperbarui master."})
		return
	}
	if action != "create" && id == 0 {
		reply(&c.Controller, 400, map[string]string{"error": "ID master tidak valid."})
		return
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 16384)
	decoder := json.NewDecoder(c.Ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var input MasterInput
	if decoder.Decode(&input) != nil {
		reply(&c.Controller, 400, map[string]string{"error": "Isi formulir tidak valid."})
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		reply(&c.Controller, 400, map[string]string{"error": "JSON tidak valid."})
		return
	}
	if action == "create" || action == "update" {
		if e = validateMaster(entity, &input); e != nil {
			reply(&c.Controller, 422, map[string]string{"error": e.Error()})
			return
		}
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 10*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Database belum tersedia."})
		return
	}
	defer tx.Rollback()
	fail := func(status int, msg string) { reply(&c.Controller, status, map[string]string{"error": msg}) }
	var before map[string]any
	if action != "create" {
		before, e = rowSnapshot(ctx, tx, table, id)
		if e == sql.ErrNoRows {
			fail(404, "Data tidak ditemukan.")
			return
		}
		if e != nil {
			fail(503, "Data belum dapat dikunci untuk perubahan.")
			return
		}
		version, _ := before["row_version"].(float64)
		if input.RowVersion != int64(version) {
			fail(409, "Data telah berubah. Muat ulang sebelum menyimpan.")
			return
		}
		if (action == "restore") != (before["deleted_at"] != nil) {
			fail(409, "Status arsip telah berubah. Muat ulang data.")
			return
		}
	}
	if entity == "vehicles" && action == "archive" {
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM gps_devices WHERE vehicle_id=? AND deleted_at IS NULL", id).Scan(&n); e != nil {
			fail(503, "Relasi perangkat belum dapat diperiksa.")
			return
		}
		if n > 0 {
			fail(409, "Kendaraan masih memiliki perangkat. Pindahkan atau arsipkan perangkat terlebih dahulu.")
			return
		}
	}
	if entity == "devices" && (action == "create" || action == "update" || action == "restore") {
		vehicleID := input.VehicleID
		if action == "restore" {
			if v, ok := before["vehicle_id"].(float64); ok {
				x := int64(v)
				vehicleID = &x
			}
		}
		if vehicleID != nil {
			var v int64
			if e = tx.QueryRowContext(ctx, "SELECT id FROM vehicles WHERE id=? AND deleted_at IS NULL FOR UPDATE", *vehicleID).Scan(&v); e != nil {
				fail(422, "Kendaraan tidak ditemukan atau telah diarsipkan.")
				return
			}
		}
		if action == "update" && before["imei"] != input.IMEI {
			var n int
			if e = tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM device_raw_messages WHERE device_id=?)+(SELECT COUNT(*) FROM gps_positions WHERE device_id=?)", id, id).Scan(&n); e != nil {
				fail(503, "Riwayat perangkat belum dapat diperiksa.")
				return
			}
			if n > 0 {
				fail(409, "IMEI perangkat dengan riwayat tidak dapat diganti. Daftarkan perangkat pengganti sebagai data baru.")
				return
			}
		}
	}
	if entity == "devices" && action == "archive" {
		var linked int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM device_fuel_sensors WHERE device_id=? AND deleted_at IS NULL`, id).Scan(&linked); e != nil {
			fail(503, "Relasi sensor belum dapat diperiksa.")
			return
		}
		if linked > 0 {
			fail(409, "GPS masih memiliki sensor BBM. Pindahkan atau lepas relasi sensor dahulu.")
			return
		}
	}
	if action == "archive" {
		_, e = tx.ExecContext(ctx, "UPDATE "+table+" SET deleted_at=?,is_active=0,row_version=row_version+1 WHERE id=?", time.Now().UTC(), id)
	} else if action == "restore" {
		_, e = tx.ExecContext(ctx, "UPDATE "+table+" SET deleted_at=NULL,is_active=0,row_version=row_version+1 WHERE id=?", id)
	} else {
		var cols string
		var args []any
		if entity == "vehicles" {
			cols = "plate_number=?,vehicle_name=?,vehicle_type=?,fleet_code=?,is_active=?"
			args = []any{input.Plate, input.Name, input.Type, input.Fleet, input.Active}
		} else {
			cols = "imei=?,model=?,protocol_name=?,protocol_version=?,firmware_version=?,vehicle_id=?,metadata=?,is_active=?"
			var metadata any
			if len(input.Metadata) > 0 && string(input.Metadata) != "null" {
				metadata = string(input.Metadata)
			}
			args = []any{input.IMEI, input.Model, input.Protocol, input.Version, input.Firmware, input.VehicleID, metadata, input.Active}
		}
		if action == "create" {
			var res sql.Result
			res, e = tx.ExecContext(ctx, "INSERT INTO "+table+" SET "+cols, args...)
			if e == nil {
				id, e = res.LastInsertId()
			}
		} else {
			args = append(args, id)
			_, e = tx.ExecContext(ctx, "UPDATE "+table+" SET "+cols+",row_version=row_version+1 WHERE id=?", args...)
		}
	}
	if e != nil {
		var me *mysql.MySQLError
		if errors.As(e, &me) && me.Number == 1062 {
			fail(409, "Nomor polisi atau IMEI sudah terdaftar, termasuk data dalam arsip.")
			return
		}
		log.Printf("master %s: %v", action, e)
		fail(503, "Perubahan gagal disimpan; tidak ada perubahan yang diterapkan.")
		return
	}
	if entity == "vehicles" && (action == "create" || action == "update") {
		a := input.Asset
		optional := func(s string) any {
			if s == "" {
				return nil
			}
			return s
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO vehicle_assets (vehicle_id,make_model,production_year,fuel_type,ownership,owner_name,color,transmission,seats,vin,engine_number,insurance_provider,insurance_policy,insurance_expiry,tax_expiry,registration_expiry,tank_capacity_l,notes,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE make_model=VALUES(make_model),production_year=VALUES(production_year),fuel_type=VALUES(fuel_type),ownership=VALUES(ownership),owner_name=VALUES(owner_name),color=VALUES(color),transmission=VALUES(transmission),seats=VALUES(seats),vin=VALUES(vin),engine_number=VALUES(engine_number),insurance_provider=VALUES(insurance_provider),insurance_policy=VALUES(insurance_policy),insurance_expiry=VALUES(insurance_expiry),tax_expiry=VALUES(tax_expiry),registration_expiry=VALUES(registration_expiry),tank_capacity_l=VALUES(tank_capacity_l),notes=VALUES(notes),updated_at=VALUES(updated_at)`, id, optional(a.MakeModel), a.Year, optional(a.FuelType), optional(a.Ownership), optional(a.OwnerName), optional(a.Color), optional(a.Transmission), a.Seats, optional(a.VIN), optional(a.EngineNumber), optional(a.InsuranceProvider), optional(a.InsurancePolicy), optional(a.InsuranceExpiry), optional(a.TaxExpiry), optional(a.RegistrationExpiry), a.TankCapacity, optional(a.Notes))
		if e != nil {
			log.Printf("vehicle asset write: %v", e)
			fail(503, "Data aset gagal disimpan.")
			return
		}
	}
	after, e := rowSnapshot(ctx, tx, table, id)
	req := requestID()
	if e == nil {
		e = audit(ctx, tx, action, entity, id, before, after, req)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		log.Printf("master audit/commit: %v", e)
		fail(503, "Perubahan belum dapat dikonfirmasi. Muat ulang sebelum mencoba lagi.")
		return
	}
	publishRealtime("master")
	reply(&c.Controller, 200, map[string]any{"data": after, "request_id": req, "message": "Perubahan berhasil disimpan."})
}
