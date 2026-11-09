package models

import (
	"fmt"
	"testing"
)

func TestFleetRoutesAutoSelectDeviceAndKeepEmptyVehicles(t *testing.T) {
	vehicles := []map[string]any{{"id": 1, "plate_number": "A"}, {"id": 2, "plate_number": "B"}, {"id": 3, "plate_number": "EMPTY"}, {"id": 4, "plate_number": "NO DEVICE"}}
	devices := []map[string]any{{"id": 10, "vehicle_id": 1}, {"id": 11, "vehicle_id": 1}, {"id": 20, "vehicle_id": 2}, {"id": 30, "vehicle_id": 3}}
	fix := func(id, device, minute int) map[string]any {
		return map[string]any{"id": id, "device_id": device, "latitude": -6, "longitude": 106, "speed_kmh": 0, "gps_time": fmt.Sprintf("2026-10-30T00:%02d:00Z", minute)}
	}
	positions := []map[string]any{fix(1, 10, 0), fix(2, 10, 0), fix(3, 10, 0), fix(4, 11, 0), fix(5, 11, 5), fix(6, 20, 0), fix(7, 20, 5)}
	routes, e := fleetTrackingRoutes(vehicles, devices, positions)
	if e != nil {
		t.Fatal(e)
	}
	if len(routes) != 4 || routes[0].Device["id"] != 11 || len(routes[0].Data) != 2 {
		t.Fatal("wrong device/vehicles", routes)
	}
	if routes[2].Device == nil || len(routes[2].Data) != 0 || routes[3].Device != nil || routes[3].Data == nil {
		t.Fatal("missing vehicles discarded")
	}
	for _, i := range []int{0, 1} {
		if len(routes[i].Stops) != 1 || routes[i].Stops[0].DeviceID != routes[i].Device["id"] || routes[i].Stops[0].Samples != 2 {
			t.Fatal("stop mixed across devices", routes[i])
		}
	}
	if routes[0].Data[0]["plate_number"] != "A" || routes[1].Data[0]["plate_number"] != "B" {
		t.Fatal("wrong vehicle labels")
	}
}
func TestTrackingFleetPerDeviceLimitAndValidFixes(t *testing.T) {
	rows := make([]map[string]any, 20001)
	for i := range rows {
		rows[i] = map[string]any{"device_id": 1}
	}
	if _, e := fleetTrackingRoutes(nil, nil, rows); e != ErrTrackingFleetLimit {
		t.Fatal("silent truncation", e)
	}
	if trackingFixCount([]map[string]any{{"latitude": nil, "longitude": 106, "gps_time": "x"}, {"latitude": 0, "longitude": 0, "gps_time": "x"}}) != 0 {
		t.Fatal("invalid fixes chosen")
	}
}
