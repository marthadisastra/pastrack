package alertrules

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pastrack/internal/alertengine"
	"pastrack/internal/telemetry"
)

var testNow = time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return testNow }
func ptr[T any](v T) *T     { return &v }

func fuelEvent(id, vehicle, sensor int64, percent float64, quality telemetry.TimestampQuality, at time.Time) alertengine.Event {
	source := "SENSOR_TIME"
	if quality == telemetry.CorrelatedTime {
		source = "SAME_RAW_GPS_TIME"
	}
	if quality == telemetry.ReceivedTimeOnly {
		source = "RECEIVED_TIME_ONLY"
	}
	v := telemetry.NormalizedFuel{Kind: telemetry.FuelReading, FuelReadingID: id, TelemetryID: id, RawMessageID: id + 1000, SensorID: sensor, MAC: "AABBCCDDEEFF", ReceiverDeviceID: 99, VehicleID: ptr(vehicle), FuelPercent: ptr(percent), CalibrationID: ptr(int64(55)), Validity: "valid", EventTime: &at, ReceivedAt: testNow, TimestampQuality: quality, TimestampSource: source, ResolutionStatus: telemetry.Resolved}
	if quality == telemetry.ReceivedTimeOnly || quality == telemetry.UnknownTime {
		v.EventTime = nil
	}
	payload, _ := json.Marshal(v)
	return alertengine.Event{ID: id, Type: string(telemetry.FuelReading), VehicleID: vehicle, SourceType: "BLE", SourceIdentifier: v.MAC, DeviceID: ptr(int64(99)), SensorID: ptr(sensor), FuelReadingID: ptr(id), SensorTelemetryID: ptr(id), RawMessageID: ptr(id + 1000), EventTime: v.EventTime, ReceivedAt: v.ReceivedAt, TimestampQuality: quality, ResolutionStatus: telemetry.Resolved, Payload: payload}
}

func gpsEvent(id, vehicle, device int64, speed float64, at time.Time, fix bool) alertengine.Event {
	v := telemetry.NormalizedGPS{Kind: telemetry.GPSPosition, PositionID: id, DeviceID: device, IMEI: "123", VehicleID: ptr(vehicle), Latitude: -6.2, Longitude: 106.8, SpeedKMH: ptr(speed), EventTime: &at, ReceivedAt: testNow, FixValid: fix, TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: telemetry.Resolved}
	payload, _ := json.Marshal(v)
	return alertengine.Event{ID: id, Type: string(telemetry.GPSPosition), VehicleID: vehicle, SourceType: "GPS", SourceIdentifier: v.IMEI, DeviceID: ptr(device), EventTime: &at, ReceivedAt: v.ReceivedAt, TimestampQuality: v.TimestampQuality, ResolutionStatus: v.ResolutionStatus, Payload: payload}
}

func evaluate(t *testing.T, r alertengine.RuleEvaluator, event alertengine.Event, state json.RawMessage) (alertengine.Evaluation, json.RawMessage) {
	t.Helper()
	if err := r.Validate(event); err != nil {
		t.Fatal(err)
	}
	out, err := r.Evaluate(event, state)
	if err != nil {
		t.Fatal(err)
	}
	return out, out.State
}

