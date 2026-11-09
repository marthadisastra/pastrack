package models

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

const StopRadiusMeters = 50
const StopMinimumSeconds = 300
const StopMaximumGapSeconds = 300

type Stop struct {
	PositionID    int64     `json:"position_id"`
	EndPositionID int64     `json:"end_position_id"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Duration      int64     `json:"duration_seconds"`
	Samples       int       `json:"samples"`
	BoundaryStart bool      `json:"boundary_start"`
	BoundaryEnd   bool      `json:"boundary_end"`
}

func coordinateDistance(a, b, c, d float64) float64 {
	rad := math.Pi / 180
	h := math.Pow(math.Sin((c-a)*rad/2), 2) + math.Cos(a*rad)*math.Cos(c*rad)*math.Pow(math.Sin((d-b)*rad/2), 2)
	return 6371000 * 2 * math.Atan2(math.Sqrt(math.Min(1, h)), math.Sqrt(math.Max(0, 1-h)))
}
func number(v any) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("missing")
	}
	if b, ok := v.([]byte); ok {
		return strconv.ParseFloat(string(b), 64)
	}
	return strconv.ParseFloat(fmt.Sprint(v), 64)
}
func DetectStops(rows []map[string]any) []Stop {
	out := []Stop{}
	var cluster *Stop
	lastIndex := -1
	flush := func() {
		if cluster != nil {
			cluster.Duration = int64(cluster.End.Sub(cluster.Start).Seconds())
			if cluster.Samples >= 2 && cluster.Duration >= StopMinimumSeconds {
				cluster.BoundaryEnd = lastIndex == len(rows)-1
				out = append(out, *cluster)
			}
		}
		cluster = nil
	}
	for i, r := range rows {
		lat, le := number(r["latitude"])
		lon, loe := number(r["longitude"])
		stamp, te := time.Parse(time.RFC3339Nano, fmt.Sprint(r["gps_time"]))
		id, ie := strconv.ParseInt(fmt.Sprint(r["id"]), 10, 64)
		speed, se := number(r["speed_kmh"])
		if le != nil || loe != nil || te != nil || ie != nil || math.IsNaN(lat) || math.IsNaN(lon) || math.Abs(lat) > 90 || math.Abs(lon) > 180 || (lat == 0 && lon == 0) || (se == nil && speed > 3) {
			flush()
			continue
		}
		if cluster != nil && stamp.Equal(cluster.End) && coordinateDistance(cluster.Latitude, cluster.Longitude, lat, lon) < 0.1 {
			lastIndex = i
			continue
		}
		if cluster != nil && (stamp.Sub(cluster.End) > StopMaximumGapSeconds*time.Second || !stamp.After(cluster.End) || coordinateDistance(cluster.Latitude, cluster.Longitude, lat, lon) > StopRadiusMeters) {
			flush()
		}
		if cluster == nil {
			cluster = &Stop{PositionID: id, EndPositionID: id, Latitude: lat, Longitude: lon, Start: stamp, End: stamp, Samples: 1, BoundaryStart: i == 0}
		} else {
			cluster.End = stamp
			cluster.EndPositionID = id
			cluster.Samples++
		}
		lastIndex = i
	}
	flush()
	return out
}
