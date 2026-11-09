package fuel

import (
	"testing"
	"time"
)

func TestCalibrationSafeguards(t *testing.T) {
	c := Calibration{ID: 1, Approved: true, Media: "fuel_tank", Quantity: "fuel_height_raw", Capacity: 100, Points: []Point{{0, 0}, {1000, 40}, {2000, 100}}}
	raw := 1500.0
	r := c.Convert(&raw)
	if r.Validity != "valid" || *r.Liters != 70 || *r.Percent != 70 {
		t.Fatalf("wrong interpolation: %+v", r)
	}
	raw = 2100
	if c.Convert(&raw).Liters != nil {
		t.Fatal("out of range was clamped")
	}
	c.Media = "water_test"
	raw = 1000
	if c.Convert(&raw).Liters != nil {
		t.Fatal("water became liters")
	}
	if c.Convert(nil).Liters != nil {
		t.Fatal("missing became zero")
	}
}
func TestSustainedFuelEvents(t *testing.T) {
	start := time.Now()
	samples := []Sample{}
	for i, v := range []float64{100, 100, 100, 80, 80, 80} {
		value := v
		samples = append(samples, Sample{At: start.Add(time.Duration(i) * time.Minute), Liters: &value, Valid: true, CalibrationID: 1})
	}
	rule := Rule{3, 10, 2 * time.Minute, 2 * time.Minute}
	event := Detect(samples, rule)
	if event == nil || event.Type != "fuel_drop" || event.DeltaLiters != -20 {
		t.Fatal(event)
	}
	samples[4].Liters = nil
	if Detect(samples, rule) != nil {
		t.Fatal("disconnect generated event")
	}
}
