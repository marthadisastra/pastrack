package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"pastrack/internal/geofence"
	"time"
)

func processGeofences(ctx context.Context, tx *sql.Tx, vehicleID, positionID int64, p *Position) error {
	if p.GPSTime == nil {
		return nil
	}
	var previous geofence.Point
	var previousTime time.Time
	err := tx.QueryRowContext(ctx, `SELECT latitude,longitude,gps_time FROM gps_positions WHERE vehicle_id=? AND fix_status='valid' AND gps_time<? ORDER BY gps_time DESC,id DESC LIMIT 1`, vehicleID, p.GPSTime).Scan(&previous.Lat, &previous.Lon, &previousTime)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if p.GPSTime.Sub(previousTime) > 5*time.Minute {
		return nil
	}
	// Late telemetry is historical only; do not raise a current crossing alarm.
	var latest sql.NullTime
	if err = tx.QueryRowContext(ctx, "SELECT MAX(gps_time) FROM gps_positions WHERE vehicle_id=? AND fix_status='valid'", vehicleID).Scan(&latest); err != nil {
		return err
	}
	if latest.Valid && latest.Time.After(*p.GPSTime) {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT g.id,g.shape,g.center_lat,g.center_lon,g.radius_m,g.polygon_geojson FROM geofences g JOIN geofence_vehicle_assignments a ON a.geofence_id=g.id WHERE a.vehicle_id=? AND g.is_active=1 AND a.starts_at<=? AND (a.ends_at IS NULL OR a.ends_at>?) LIMIT 1000`, vehicleID, p.GPSTime, p.GPSTime)
	if err != nil {
		return err
	}
	type fence struct {
		ID               int64
		Shape            string
		Lat, Lon, Radius sql.NullFloat64
		Polygon          sql.NullString
	}
	fences := []fence{}
	for rows.Next() {
		var f fence
		if err = rows.Scan(&f.ID, &f.Shape, &f.Lat, &f.Lon, &f.Radius, &f.Polygon); err != nil {
			rows.Close()
			return err
		}
		fences = append(fences, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, f := range fences {
		var before, after bool
		current := geofence.Point{Lat: p.Latitude, Lon: p.Longitude}
		switch f.Shape {
		case "circle":
			if !f.Lat.Valid || !f.Lon.Valid || !f.Radius.Valid {
				continue
			}
			center := geofence.Point{Lat: f.Lat.Float64, Lon: f.Lon.Float64}
			before = geofence.Circle(previous, center, f.Radius.Float64)
			after = geofence.Circle(current, center, f.Radius.Float64)
		case "polygon":
			var geo struct {
				Type        string        `json:"type"`
				Coordinates [][][]float64 `json:"coordinates"`
			}
			if !f.Polygon.Valid || json.Unmarshal([]byte(f.Polygon.String), &geo) != nil || geo.Type != "Polygon" {
				continue
			}
			before = geofence.Polygon(previous, geo.Coordinates)
			after = geofence.Polygon(current, geo.Coordinates)
		default:
			continue
		}
		if before == after {
			continue
		}
		transition := "exit"
		if after {
			transition = "enter"
		}
		if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO geofence_events(geofence_id,vehicle_id,position_id,transition,event_at) VALUES(?,?,?,?,?)", f.ID, vehicleID, positionID, transition, p.GPSTime); err != nil {
			return err
		}
	}
	return nil
}
