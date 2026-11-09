package models

import (
	"context"
	"math"
	"sort"
)

type FuelPoint struct {
	ID      int64   `json:"id"`
	Percent float64 `json:"level_percent"`
	Raw     int     `json:"raw_value"`
	Note    string  `json:"note"`
}

func EstimateFuelPercent(raw int, points []FuelPoint) (float64, bool) {
	if len(points) < 2 {
		return 0, false
	}
	copyPoints := append([]FuelPoint(nil), points...)
	sort.Slice(copyPoints, func(i, j int) bool { return copyPoints[i].Raw < copyPoints[j].Raw })
	if copyPoints[0].Percent != 0 || copyPoints[len(copyPoints)-1].Percent != 100 {
		return 0, false
	}
	for i, p := range copyPoints {
		if p.Raw < 0 || p.Raw > 65535 || math.IsNaN(p.Percent) || p.Percent < 0 || p.Percent > 100 || (i > 0 && (p.Raw <= copyPoints[i-1].Raw || p.Percent <= copyPoints[i-1].Percent)) {
			return 0, false
		}
	}
	if raw <= copyPoints[0].Raw {
		return 0, true
	}
	if raw >= copyPoints[len(copyPoints)-1].Raw {
		return 100, true
	}
	for i := 1; i < len(copyPoints); i++ {
		if raw <= copyPoints[i].Raw {
			a, b := copyPoints[i-1], copyPoints[i]
			return a.Percent + float64(raw-a.Raw)*((b.Percent-a.Percent)/float64(b.Raw-a.Raw)), true
		}
	}
	return 0, false
}

func FuelPoints(ctx context.Context, q Queryer, sensorPK int64) ([]FuelPoint, error) {
	rows, err := q.QueryContext(ctx, `SELECT id,level_percent,raw_value,COALESCE(note,'') FROM fuel_calibration_points WHERE sensor_pk=? ORDER BY raw_value`, sensorPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	points := []FuelPoint{}
	for rows.Next() {
		var p FuelPoint
		if err = rows.Scan(&p.ID, &p.Percent, &p.Raw, &p.Note); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func FuelSensorStatus(ctx context.Context, deviceID int64) ([]map[string]any, error) {
	where := "s.deleted_at IS NULL AND (d.deleted_at IS NULL OR d.id IS NULL)"
	args := []any{}
	if deviceID > 0 {
		where += " AND d.id=?"
		args = append(args, deviceID)
	}
	rows, err := Query(ctx, `SELECT s.id sensor_pk,s.device_id,d.imei,d.vehicle_id,v.plate_number,a.tank_capacity_l,s.reported_device_id,rd.imei reported_imei,s.sensor_id,s.model,s.last_seen_at,s.observed_raw_value,s.observed_at,s.calibration_media,s.is_active,(SELECT COUNT(*) FROM fuel_calibration_points p WHERE p.sensor_pk=s.id) point_count FROM device_fuel_sensors s LEFT JOIN gps_devices d ON d.id=s.device_id LEFT JOIN gps_devices rd ON rd.id=s.reported_device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN vehicle_assets a ON a.vehicle_id=v.id WHERE `+where+` ORDER BY v.plate_number,d.imei,s.sensor_id`, args...)
	if err != nil {
		return nil, err
	}
	pointRows, err := DB.QueryContext(ctx, `SELECT p.sensor_pk,p.id,p.level_percent,p.raw_value,COALESCE(p.note,'') FROM fuel_calibration_points p JOIN device_fuel_sensors s ON s.id=p.sensor_pk LEFT JOIN gps_devices d ON d.id=s.device_id WHERE `+where+` ORDER BY p.sensor_pk,p.raw_value`, args...)
	if err != nil {
		return nil, err
	}
	defer pointRows.Close()
	pointsBySensor := map[int64][]FuelPoint{}
	for pointRows.Next() {
		var sensorPK int64
		var point FuelPoint
		if err = pointRows.Scan(&sensorPK, &point.ID, &point.Percent, &point.Raw, &point.Note); err != nil {
			return nil, err
		}
		pointsBySensor[sensorPK] = append(pointsBySensor[sensorPK], point)
	}
	if err = pointRows.Err(); err != nil {
		return nil, err
	}
	for _, r := range rows {
		id, e := number(r["sensor_pk"])
		if e != nil {
			continue
		}
		points := pointsBySensor[int64(id)]
		if points == nil {
			points = []FuelPoint{}
		}
		r["points"] = points
		if raw, e := number(r["observed_raw_value"]); e == nil {
			if percent, valid := EstimateFuelPercent(int(raw), points); valid {
				r["estimated_percent"] = math.Round(percent*10) / 10
				if capacity, e := number(r["tank_capacity_l"]); e == nil && capacity > 0 {
					r["estimated_liters"] = math.Round((percent*capacity/100)*10) / 10
				}
			}
		}
	}
	return rows, nil
}
