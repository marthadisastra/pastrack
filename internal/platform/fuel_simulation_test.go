package platform

import (
	"math"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFuelSimulationScenarios(t *testing.T) {
	normal, anomaly := fuelSimulation("normal"), fuelSimulation("anomaly")
	np := normal["positions"].([]map[string]any)
	nf := normal["readings"].([]map[string]any)
	ap := anomaly["positions"].([]map[string]any)
	af := anomaly["readings"].([]map[string]any)
	if len(np) != 361 || len(nf) != 361 || len(ap) != 361 || len(af) != 361 {
		t.Fatal("expected a minute-by-minute six-hour journey")
	}
	if np[0]["gps_time"] != "2026-10-30T23:00:00Z" || np[360]["gps_time"] != "2026-10-31T05:00:00Z" {
		t.Fatal("WIB date range incorrect")
	}
	for i := 1; i < len(nf); i++ {
		if nf[i]["fuel_liters"].(float64) > nf[i-1]["fuel_liters"].(float64) || nf[i-1]["fuel_liters"].(float64)-nf[i]["fuel_liters"].(float64) > 1 {
			t.Fatal("normal profile must have gradual consumption")
		}
		a, _ := time.Parse(time.RFC3339, np[i-1]["gps_time"].(string))
		b, _ := time.Parse(time.RFC3339, np[i]["gps_time"].(string))
		if b.Sub(a) != time.Minute {
			t.Fatal("non-contiguous samples")
		}
		if np[i]["speed_kmh"].(float64) < 0 || np[i]["speed_kmh"].(float64) > 90 {
			t.Fatal("implausible speed")
		}
	}
	if len(normal["events"].([]map[string]any)) != 0 {
		t.Fatal("normal scenario has anomaly")
	}
	events := anomaly["events"].([]map[string]any)
	if len(events) != 1 || events[0]["delta_l"] != -40 || events[0]["duration_seconds"] != 120 {
		t.Fatal("anomaly missing")
	}
	if math.Abs(af[180]["fuel_liters"].(float64)-af[182]["fuel_liters"].(float64)-40) > 1e-8 {
		t.Fatal("drop does not match readings")
	}
	if ap[180]["latitude"] != ap[182]["latitude"] || ap[182]["speed_kmh"] != float64(0) {
		t.Fatal("anomaly should occur at the stationary point")
	}
	if math.Abs(nf[360]["fuel_liters"].(float64)-af[360]["fuel_liters"].(float64)-40) > 1e-8 {
		t.Fatal("normal and anomaly must differ by 40 liters")
	}
	if normal["simulation"] != true || anomaly["simulation"] != true {
		t.Fatal("simulation provenance missing")
	}
}
func TestRetiredFuelSimulationAPI(t *testing.T) {
	s := New(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/fuel/events/simulation?scenario=normal", nil)
	if s.fuelEventOperations(w, r, &Identity{Role: "viewer"}, "fuel/events/simulation") {
		t.Fatal("retired endpoint is still served")
	}
}
