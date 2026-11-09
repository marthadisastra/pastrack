package telemetry

import (
	"context"
	"testing"
	"time"
)

type fakeFacts struct {
	gps        GPSFact
	fuel       FuelFact
	correlated *time.Time
}

func (f fakeFacts) GPSFact(context.Context, int64) (GPSFact, error)   { return f.gps, nil }
func (f fakeFacts) FuelFact(context.Context, int64) (FuelFact, error) { return f.fuel, nil }
func (f fakeFacts) CorrelatedGPSTime(context.Context, int64) (*time.Time, error) {
	return f.correlated, nil
}

func TestFuelTimestampQualityDoesNotInventReadingTime(t *testing.T) {
	repo := fixtureRepository()
	received := time.Date(2026, 11, 5, 2, 0, 0, 0, time.UTC)
	fact := FuelFact{ID: 1, TelemetryID: 2, SensorID: 20, DeviceID: 10, RawMessageID: 3, MAC: "AABBCCDDEEFF", IMEI: "123456789012345", ReceivedAt: received, FuelPercent: pointer(9.5)}
	n := Normalizer{Facts: fakeFacts{fuel: fact}, Resolver: Resolver{Repository: repo}}
	got, err := n.Fuel(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventTime != nil || got.TimestampQuality != ReceivedTimeOnly || got.ResolutionStatus != Unassigned || got.FuelPercent == nil || *got.FuelPercent != 9.5 {
		t.Fatalf("invented time or changed fuel: %#v", got)
	}
	correlated := received.Add(-time.Hour)
	n.Facts = fakeFacts{fuel: fact, correlated: &correlated}
	got, err = n.Fuel(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventTime == nil || !got.EventTime.Equal(correlated) || got.TimestampQuality != CorrelatedTime || got.TimestampSource != "SAME_RAW_GPS_TIME" || got.RawMessageID != fact.RawMessageID || got.ResolutionStatus != Resolved {
		t.Fatalf("correlated time lost: %#v", got)
	}
	wib := time.FixedZone("WIB", 7*60*60)
	if formatted := got.EventTime.In(wib).Format("2006-01-02 15:04:05 MST"); formatted != "2026-11-05 08:00:00 WIB" {
		t.Fatalf("WIB representation shifted incorrectly: %s", formatted)
	}
}

func TestGPSNormalizationUsesDeviceTime(t *testing.T) {
	repo := fixtureRepository()
	at := time.Date(2026, 11, 5, 2, 0, 0, 0, time.UTC)
	speed := 81.0
	n := Normalizer{Facts: fakeFacts{gps: GPSFact{ID: 1, DeviceID: 10, IMEI: "123456789012345", Latitude: -6.2, Longitude: 106.8, SpeedKMH: &speed, GPSTime: &at, ReceivedAt: at.Add(time.Minute), FixStatus: "valid"}}, Resolver: Resolver{Repository: repo}}
	got, err := n.GPS(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ResolutionStatus != Resolved || got.VehicleID == nil || *got.VehicleID != 40 || got.TimestampQuality != OriginalDeviceTime || !got.FixValid {
		t.Fatalf("unexpected GPS normalization %#v", got)
	}
}
