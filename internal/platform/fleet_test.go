package platform

import (
	"testing"
	"time"
)

func TestFleetExclusiveStatesAndFreshness(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-time.Minute).Format(time.RFC3339Nano)
	old := now.Add(-10 * time.Minute).Format(time.RFC3339Nano)
	cases := []struct {
		name     string
		device   any
		received any
		gps      any
		speed    any
		fix      string
		want     string
	}{
		{"unassigned", nil, nil, nil, nil, "", "UNASSIGNED"}, {"offline", int64(1), old, fresh, "20", "valid", "OFFLINE"}, {"no fix", int64(1), fresh, nil, nil, "", "GPS_NO_FIX"}, {"buffered", int64(1), fresh, old, "20", "valid", "STALE"}, {"unknown speed", int64(1), fresh, fresh, nil, "valid", "UNKNOWN_SPEED"}, {"moving", int64(1), fresh, fresh, "3.01", "valid", "MOVING"}, {"threshold", int64(1), fresh, fresh, "3.00", "valid", "STOPPED"}, {"future clock", int64(1), fresh, now.Add(time.Hour).Format(time.RFC3339Nano), "10", "valid", "STALE"}}
	rows := []map[string]any{}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := map[string]any{"device_id": tt.device, "device_last_received_at": tt.received, "gps_time": tt.gps, "latitude": "-6", "speed_kmh": tt.speed, "fix_status": tt.fix, "active_alerts": int64(1)}
			classifyFleet(r, now)
			if r["movement_status"] != tt.want {
				t.Fatal(r)
			}
			rows = append(rows, r)
		})
	}
	summary := fleetSummary(rows)
	sum := 0
	for _, key := range []string{"moving", "stopped", "vehicles_offline", "unassigned", "gps_stale", "gps_no_fix", "unknown_speed"} {
		sum += summary[key].(int)
	}
	if sum != len(cases) || summary["active_alerts"] != len(cases) {
		t.Fatal(summary)
	}
}

func TestFleetBufferedFuelIsNotFresh(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-time.Minute).Format(time.RFC3339Nano)
	old := now.Add(-24 * time.Hour).Format(time.RFC3339Nano)
	r := map[string]any{"device_id": int64(1), "device_last_received_at": fresh, "fuel_received_at": fresh, "fuel_measured_at": old, "sensor_last_received_at": old, "sensor_status": "connected"}
	classifyFleet(r, now)
	if r["fuel_fresh"] != false || r["sensor_status"] != "STALE" {
		t.Fatal(r)
	}
	r["fuel_measured_at"] = nil
	classifyFleet(r, now)
	if r["fuel_fresh"] != false {
		t.Fatal("unknown measurement time treated as current")
	}
	r["fuel_measured_at"] = fresh
	classifyFleet(r, now)
	if r["fuel_fresh"] != true {
		t.Fatal("fresh timed measurement rejected")
	}
}

func TestFleetFreshnessAcceptsSQLTimeValues(t *testing.T) {
	now := time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	r := map[string]any{
		"device_id":               int64(1),
		"device_last_received_at": fresh,
		"fuel_received_at":        fresh,
		"fuel_measured_at":        fresh,
		"fuel_validity":           "valid",
		"gps_time":                fresh,
		"latitude":                -6.2,
		"fix_status":              "valid",
		"speed_kmh":               0.0,
	}
	classifyFleet(r, now)
	if r["fuel_fresh"] != true || r["gps_fresh"] != true {
		t.Fatalf("SQL time.Time values were not classified fresh: %#v", r)
	}
}

func TestIgnitionPersistsWhenGatewayOffline(t *testing.T) {
	payload := "+RESP:GTIGL,80201E0306,000000000000000,GV30CEU,,00,1,5,0.0,0,65.4,106.872339,-6.133696,20261103094920,0510,0001,7D06,033CA45D,01,8,0.0,20261103094921,4E95$"
	row := map[string]any{"device_id": int64(1), "ignition_payload": payload, "speed_kmh": "0"}
	classifyFleet(row, time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	if row["ignition_status"] != "ON" || row["movement_status"] != "OFFLINE" {
		t.Fatal(row)
	}
	if row["ignition_reported_at"] != "2026-11-03T09:49:21Z" {
		t.Fatal("GPS time used as ignition event time", row)
	}
}

func TestGTIGFLastStateUsesSendTimeDespiteOldNoFixPosition(t *testing.T) {
	payload := "+RESP:GTIGF,80201E0306,000000000000000,GV30CEU,8,0,13.0,128,23.5,106.876348,-6.128524,20261103103229,0510,0001,7D06,0337D748,01,0,,0.0,20261103105224,518D$"
	row := map[string]any{"device_id": int64(1), "ignition_payload": payload, "fix_status": "no_fix", "speed_kmh": "13"}
	classifyFleet(row, time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC))
	if row["ignition_status"] != "OFF" || row["ignition_source"] != "GTIGF" || row["ignition_reported_at"] != "2026-11-03T10:52:24Z" {
		t.Fatal(row)
	}
	if row["ignition_gps_time"] != "2026-11-03T10:32:29Z" {
		t.Fatal(row)
	}
}

func TestRemoteOutputIndependentFromIgnition(t *testing.T) {
	row := map[string]any{"remote_output_payload": "+RESP:GTDOS,80201E0306,000000000000000,GV30CEU,1,1,0,13.0,128,23.5,106.876348,-6.128524,20261103103229,0510,0001,7D06,0337D748,01,0,20261103105142,1234$", "ignition_payload": "+RESP:GTIGF,80201E0306,000000000000000,GV30CEU,8,0,13.0,128,23.5,106.876348,-6.128524,20261103103229,0510,0001,7D06,0337D748,01,0,,0.0,20261103105224,518D$"}
	classifyFleet(row, time.Now())
	if row["remote_output_active"] != true || row["ignition_status"] != "OFF" {
		t.Fatal(row)
	}
}
