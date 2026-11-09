package models

import (
	"strings"
	"testing"
	"time"
)

func sample(id int64, minutes int, lon float64) map[string]any {
	return map[string]any{"id": id, "latitude": "-6.0", "longitude": lon, "gps_time": time.Date(2026, 10, 29, 0, minutes, 0, 0, time.UTC).Format(time.RFC3339), "speed_kmh": "0.5"}
}
func TestStopsObservedTime(t *testing.T) {
	rows := []map[string]any{sample(1, 0, 106), sample(2, 2, 106), sample(3, 5, 106), sample(4, 6, 106.01)}
	stops := DetectStops(rows)
	if len(stops) != 1 || stops[0].Duration != 300 || stops[0].Samples != 3 || !stops[0].BoundaryStart || stops[0].BoundaryEnd {
		t.Fatalf("Unexpected stops %+v", stops)
	}
}
func TestStopsNeverInferAcrossGapsOrDuplicates(t *testing.T) {
	for _, rows := range [][]map[string]any{{sample(1, 0, 106), sample(2, 10, 106)}, {sample(1, 0, 106), sample(2, 0, 106), sample(3, 0, 106)}, {sample(1, 0, 106), sample(2, 2, 106.002), sample(3, 5, 106)}} {
		if len(DetectStops(rows)) != 0 {
			t.Fatal("Unsupported dwell inferred")
		}
	}
	moving := []map[string]any{sample(1, 0, 106), sample(2, 5, 106)}
	moving[1]["speed_kmh"] = 20
	if len(DetectStops(moving)) != 0 {
		t.Fatal("Moving sample accepted")
	}
}
func TestGeocoderParsing(t *testing.T) {
	a, e := decodeGeo(strings.NewReader(`{"results":[{"address_line1":"Jl. Contoh","formatted":"Jl. Contoh, Bekasi","result_type":"street","distance":12}]}`))
	if e != nil || a.Name != "Jl. Contoh" || a.Status != "ok" || *a.Distance != 12 {
		t.Fatalf("Unexpected geocoder %+v %v", a, e)
	}
	a, e = decodeGeo(strings.NewReader(`{"results":[]}`))
	if e != nil || a.Status != "not_found" {
		t.Fatal(a, e)
	}
}
