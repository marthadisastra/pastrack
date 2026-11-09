package geofence

import (
	"math"
	"pastrack/internal/trip"
)

type Point struct{ Lat, Lon float64 }

func Circle(point, center Point, radius float64) bool {
	return radius > 0 && trip.Distance(trip.Position{Lat: point.Lat, Lon: point.Lon}, trip.Position{Lat: center.Lat, Lon: center.Lon}) <= radius
}

// Polygon handles closed GeoJSON rings in lon/lat order. Boundary is inside.
func Polygon(point Point, rings [][][]float64) bool {
	if len(rings) == 0 || !ring(point, rings[0]) {
		return false
	}
	for _, hole := range rings[1:] {
		if ring(point, hole) {
			return false
		}
	}
	return true
}
func ring(p Point, points [][]float64) bool {
	if len(points) < 4 {
		return false
	}
	inside := false
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if len(a) != 2 || len(b) != 2 {
			return false
		}
		cross := (p.Lon-a[0])*(b[1]-a[1]) - (p.Lat-a[1])*(b[0]-a[0])
		if math.Abs(cross) < 1e-10 && p.Lon >= math.Min(a[0], b[0]) && p.Lon <= math.Max(a[0], b[0]) && p.Lat >= math.Min(a[1], b[1]) && p.Lat <= math.Max(a[1], b[1]) {
			return true
		}
		if (a[1] > p.Lat) != (b[1] > p.Lat) && p.Lon < (b[0]-a[0])*(p.Lat-a[1])/(b[1]-a[1])+a[0] {
			inside = !inside
		}
	}
	return inside
}