func TestFuelLowThresholdRecoveryAndTimestampQualities(t *testing.T) {
	r := NewFuelLow(1, 1, FuelLowConfig{Threshold: 10, Recovery: 15, MinimumSamples: 2, RecoverySamples: 2, FreshnessSeconds: 300}, fixedClock)
	at := testNow.Add(-time.Minute)
	var state json.RawMessage
	out, state := evaluate(t, r, fuelEvent(1, 1, 7, 9, telemetry.ReceivedTimeOnly, at), state)
	if out.Action != alertengine.Observe {
		t.Fatalf("first=%s", out.Action)
	}
	out, state = evaluate(t, r, fuelEvent(2, 1, 7, 9, telemetry.UnknownTime, at), state)
	if out.Action != alertengine.Trigger {
		t.Fatalf("second=%s", out.Action)
	}
	out, state = evaluate(t, r, fuelEvent(3, 1, 7, 15, telemetry.CorrelatedTime, at), state)
	if out.Action != alertengine.Observe {
		t.Fatalf("first recovery=%s", out.Action)
	}
	out, _ = evaluate(t, r, fuelEvent(4, 1, 7, 16, telemetry.CorrelatedTime, at), state)
	if out.Action != alertengine.Recover {
		t.Fatalf("recovery=%s", out.Action)
	}
	for _, p := range []float64{10, 11} {
		out, _ = evaluate(t, r, fuelEvent(10+int64(p), 1, 8, p, telemetry.ReceivedTimeOnly, at), nil)
		if out.Action == alertengine.Trigger {
			t.Fatalf("percent %.0f triggered", p)
		}
	}
}

func TestFuelLowRejectsStaleInvalidAndScopesSensor(t *testing.T) {
	r := NewFuelLow(1, 1, FuelLowConfig{Threshold: 10, Recovery: 15, MinimumSamples: 2, RecoverySamples: 2, FreshnessSeconds: 300}, fixedClock)
	e := fuelEvent(1, 1, 7, 9, telemetry.ReceivedTimeOnly, testNow)
	e.ReceivedAt = testNow.Add(-6 * time.Minute)
	if err := r.Validate(e); err == nil {
		t.Fatal("stale accepted")
	}
	e = fuelEvent(2, 1, 7, 101, telemetry.ReceivedTimeOnly, testNow)
	if err := r.Validate(e); err == nil {
		t.Fatal("invalid percent accepted")
	}
	a, _ := evaluate(t, r, fuelEvent(3, 1, 7, 9, telemetry.ReceivedTimeOnly, testNow), nil)
	b, _ := evaluate(t, r, fuelEvent(4, 1, 8, 9, telemetry.ReceivedTimeOnly, testNow), nil)
	if a.ScopeKey == b.ScopeKey {
		t.Fatal("different sensors share scope")
	}
}

func TestFuelDropTimestampPolicy(t *testing.T) {
	r := NewFuelDrop(2, 1, FuelDropConfig{MinimumDrop: 8, WindowSeconds: 300, MinimumSamples: 3, FreshnessSeconds: 300, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 180, MaximumCorrelationLagSeconds: 300}, fixedClock)
	qualities := []telemetry.TimestampQuality{telemetry.CorrelatedTime, telemetry.ReceivedTimeOnly, telemetry.UnknownTime}
	for i, q := range qualities {
		out, _ := evaluate(t, r, fuelEvent(int64(i+1), 1, 7, 70, q, testNow.Add(time.Duration(i)*time.Minute)), nil)
		var stored fuelDropState
		_ = json.Unmarshal(out.State, &stored)
		if out.Action != alertengine.Observe || (q == telemetry.CorrelatedTime && len(stored.Samples) != 1) || (q != telemetry.CorrelatedTime && stored.Confidence != "INSUFFICIENT") {
			t.Fatalf("quality %s affected state", q)
		}
	}
}

