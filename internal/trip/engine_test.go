package trip

import (
	"testing"
	"time"
)

func TestTripsRejectJumpsAndGaps(t *testing.T) {
	at := time.Now()
	c := Config{MinimumSpeed: 3, MaximumSpeed: 160, MinimumDistanceMeters: 30, MinimumDuration: time.Minute, StopDuration: time.Minute, MaximumGap: 3 * time.Minute}
	p := []Position{{1, at, -6, 106, 20, true}, {2, at.Add(time.Minute), -6.001, 106, 20, true}, {3, at.Add(2 * time.Minute), -60, 106, 20, true}, {4, at.Add(2 * time.Minute), -6.002, 106, 0, true}, {5, at.Add(3 * time.Minute), -6.002, 106, 0, true}}
	trips := Build(p, c)
	if len(trips) != 1 || trips[0].DistanceMeters > 300 || trips[0].EndReason != "stopped" {
		t.Fatal(trips)
	}
	p[2].At = at.Add(10 * time.Minute)
	trips = Build(p, c)
	for _, trip := range trips {
		if trip.DistanceMeters > 500 {
			t.Fatal("gap invented distance")
		}
	}
}
