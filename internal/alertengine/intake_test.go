package alertengine

import (
	"encoding/json"
	"testing"
	"time"

	"pastrack/internal/telemetry"
)

func validEvent() Event {
	now := time.Date(2026, 11, 5, 1, 2, 3, 0, time.UTC)
	return Event{Key: "test:1", Type: "GPS_POSITION", VehicleID: 1, SourceType: "GPS", SourceIdentifier: "test", EventTime: &now, ReceivedAt: now.Add(time.Second), TimestampQuality: telemetry.OriginalDeviceTime, ResolutionStatus: telemetry.Resolved, Payload: json.RawMessage(`{"speed":1}`)}
}

func TestValidateEventIdentityAndTimestampSemantics(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Event)
		want   bool
	}{
		{"valid original", func(*Event) {}, true},
		{"valid correlated", func(e *Event) { e.TimestampQuality = telemetry.CorrelatedTime }, true},
		{"valid received only", func(e *Event) { e.TimestampQuality = telemetry.ReceivedTimeOnly; e.EventTime = nil }, true},
		{"valid unknown", func(e *Event) { e.TimestampQuality = telemetry.UnknownTime; e.EventTime = nil }, true},
		{"missing vehicle", func(e *Event) { e.VehicleID = 0 }, false},
		{"unassigned", func(e *Event) { e.ResolutionStatus = telemetry.Unassigned }, false},
		{"ambiguous", func(e *Event) { e.ResolutionStatus = telemetry.Ambiguous }, false},
		{"invalid", func(e *Event) { e.ResolutionStatus = telemetry.Invalid }, false},
		{"received promoted", func(e *Event) { e.TimestampQuality = telemetry.ReceivedTimeOnly }, false},
		{"unknown promoted", func(e *Event) { e.TimestampQuality = telemetry.UnknownTime }, false},
		{"original missing", func(e *Event) { e.EventTime = nil }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := validEvent()
			tc.mutate(&event)
			err := ValidateEvent(event)
			if (err == nil) != tc.want {
				t.Fatalf("ValidateEvent() error=%v, want valid=%v", err, tc.want)
			}
		})
	}
}
