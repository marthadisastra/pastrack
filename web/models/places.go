package models

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

type Place struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    int     `json:"radius_m"`
	Version   int64   `json:"row_version"`
}

func ValidatePlace(p *Place) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || utf8.RuneCountInString(p.Name) > 100 {
		return errors.New("Nama lokasi wajib diisi, maksimal 100 karakter")
	}
	if math.IsNaN(p.Latitude) || math.IsInf(p.Latitude, 0) || math.IsNaN(p.Longitude) || math.IsInf(p.Longitude, 0) || math.Abs(p.Latitude) > 90 || math.Abs(p.Longitude) > 180 || (p.Latitude == 0 && p.Longitude == 0) {
		return errors.New("Koordinat lokasi tidak valid")
	}
	if p.Radius < 25 || p.Radius > 1000 {
		return errors.New("Radius harus antara 25 dan 1.000 meter")
	}
	if p.ID < 0 {
		return errors.New("ID lokasi tidak valid")
	}
	return nil
}
func Places(ctx context.Context) ([]Place, error) {
	rows, e := DB.QueryContext(ctx, `SELECT id,name,latitude,longitude,radius_m,row_version FROM tracking_places ORDER BY name,id LIMIT 1001`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Place{}
	for rows.Next() {
		var p Place
		if e = rows.Scan(&p.ID, &p.Name, &p.Latitude, &p.Longitude, &p.Radius, &p.Version); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	if len(out) > 1000 {
		return nil, errors.New("Jumlah lokasi melebihi batas 1.000")
	}
	return out, rows.Err()
}
func SavePlace(ctx context.Context, tx *sql.Tx, p *Place) error {
	if e := ValidatePlace(p); e != nil {
		return e
	}
	if p.ID > 0 {
		r, e := tx.ExecContext(ctx, `UPDATE tracking_places SET name=?,latitude=?,longitude=?,radius_m=?,row_version=row_version+1,updated_at=UTC_TIMESTAMP(3) WHERE id=? AND row_version=?`, p.Name, p.Latitude, p.Longitude, p.Radius, p.ID, p.Version)
		if e != nil {
			return e
		}
		n, _ := r.RowsAffected()
		if n == 0 {
			return errors.New("Lokasi sudah berubah. Muat ulang daftar lokasi")
		}
		p.Version++
		return nil
	}
	var count int
	if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tracking_places`).Scan(&count); e != nil {
		return e
	}
	if count >= 1000 {
		return errors.New("Batas 1.000 lokasi tercapai")
	}
	r, e := tx.ExecContext(ctx, `INSERT INTO tracking_places(name,latitude,longitude,radius_m,created_at,updated_at) VALUES(?,?,?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3))`, p.Name, p.Latitude, p.Longitude, p.Radius)
	if e == nil {
		p.ID, _ = r.LastInsertId()
		p.Version = 1
	}
	return e
}
