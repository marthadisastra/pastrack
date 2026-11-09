package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
)

var ErrTrackingFleetLimit = errors.New("Rute melebihi 100.000 titik total atau 20.000 titik per perangkat. Pilih periode lebih pendek atau satu kendaraan.")

type FleetTrackingStop struct {
	Stop
	DeviceID  any `json:"device_id"`
	VehicleID any `json:"vehicle_id"`
	Plate     any `json:"plate_number"`
}
type TrackingRoute struct {
	LastPosition map[string]any      `json:"last_position,omitempty"`
	Vehicle      map[string]any      `json:"vehicle"`
	Device       map[string]any      `json:"device,omitempty"`
	Data         []map[string]any    `json:"data"`
	Stops        []FleetTrackingStop `json:"stops"`
}
type FleetTracking struct {
	Vehicles  []map[string]any `json:"vehicles"`
	Tracks    []TrackingRoute  `json:"tracks"`
	Start     time.Time        `json:"start"`
	End       time.Time        `json:"end"`
	Generated time.Time        `json:"generated_at"`
	Basis     string           `json:"basis"`
	Total     int              `json:"total"`
	GeoReady  bool             `json:"geocoding_configured"`
}

// Valid unique fixes select a primary reporting device without double-counting a vehicle.
func trackingFixCount(rows []map[string]any) int {
	seen := map[string]bool{}
	for _, r := range rows {
		lat, a := number(r["latitude"])
		lon, b := number(r["longitude"])
		if a != nil || b != nil || math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) || math.Abs(lat) > 90 || math.Abs(lon) > 180 || (lat == 0 && lon == 0) || r["gps_time"] == nil {
			continue
		}
		seen[fmt.Sprint(r["gps_time"])] = true
	}
	return len(seen)
}
func fleetTrackingRoutes(vehicles, devices, positions []map[string]any) ([]TrackingRoute, error) {
	grouped := map[string][]map[string]any{}
	for _, p := range positions {
		key := fmt.Sprint(p["device_id"])
		grouped[key] = append(grouped[key], p)
		if len(grouped[key]) > 20000 {
			return nil, ErrTrackingFleetLimit
		}
	}
	out := []TrackingRoute{}
	for _, v := range vehicles {
		route := TrackingRoute{Vehicle: v, Data: []map[string]any{}, Stops: []FleetTrackingStop{}}
		best := -1
		for _, d := range devices {
			if fmt.Sprint(d["vehicle_id"]) != fmt.Sprint(v["id"]) {
				continue
			}
			points := grouped[fmt.Sprint(d["id"])]
			count := trackingFixCount(points)
			if count > best {
				route.Device = d
				route.Data = points
				if route.Data == nil {
					route.Data = []map[string]any{}
				}
				best = count
			}
		}
		if route.Device != nil && route.Device["latitude"] != nil && route.Device["longitude"] != nil {
			route.LastPosition = map[string]any{"latitude": route.Device["latitude"], "longitude": route.Device["longitude"], "gps_time": route.Device["gps_time"], "plate_number": v["plate_number"], "vehicle_name": v["vehicle_name"], "vehicle_id": v["id"]}
		}
		for _, p := range route.Data {
			p["vehicle_id"] = v["id"]
			p["plate_number"] = v["plate_number"]
			p["vehicle_name"] = v["vehicle_name"]
		}
		for _, stop := range DetectStops(route.Data) {
			route.Stops = append(route.Stops, FleetTrackingStop{Stop: stop, DeviceID: route.Device["id"], VehicleID: v["id"], Plate: v["plate_number"]})
		}
		out = append(out, route)
	}
	return out, nil
}
func LoadFleetTracking(ctx context.Context, vehicle int64, start, end time.Time, options bool) (FleetTracking, error) {
	out := FleetTracking{Start: start, End: end, Generated: time.Now().UTC(), Basis: "gps_time", GeoReady: GeoReady(), Tracks: []TrackingRoute{}}
	tx, e := DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	where := "v.deleted_at IS NULL"
	args := []any{}
	if vehicle > 0 {
		where += " AND v.id=?"
		args = append(args, vehicle)
	}
	out.Vehicles, e = QueryOn(ctx, tx, `SELECT v.id,v.plate_number,v.vehicle_name,v.fleet_code,(SELECT COUNT(*) FROM gps_devices d WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL) device_count FROM vehicles v WHERE `+where+` ORDER BY v.plate_number,v.id`, args...)
	if e != nil {
		return out, e
	}
	if vehicle > 0 && len(out.Vehicles) == 0 {
		return out, sql.ErrNoRows
	}
	if options {
		return out, tx.Commit()
	}
	devices, e := QueryOn(ctx, tx, `SELECT d.id,d.vehicle_id,d.imei,d.model,v.plate_number,p.latitude,p.longitude,p.gps_time FROM gps_devices d JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN gps_positions p ON p.id=(SELECT pp.id FROM gps_positions pp WHERE pp.device_id=d.id ORDER BY pp.gps_time DESC,pp.id DESC LIMIT 1) WHERE d.deleted_at IS NULL AND `+where+` ORDER BY d.id`, args...)
	if e != nil {
		return out, e
	}
	pointArgs := append([]any{start, end}, args...)
	positions, e := QueryOn(ctx, tx, `SELECT p.id,p.device_id,p.latitude,p.longitude,p.speed_kmh,p.heading_deg,p.altitude_m,p.gps_time,p.device_send_time,p.created_at FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id JOIN vehicles v ON v.id=d.vehicle_id WHERE p.gps_time>=? AND p.gps_time<=? AND d.deleted_at IS NULL AND `+where+` ORDER BY p.device_id,p.gps_time,p.id LIMIT 100001`, pointArgs...)
	if e != nil {
		return out, e
	}
	if len(positions) > 100000 {
		return out, ErrTrackingFleetLimit
	}
	out.Tracks, e = fleetTrackingRoutes(out.Vehicles, devices, positions)
	if e != nil {
		return out, e
	}
	for _, route := range out.Tracks {
		out.Total += len(route.Data)
	}
	return out, tx.Commit()
}
