// Package trip derives movement trips without making claims about engine ignition.
package trip

import (
	"math"
	"time"
)

type Position struct {
	ID              int64
	At              time.Time
	Lat, Lon, Speed float64
	Valid           bool
}
type Config struct {
	MinimumSpeed, MaximumSpeed, MinimumDistanceMeters float64
	MinimumDuration, StopDuration, MaximumGap         time.Duration
}
type Trip struct {
	StartID, EndID                         int64
	StartedAt, EndedAt                     time.Time
	DistanceMeters, MaxSpeed, AverageSpeed float64
	DrivingSeconds, StoppedSeconds         float64
	EndReason                              string
}

func valid(p Position) bool {
	return p.Valid && !p.At.IsZero() && !math.IsNaN(p.Lat) && !math.IsNaN(p.Lon) && !math.IsNaN(p.Speed) && !math.IsInf(p.Lat, 0) && !math.IsInf(p.Lon, 0) && !math.IsInf(p.Speed, 0) && math.Abs(p.Lat) <= 90 && math.Abs(p.Lon) <= 180 && p.Speed >= 0
}
func Distance(a, b Position) float64 {
	const rad = math.Pi / 180
	dlat := (b.Lat - a.Lat) * rad
	dlon := (b.Lon - a.Lon) * rad
	x := math.Pow(math.Sin(dlat/2), 2) + math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Pow(math.Sin(dlon/2), 2)
	return 6371000 * 2 * math.Asin(math.Sqrt(math.Min(1, x)))
}

// Build accepts event-time ordered data. Duplicate/late points are ignored. A gap
// closes a trip with an explicit reason; no distance is invented across the gap.
func Build(points []Position, c Config) []Trip {
	out := []Trip{}
	if c.MinimumSpeed <= 0 || c.MaximumSpeed <= c.MinimumSpeed || c.MaximumGap <= 0 || c.StopDuration <= 0 {
		return out
	}
	var active *Trip
	var prev *Position
	var stoppedAt time.Time
	finish := func(reason string) {
		if active != nil {
			active.EndReason = reason
			duration := active.EndedAt.Sub(active.StartedAt)
			if duration >= c.MinimumDuration && active.DistanceMeters >= c.MinimumDistanceMeters {
				if active.DrivingSeconds > 0 {
					active.AverageSpeed = active.DistanceMeters / active.DrivingSeconds * 3.6
				}
				out = append(out, *active)
			}
		}
		active = nil
		stoppedAt = time.Time{}
	}
	for _, p := range points {
		if !valid(p) || p.Speed > c.MaximumSpeed {
			continue
		}
		if prev != nil && !p.At.After(prev.At) {
			continue
		}
		if prev != nil {
			gap := p.At.Sub(prev.At)
			if gap > c.MaximumGap {
				finish("data_gap")
				prev = nil
			} else if Distance(*prev, p)/gap.Seconds()*3.6 > c.MaximumSpeed {
				continue
			}
		}
		if active == nil && p.Speed >= c.MinimumSpeed {
			active = &Trip{StartID: p.ID, EndID: p.ID, StartedAt: p.At, EndedAt: p.At, MaxSpeed: p.Speed}
			prev = &p
			continue
		}
		if active != nil && prev != nil {
			seconds := p.At.Sub(prev.At).Seconds()
			active.EndID = p.ID
			active.EndedAt = p.At
			if p.Speed >= c.MinimumSpeed || prev.Speed >= c.MinimumSpeed {
				active.DistanceMeters += Distance(*prev, p)
				active.DrivingSeconds += seconds
			} else {
				active.StoppedSeconds += seconds
			}
			if p.Speed >= c.MinimumSpeed {
				active.MaxSpeed = math.Max(active.MaxSpeed, p.Speed)
				stoppedAt = time.Time{}
			} else {
				if stoppedAt.IsZero() {
					stoppedAt = p.At
				}
				if p.At.Sub(stoppedAt) >= c.StopDuration {
					finish("stopped")
				}
			}
		}
		prev = &p
	}
	// An unfinished trip is not emitted as a completed journey.
	return out
}