func TestFuelDropCorrelatedAndMixedConfidence(t *testing.T) {
	r := NewFuelDrop(2, 1, FuelDropConfig{MinimumDrop: 8, WindowSeconds: 300, MinimumSamples: 3, FreshnessSeconds: 300, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 180, MaximumCorrelationLagSeconds: 300}, fixedClock)
	base := testNow.Add(-4 * time.Minute)
	var state json.RawMessage
	var out alertengine.Evaluation
	for i, p := range []float64{75, 71, 67} {
		q := telemetry.CorrelatedTime
		if i == 0 {
			q = telemetry.OriginalDeviceTime
		}
		out, state = evaluate(t, r, fuelEvent(int64(100+i), 1, 7, p, q, base.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action != alertengine.Trigger || !strings.Contains(out.Description, "Confidence MEDIUM") || len(out.RelatedEventIDs) != 3 {
		t.Fatalf("mixed result=%+v", out)
	}
	r = NewFuelDrop(2, 1, r.config, fixedClock)
	state = nil
	for i, p := range []float64{75, 71, 67} {
		out, state = evaluate(t, r, fuelEvent(int64(200+i), 1, 7, p, telemetry.OriginalDeviceTime, base.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action != alertengine.Trigger || !strings.Contains(out.Description, "Confidence HIGH") {
		t.Fatalf("original result=%+v", out)
	}
}

func TestFuelDropRejectsUnprovenCorrelationDelayAndGap(t *testing.T) {
	r := NewFuelDrop(2, 1, FuelDropConfig{MinimumDrop: 8, WindowSeconds: 300, MinimumSamples: 3, FreshnessSeconds: 300, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 90, MaximumCorrelationLagSeconds: 120}, fixedClock)
	base := testNow.Add(-time.Minute)
	e := fuelEvent(1, 1, 7, 75, telemetry.CorrelatedTime, base)
	var v telemetry.NormalizedFuel
	_ = json.Unmarshal(e.Payload, &v)
	v.RawMessageID++
	e.Payload, _ = json.Marshal(v)
	out, _ := evaluate(t, r, e, nil)
	var insufficient fuelDropState
	_ = json.Unmarshal(out.State, &insufficient)
	if insufficient.Confidence != "INSUFFICIENT" || len(insufficient.Samples) != 0 {
		t.Fatal("different raw message accepted")
	}
	e = fuelEvent(2, 1, 7, 75, telemetry.CorrelatedTime, testNow.Add(-3*time.Minute))
	out, _ = evaluate(t, r, e, nil)
	_ = json.Unmarshal(out.State, &insufficient)
	if insufficient.Confidence != "INSUFFICIENT" || len(insufficient.Samples) != 0 {
		t.Fatal("delayed correlation accepted")
	}
	first := fuelEvent(3, 1, 7, 75, telemetry.CorrelatedTime, testNow.Add(-2*time.Minute))
	out, state := evaluate(t, r, first, nil)
	second := fuelEvent(4, 1, 7, 60, telemetry.CorrelatedTime, testNow.Add(-20*time.Second))
	out, state = evaluate(t, r, second, state)
	var stored fuelDropState
	_ = json.Unmarshal(state, &stored)
	if len(stored.Samples) != 1 {
		t.Fatalf("gap did not reset samples: %+v", stored)
	}
	duplicateRaw := fuelEvent(5, 1, 7, 50, telemetry.CorrelatedTime, testNow.Add(-10*time.Second))
	duplicateRaw.RawMessageID = second.RawMessageID
	var duplicateValue telemetry.NormalizedFuel
	_ = json.Unmarshal(duplicateRaw.Payload, &duplicateValue)
	duplicateValue.RawMessageID = *second.RawMessageID
	duplicateRaw.Payload, _ = json.Marshal(duplicateValue)
	out, duplicateState := evaluate(t, r, duplicateRaw, state)
	var duplicateStored fuelDropState
	_ = json.Unmarshal(duplicateState, &duplicateStored)
	if len(duplicateStored.Samples) != 1 {
		t.Fatal("duplicate raw message accepted")
	}
}

func TestFuelDropRecoveryAndNewEpisodeState(t *testing.T) {
	r := NewFuelDrop(2, 1, FuelDropConfig{MinimumDrop: 8, WindowSeconds: 300, MinimumSamples: 3, FreshnessSeconds: 600, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 180, MaximumCorrelationLagSeconds: 600}, fixedClock)
	base := testNow.Add(-5 * time.Minute)
	var state json.RawMessage
	var out alertengine.Evaluation
	for i, p := range []float64{75, 71, 67} {
		out, state = evaluate(t, r, fuelEvent(int64(300+i), 1, 7, p, telemetry.OriginalDeviceTime, base.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action != alertengine.Trigger {
		t.Fatal("initial episode not triggered")
	}
	for i, p := range []float64{67, 67.5, 67, 67.2, 67.1} {
		out, state = evaluate(t, r, fuelEvent(int64(310+i), 1, 7, p, telemetry.OriginalDeviceTime, base.Add(time.Duration(5+i)*time.Minute)), state)
	}
	if out.Action != alertengine.Recover {
		t.Fatalf("recovery=%s", out.Action)
	}
	state = nil
	for i, p := range []float64{80, 75, 72} {
		out, state = evaluate(t, r, fuelEvent(int64(320+i), 1, 7, p, telemetry.OriginalDeviceTime, testNow.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action != alertengine.Trigger {
		t.Fatal("new episode state not triggered")
	}
}

func TestFuelDropTriggerBoundariesOrderingAndDuplicate(t *testing.T) {
	r := NewFuelDrop(2, 1, FuelDropConfig{MinimumDrop: 8, WindowSeconds: 300, MinimumSamples: 3, FreshnessSeconds: 300, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 180, MaximumCorrelationLagSeconds: 300}, fixedClock)
	base := testNow.Add(-4 * time.Minute)
	var state json.RawMessage
	var out alertengine.Evaluation
	for i, p := range []float64{75, 71, 67} {
		out, state = evaluate(t, r, fuelEvent(int64(i+1), 1, 7, p, telemetry.OriginalDeviceTime, base.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action != alertengine.Trigger {
		t.Fatalf("8 point drop=%s", out.Action)
	}
	dup, _ := evaluate(t, r, fuelEvent(3, 1, 7, 67, telemetry.OriginalDeviceTime, base.Add(2*time.Minute)), state)
	if dup.Action != alertengine.Observe {
		t.Fatalf("duplicate=%s", dup.Action)
	}
	r2 := NewFuelDrop(2, 1, r.config, fixedClock)
	state = nil
	for i, p := range []float64{75, 72, 68} {
		out, state = evaluate(t, r2, fuelEvent(int64(10+i), 1, 7, p, telemetry.OriginalDeviceTime, base.Add(time.Duration(i)*time.Minute)), state)
	}
	if out.Action == alertengine.Trigger {
		t.Fatal("7 point drop triggered")
	}
	late := fuelEvent(20, 1, 7, 50, telemetry.OriginalDeviceTime, base)
	out, _ = evaluate(t, r2, late, state)
	if out.Action == alertengine.Trigger {
		t.Fatal("out-of-order triggered")
	}
}

func TestOverSpeedDurationGapRecovery(t *testing.T) {
	r := NewOverSpeed(3, 1, OverSpeedConfig{Threshold: 80, Recovery: 75, MinimumDurationSeconds: 10, MaximumGapSeconds: 15, FreshnessSeconds: 300, MaximumSpeed: 250}, fixedClock)
	base := testNow.Add(-time.Minute)
	var state json.RawMessage
	out, state := evaluate(t, r, gpsEvent(1, 1, 9, 81, base, true), state)
	if out.Action != alertengine.Observe {
		t.Fatal(out.Action)
	}
	out, state = evaluate(t, r, gpsEvent(2, 1, 9, 90, base.Add(9*time.Second), true), state)
	if out.Action == alertengine.Trigger {
		t.Fatal("triggered before 10 seconds")
	}
	out, state = evaluate(t, r, gpsEvent(3, 1, 9, 85, base.Add(10*time.Second), true), state)
	if out.Action != alertengine.Trigger {
		t.Fatalf("10 sec=%s", out.Action)
	}
	out, _ = evaluate(t, r, gpsEvent(4, 1, 9, 75, base.Add(12*time.Second), true), state)
	if out.Action != alertengine.Recover {
		t.Fatalf("75 recovery=%s", out.Action)
	}
	r = NewOverSpeed(3, 1, r.config, fixedClock)
	out, state = evaluate(t, r, gpsEvent(10, 1, 9, 81, base, true), nil)
	out, _ = evaluate(t, r, gpsEvent(11, 1, 9, 90, base.Add(20*time.Second), true), state)
	if out.Action == alertengine.Trigger {
		t.Fatal("sampling gap triggered")
	}
}

func TestOverSpeedValidationAndThreshold(t *testing.T) {
	r := NewOverSpeed(3, 1, OverSpeedConfig{Threshold: 80, Recovery: 75, MinimumDurationSeconds: 10, MaximumGapSeconds: 15, FreshnessSeconds: 300, MaximumSpeed: 250}, fixedClock)
	base := testNow.Add(-time.Minute)
	for _, speed := range []float64{79, 80} {
		out, _ := evaluate(t, r, gpsEvent(int64(speed), 1, 9, speed, base, true), nil)
		if out.Action == alertengine.Trigger {
			t.Fatalf("%.0f triggered", speed)
		}
	}
	e := gpsEvent(1, 1, 9, 81, base, false)
	if err := r.Validate(e); err == nil {
		t.Fatal("invalid fix accepted")
	}
	e = gpsEvent(2, 1, 9, 251, base, true)
	if err := r.Validate(e); err == nil {
		t.Fatal("impossible speed accepted")
	}
	e = gpsEvent(3, 1, 9, 81, base, true)
	e.ReceivedAt = testNow.Add(-6 * time.Minute)
	if err := r.Validate(e); err == nil {
		t.Fatal("stale GPS accepted")
	}
}

func crashEvent(message, key, status string) alertengine.Event {
	at := testNow
	device := int64(9)
	vehicle := int64(1)
	raw := int64(88)
	counter := uint8(3)
	value := telemetry.NormalizedCrash{Kind: "CRASH_SIGNAL", SignalEventID: 77, RawMessageID: raw, DeviceID: device, IMEI: "123456789012345", VehicleID: &vehicle, MessageType: message, SequenceKey: key, CrashCounter: &counter, EventTime: &at, ReceivedAt: at, TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: telemetry.Resolved, EvidenceStatus: status, ProtocolVersion: "80201E0306", RelatedInputEventIDs: []int64{10, 11}}
	payload, _ := json.Marshal(value)
	return alertengine.Event{ID: 12, Key: "crash:77", Type: "CRASH_SIGNAL", VehicleID: vehicle, SourceType: "GPS", SourceIdentifier: value.IMEI, DeviceID: &device, RawMessageID: &raw, EventTime: &at, ReceivedAt: at, TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: telemetry.Resolved, Payload: payload}
}
func TestCrashNativeTriggersAndSupportingEvidence(t *testing.T) {
	rule := NewCrash(4, 1, CrashConfig{})
	out, _ := evaluate(t, rule, crashEvent("GTCRA", "sequence-a", "PENDING"), nil)
	if out.Action != alertengine.Trigger || !strings.Contains(out.Description, "sedang dilengkapi") || len(out.RelatedEventIDs) != 2 {
		t.Fatalf("GTCRA %+v", out)
	}
	out, _ = evaluate(t, rule, crashEvent("GTCRD", "sequence-a", "COMPLETE"), nil)
	if out.Action != alertengine.Observe || !strings.Contains(out.Description, "lengkap") {
		t.Fatalf("GTCRD %+v", out)
	}
	out, _ = evaluate(t, rule, crashEvent("GTECR", "sequence-a", "PENDING"), nil)
	if out.Action != alertengine.Observe {
		t.Fatalf("conditional GTECR=%s", out.Action)
	}
	enabled := NewCrash(4, 1, CrashConfig{GTECREnabled: true})
	out, _ = evaluate(t, enabled, crashEvent("GTECR", "sequence-a", "PENDING"), nil)
	if out.Action != alertengine.Trigger {
		t.Fatalf("enabled GTECR=%s", out.Action)
	}
}
