package platform

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

func (s *Server) vehicleInventory(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "vehicles" {
		return false
	}
	var id int64
	var err error
	if len(parts) == 3 && (parts[2] == "photo" || parts[2] == "inventory") {
		id, err = parseID(parts[1])
		if err != nil {
			fail(w, 400, "invalid_vehicle")
			return true
		}
	} else if path != "vehicles/inventory" {
		return false
	}
	if len(parts) == 3 && parts[2] == "photo" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		var photo []byte
		var mime sql.NullString
		err = s.DB.QueryRowContext(r.Context(), "SELECT i.photo,i.photo_mime FROM vehicles v LEFT JOIN vehicle_inventory i ON i.vehicle_id=v.id WHERE v.id=? AND v.deleted_at IS NULL", id).Scan(&photo, &mime)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				fail(w, 404, "vehicle_not_found")
			} else {
				databaseError(w, err)
			}
			return true
		}
		if len(photo) == 0 {
			photo, _ = assets.ReadFile("ui/vehicle-default.jpg")
			mime.String = "image/jpeg"
		}
		w.Header().Set("Content-Type", mime.String)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-cache")
		http.ServeContent(w, r, "vehicle", time.Time{}, bytes.NewReader(photo))
		return true
	}
	if r.Method != "POST" && r.Method != "PUT" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if (r.Method == "PUT") != (id > 0) {
		fail(w, 400, "invalid_mutation_target")
		return true
	}
	if s.ReadOnly || u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	var input struct {
		Code     string   `json:"vehicle_code"`
		Name     string   `json:"vehicle_name"`
		Plate    string   `json:"plate_number"`
		Fleet    string   `json:"fleet_code"`
		Capacity *float64 `json:"tank_capacity_l"`
		Photo    *string  `json:"photo_base64"`
		Version  int64    `json:"row_version"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 3*1024*1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		fail(w, 400, "invalid_vehicle_form")
		return true
	}
	if decoder.Decode(new(any)) != io.EOF {
		fail(w, 400, "invalid_vehicle_form")
		return true
	}
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Plate = strings.TrimSpace(input.Plate)
	input.Fleet = strings.TrimSpace(input.Fleet)
	if input.Name == "" || len(input.Name) > 100 || input.Plate == "" || len(input.Plate) > 32 || len(input.Code) > 64 || len(input.Fleet) > 64 || input.Capacity != nil && (math.IsNaN(*input.Capacity) || math.IsInf(*input.Capacity, 0) || *input.Capacity <= 0 || *input.Capacity > 999999999) {
		fail(w, 400, "invalid_vehicle_field")
		return true
	}
	var photo []byte
	var mime string
	if input.Photo != nil && *input.Photo != "" {
		photo, err = base64.StdEncoding.DecodeString(*input.Photo)
		if err != nil || len(photo) > 2*1024*1024 {
			fail(w, 400, "Foto maksimal 2 MB dalam format JPG atau PNG.")
			return true
		}
		config, format, e := image.DecodeConfig(bytes.NewReader(photo))
		if e != nil || (format != "jpeg" && format != "png") || config.Width < 1 || config.Height < 1 || config.Width > 5000 || config.Height > 5000 {
			fail(w, 400, "Foto JPG atau PNG tidak valid.")
			return true
		}
		mime = "image/" + format
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		databaseError(w, err)
		return true
	}
	defer tx.Rollback()
	if id > 0 {
		var version int64
		err = tx.QueryRowContext(r.Context(), "SELECT row_version FROM vehicles WHERE id=? AND deleted_at IS NULL FOR UPDATE", id).Scan(&version)
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 404, "vehicle_not_found")
			return true
		}
		if err != nil {
			databaseError(w, err)
			return true
		}
		if version != input.Version {
			fail(w, 409, "Data kendaraan berubah. Muat ulang sebelum menyimpan.")
			return true
		}
		_, err = tx.ExecContext(r.Context(), "UPDATE vehicles SET vehicle_name=?,plate_number=?,fleet_code=?,row_version=row_version+1 WHERE id=?", input.Name, input.Plate, nullableText(input.Fleet), id)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(r.Context(), "INSERT INTO vehicles(vehicle_name,plate_number,fleet_code,row_version) VALUES(?,?,?,1)", input.Name, input.Plate, nullableText(input.Fleet))
		if err == nil {
			id, err = result.LastInsertId()
		}
	}
	if err != nil {
		fail(w, 422, "Kode atau plat nomor sudah digunakan, atau data tidak valid.")
		return true
	}
	_, err = tx.ExecContext(r.Context(), "INSERT INTO vehicle_inventory(vehicle_id,vehicle_code,tank_capacity_l) VALUES(?,?,?) ON DUPLICATE KEY UPDATE vehicle_code=VALUES(vehicle_code),tank_capacity_l=VALUES(tank_capacity_l)", id, nullableText(input.Code), input.Capacity)
	if err != nil {
		fail(w, 422, "Kode kendaraan sudah digunakan atau kapasitas tidak valid.")
		return true
	}
	if input.Photo != nil {
		if len(photo) == 0 {
			_, err = tx.ExecContext(r.Context(), "UPDATE vehicle_inventory SET photo=NULL,photo_mime=NULL,photo_updated_at=UTC_TIMESTAMP(3) WHERE vehicle_id=?", id)
		} else {
			_, err = tx.ExecContext(r.Context(), "UPDATE vehicle_inventory SET photo=?,photo_mime=?,photo_updated_at=UTC_TIMESTAMP(3) WHERE vehicle_id=?", photo, mime, id)
		}
		if err != nil {
			databaseError(w, err)
			return true
		}
	}
	_, err = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,?,'vehicles',?,?,UTC_TIMESTAMP(3))", u.ID, r.Method, id, token())
	if err != nil {
		databaseError(w, err)
		return true
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return true
	}
	s.Notify()
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	respond(w, status, map[string]any{"data": map[string]any{"id": id}})
	return true
}
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
