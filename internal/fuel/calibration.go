// Package fuel separates sensor observations from approved tank-volume calibration.
package fuel

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type Point struct{ Raw, Liters float64 }
type Calibration struct {
	ID              int64
	Approved        bool
	Media, Quantity string
	Capacity        float64
	Points          []Point
}
type Reading struct {
	Liters, Percent *float64
	Validity        string
	CalibrationID   int64
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func (c Calibration) Validate() error {
	if !c.Approved || c.Media != "fuel_tank" {
		return fmt.Errorf("approved fuel-tank calibration required")
	}
	if c.Quantity != "fuel_height_raw" && c.Quantity != "volume_raw" {
		return fmt.Errorf("unverified input quantity")
	}
	if !finite(c.Capacity) || c.Capacity <= 0 || len(c.Points) < 2 {
		return fmt.Errorf("capacity and at least two points required")
	}
	for i, p := range c.Points {
		if !finite(p.Raw) || !finite(p.Liters) || p.Raw < 0 || p.Liters < 0 || p.Liters > c.Capacity {
			return fmt.Errorf("invalid calibration point")
		}
		if i > 0 && (p.Raw <= c.Points[i-1].Raw || p.Liters <= c.Points[i-1].Liters) {
			return fmt.Errorf("calibration must be strictly monotonic")
		}
	}
	return nil
}
func (c Calibration) Convert(raw *float64) Reading {
	r := Reading{Validity: "missing"}
	if raw == nil {
		return r
	}
	r.Validity = "uncalibrated"
	if c.Validate() != nil {
		return r
	}
	r.CalibrationID = c.ID
	if !finite(*raw) || *raw < c.Points[0].Raw || *raw > c.Points[len(c.Points)-1].Raw {
		r.Validity = "out_of_range"
		return r
	}
	i := sort.Search(len(c.Points), func(i int) bool { return c.Points[i].Raw >= *raw })
	if i == 0 {
		i = 1
	}
	a, b := c.Points[i-1], c.Points[i]
	liters := a.Liters + (*raw-a.Raw)/(b.Raw-a.Raw)*(b.Liters-a.Liters)
	percent := 100 * liters / c.Capacity
	r.Liters = &liters
	r.Percent = &percent
	r.Validity = "valid"
	return r
}

type Sample struct {
	At            time.Time
	Liters        *float64
	Valid         bool
	CalibrationID int64
}
type Rule struct {
	MinimumSamples       int
	MinDeltaLiters       float64
	MaxGap, Confirmation time.Duration
}
type Event struct {
	Type        string
	DeltaLiters float64
	At          time.Time
}

// Detect requires a stable baseline and a sustained changed level; gaps, calibration
// changes and missing values suppress events. A single spike cannot create theft/refuel.
func Detect(samples []Sample, rule Rule) *Event {
	n := rule.MinimumSamples
	if n < 2 || len(samples) < 2*n || rule.MinDeltaLiters <= 0 || rule.MaxGap <= 0 || rule.Confirmation <= 0 {
		return nil
	}
	s := samples[len(samples)-2*n:]
	for i, p := range s {
		if !p.Valid || p.Liters == nil || !finite(*p.Liters) || *p.Liters < 0 || p.CalibrationID != s[0].CalibrationID {
			return nil
		}
		if i > 0 && (p.At.Sub(s[i-1].At) <= 0 || p.At.Sub(s[i-1].At) > rule.MaxGap) {
			return nil
		}
	}
	mean := func(p []Sample) (float64, float64) {
		var sum, lo, hi float64
		lo = *p[0].Liters
		hi = lo
		for _, v := range p {
			x := *v.Liters
			sum += x
			lo = math.Min(lo, x)
			hi = math.Max(hi, x)
		}
		return sum / float64(len(p)), hi - lo
	}
	a, spreadA := mean(s[:n])
	b, spreadB := mean(s[n:])
	if spreadA > rule.MinDeltaLiters/4 || spreadB > rule.MinDeltaLiters/4 || s[len(s)-1].At.Sub(s[n].At) < rule.Confirmation {
		return nil
	}
	delta := b - a
	if math.Abs(delta) < rule.MinDeltaLiters {
		return nil
	}
	kind := "refueling"
	if delta < 0 {
		kind = "fuel_drop"
	}
	return &Event{kind, delta, s[n].At}
}
