package models

import (
	"math"
	"testing"
	"time"
)

func TestVehicleMotionQualityAndWeightedSpeed(t *testing.T) {
	m := newVehicleMotion()
	start := time.Date(2026, 10, 29, 16, 59, 0, 0, time.UTC)
	fix := func(sec int, lon, speed float64) vehicleFix {
		return vehicleFix{at: start.Add(time.Duration(sec) * time.Second), lat: -6, lon: lon, speed: speed, valid: true}
	}
	m.add(fix(0, 106, 36))
	m.add(fix(120, 106.01, 36))
	m.add(fix(120, 106.01, 36))
	if m.moving != 120 || m.duplicates != 1 || m.km < 1 || m.km > 1.2 {
		t.Fatalf("motion: %+v", m)
	}
	if math.Abs(m.daily["2026-10-29"]-m.daily["2026-10-30"]) > 0.001 || m.hourly[23] != 60 || m.hourly[0] != 60 {
		t.Fatal("midnight allocation", m.daily, m.hourly)
	}
	m.add(fix(600, 106.02, 36))
	if m.gaps != 1 || m.moving != 120 {
		t.Fatal("gap counted")
	}
	m.add(fix(660, 120, 36))
	if m.gaps != 2 {
		t.Fatal("jump accepted")
	}
	m.add(fix(720, 120, 0))
	m.add(fix(780, 120, 0))
	if m.stopped != 60 {
		t.Fatal("stationary duration", m.stopped)
	}
	m.add(fix(840, 120, math.NaN()))
	if m.rejected != 1 || m.previous != nil {
		t.Fatal("invalid point bridged")
	}
	m.add(fix(900, 120, 36))
	if m.moving != 120 {
		t.Fatal("invalid bridge")
	}
	if m.weightedSpeed/m.speedSeconds != 36 {
		t.Fatal("speed average")
	}
}
func TestVehicleMotionNoDataAndUnequalIntervals(t *testing.T) {
	m := newVehicleMotion()
	at := time.Now().UTC()
	for i, p := range []struct {
		seconds int
		speed   float64
	}{{0, 10}, {60, 20}, {240, 40}} {
		m.add(vehicleFix{at: at.Add(time.Duration(p.seconds) * time.Second), lat: -6, lon: 106 + float64(i)*.001, speed: p.speed, valid: true})
	}
	if m.moving != 240 || m.weightedSpeed/m.speedSeconds != 26.25 {
		t.Fatal("not duration weighted", m)
	}
	empty := newVehicleMotion()
	empty.add(vehicleFix{})
	if empty.samples != 0 || empty.moving != 0 {
		t.Fatal("missing data became movement")
	}
}

func TestVehicleDistanceIndependentOfSpeedAndSparseDuration(t *testing.T) {
	m := newVehicleMotion()
	at := time.Date(2026, 10, 28, 6, 0, 0, 0, time.UTC)
	m.add(vehicleFix{at: at, lat: -6.13, lon: 106.87, speed: 0.6, valid: true})
	m.add(vehicleFix{at: at.Add(2 * time.Minute), lat: -6.14, lon: 106.88, speed: 0.6, valid: true})
	if m.km < 1 || m.moving != 120 || m.stopped != 0 || m.inferred != 1 || m.weightedSpeed/m.speedSeconds <= 3 {
		t.Fatalf("coordinate movement lost: %+v", m)
	}
	before := m.km
	m.add(vehicleFix{at: at.Add(32 * time.Minute), lat: -6.15, lon: 106.89, speed: 25, valid: true})
	if m.km <= before || m.moving != 120 || m.gaps != 1 {
		t.Fatalf("sparse distance/duration: %+v", m)
	}
	before = m.km
	m.add(vehicleFix{at: at.Add(63 * time.Minute), lat: -6.16, lon: 106.90, speed: 25, valid: true})
	if m.km != before || m.gaps != 2 {
		t.Fatal("long gap bridged")
	}
}
