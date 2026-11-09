package telemetry

import (
	"context"
	"fmt"
	"time"
)

type FactRepository interface {
	GPSFact(context.Context, int64) (GPSFact, error)
	FuelFact(context.Context, int64) (FuelFact, error)
	CorrelatedGPSTime(context.Context, int64) (*time.Time, error)
}

type Normalizer struct {
	Facts    FactRepository
	Resolver Resolver
}

func (n Normalizer) GPS(ctx context.Context, positionID int64) (NormalizedGPS, error) {
	if n.Facts == nil {
		return NormalizedGPS{}, fmt.Errorf("telemetry fact repository is required")
	}
	fact, err := n.Facts.GPSFact(ctx, positionID)
	if err != nil {
		return NormalizedGPS{}, err
	}
	resolution, err := n.Resolver.ResolveGPS(ctx, GPSIdentity{DeviceID: fact.DeviceID, IMEI: fact.IMEI}, fact.GPSTime)
	if err != nil {
		return NormalizedGPS{}, err
	}
	quality := UnknownTime
	if fact.GPSTime != nil {
		quality = OriginalDeviceTime
	}
	return NormalizedGPS{Kind: GPSPosition, PositionID: fact.ID, DeviceID: fact.DeviceID, IMEI: fact.IMEI, VehicleID: resolution.VehicleID, AssignmentID: resolution.AssignmentID, Latitude: fact.Latitude, Longitude: fact.Longitude, SpeedKMH: fact.SpeedKMH, EventTime: fact.GPSTime, ReceivedAt: fact.ReceivedAt, FixValid: fact.FixStatus == "valid", ResolutionStatus: resolution.Status, ResolutionReason: resolution.Reason, TimestampQuality: quality}, nil
}

func (n Normalizer) Fuel(ctx context.Context, readingID int64) (NormalizedFuel, error) {
	if n.Facts == nil {
		return NormalizedFuel{}, fmt.Errorf("telemetry fact repository is required")
	}
	fact, err := n.Facts.FuelFact(ctx, readingID)
	if err != nil {
		return NormalizedFuel{}, err
	}
	eventTime, quality := fact.SensorTime, OriginalDeviceTime
	timestampSource := "SENSOR_TIME"
	if eventTime == nil {
		eventTime, err = n.Facts.CorrelatedGPSTime(ctx, fact.RawMessageID)
		if err != nil {
			return NormalizedFuel{}, err
		}
		quality = CorrelatedTime
		timestampSource = "SAME_RAW_GPS_TIME"
	}
	if eventTime == nil {
		quality = ReceivedTimeOnly
		timestampSource = "RECEIVED_TIME_ONLY"
	}
	resolution, err := n.Resolver.ResolveBLE(ctx, BLEIdentity{SensorID: fact.SensorID, MAC: fact.MAC, ReceiverDeviceID: fact.DeviceID, ReceiverIMEI: fact.IMEI}, eventTime, quality)
	if err != nil {
		return NormalizedFuel{}, err
	}
	return NormalizedFuel{Kind: FuelReading, FuelReadingID: fact.ID, TelemetryID: fact.TelemetryID, RawMessageID: fact.RawMessageID, SensorID: fact.SensorID, MAC: fact.MAC, ReceiverDeviceID: fact.DeviceID, ReceiverIMEI: fact.IMEI, VehicleID: resolution.VehicleID, BindingID: resolution.BindingID, FuelPercent: fact.FuelPercent, FuelLiters: fact.FuelLiters, RawFuel: fact.RawFuel, RawUnit: fact.RawUnit, CalibrationID: fact.CalibrationID, Validity: fact.Validity, EventTime: eventTime, ReceivedAt: fact.ReceivedAt, TimestampQuality: quality, TimestampSource: timestampSource, ResolutionStatus: resolution.Status, ResolutionReason: resolution.Reason}, nil
}
