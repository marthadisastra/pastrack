package models

import (
	"math"
	"testing"
)

func TestPlaceValidation(t *testing.T) {
	p := Place{Name: " Garasi ", Latitude: -6.1, Longitude: 106.8, Radius: 100}
	if e := ValidatePlace(&p); e != nil || p.Name != "Garasi" {
		t.Fatal(e)
	}
	for _, bad := range []Place{{Name: "", Latitude: -6, Longitude: 106, Radius: 100}, {Name: "QA", Latitude: math.NaN(), Longitude: 106, Radius: 100}, {Name: "QA", Latitude: 0, Longitude: 0, Radius: 100}, {Name: "QA", Latitude: -6, Longitude: 106, Radius: 10}} {
		if ValidatePlace(&bad) == nil {
			t.Fatal("Invalid place accepted")
		}
	}
}
