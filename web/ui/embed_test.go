package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestViewsRenderMVCNavigation(t *testing.T) {
	var b bytes.Buffer
	if e := RenderTo(&b, "index.html"); e != nil {
		t.Fatal(e)
	}
	html := b.String()
	for _, value := range []string{"gentelella-4.2.0.css", "data-view=\"dash-vehicles\"", "data-view=\"dash-gps\"", "data-view=\"dash-activity\"", "data-view=\"dash-fuel\"", "data-master-only hidden", "data-admin-only hidden", "tracking-map", "feedback.js", "feedback.css", "action-confirm", "id=\"fuel-sensor-list\"", "id=\"fuel-calibration-dialog\"", "data-view=\"sensors-master\"", "data-view=\"device-mapping\"", "id=\"sensor-master-dialog\""} {
		if !strings.Contains(html, value) {
			t.Fatalf("view missing %s", value)
		}
	}
	for _, item := range []string{"sensors-master", "device-mapping"} {
		marker := "data-view=\"" + item + "\" data-master-only hidden><svg"
		if !strings.Contains(html, marker) {
			t.Fatalf("sidebar icon missing for %s", item)
		}
	}
	if strings.Contains(html, "data-view=\"fuel\"") {
		t.Fatal("Legacy Data BLE navigation still visible")
	}
	if strings.Contains(html, "{{template") {
		t.Fatal("Unrendered layout")
	}
	b.Reset()
	if e := RenderTo(&b, "login.html"); e != nil {
		t.Fatal(e)
	}
}
