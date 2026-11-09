package controllers

import (
	"testing"
	"time"
)

func TestTrackingRange(t *testing.T) {
	a, b, e := trackingRange("2026-10-28T07:00:00+07:00", "2026-10-29T07:00:00+07:00")
	if e != nil || a.Hour() != 0 || b.Sub(a) != 24*time.Hour {
		t.Fatalf("WIB conversion %v %v %v", a, b, e)
	}
	for _, pair := range [][2]string{{"bad", "2026-10-29T00:00:00Z"}, {"2026-10-29T00:00:00Z", "bad"}, {"2026-10-29T00:00:00Z", "2026-10-29T00:00:00Z"}, {"2026-10-29T00:00:00Z", "2026-10-28T00:00:00Z"}, {"2026-09-01T00:00:00Z", "2026-10-29T00:00:00Z"}} {
		if _, _, e := trackingRange(pair[0], pair[1]); e == nil {
			t.Fatalf("accepted invalid range %v", pair)
		}
	}
}
