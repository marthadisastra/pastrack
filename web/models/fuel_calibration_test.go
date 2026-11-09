package models

import "testing"

func TestEstimateFuelPercent(t *testing.T) {
	p := []FuelPoint{{Percent: 100, Raw: 4992}, {Percent: 0, Raw: 209}, {Percent: 50, Raw: 2500}}
	for _, tc := range []struct {
		raw  int
		want float64
	}{{100, 0}, {209, 0}, {2500, 50}, {4992, 100}, {6000, 100}} {
		got, ok := EstimateFuelPercent(tc.raw, p)
		if !ok || got != tc.want {
			t.Fatalf("raw %d: %v %v", tc.raw, got, ok)
		}
	}
	if _, ok := EstimateFuelPercent(300, []FuelPoint{{Percent: 0, Raw: 209}, {Percent: 100, Raw: 209}}); ok {
		t.Fatal("duplicate raw accepted")
	}
	if _, ok := EstimateFuelPercent(300, []FuelPoint{{Percent: 0, Raw: 209}, {Percent: 90, Raw: 4992}}); ok {
		t.Fatal("missing full endpoint accepted")
	}
}

func TestNumberAcceptsDatabaseDecimal(t *testing.T) {
	got, ok := number([]byte("120.50"))
	if ok != nil || got != 120.5 {
		t.Fatalf("number database decimal = %v, %v", got, ok)
	}
}
