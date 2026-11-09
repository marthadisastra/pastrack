package geofence

import "testing"

func TestCircleAndPolygon(t *testing.T) {
	if !Circle(Point{-6, 106}, Point{-6, 106}, 100) || Circle(Point{-7, 106}, Point{-6, 106}, 100) {
		t.Fatal("circle bounds")
	}
	rings := [][][]float64{{{105, -7}, {107, -7}, {107, -5}, {105, -5}, {105, -7}}}
	if !Polygon(Point{-6, 106}, rings) || Polygon(Point{-8, 106}, rings) {
		t.Fatal("polygon bounds")
	}
}
