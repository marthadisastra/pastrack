// Package telemetry resolves persisted device facts to their historical vehicle
// identity and exposes a stable, read-only normalized telemetry contract.
package telemetry

import "time"

type ResolutionStatus string

const (
	Resolved   ResolutionStatus = "RESOLVED"
	Unassigned ResolutionStatus = "UNASSIGNED"
	Ambiguous  ResolutionStatus = "AMBIGUOUS"
	Invalid    ResolutionStatus = "INVALID"
)

type TimestampQuality string

const (
	OriginalDeviceTime TimestampQuality = "ORIGINAL_DEVICE_TIME"
	CorrelatedTime     TimestampQuality = "CORRELATED_TIME"
	ReceivedTimeOnly   TimestampQuality = "RECEIVED_TIME_ONLY"
	UnknownTime        TimestampQuality = "UNKNOWN"
)

type Resolution struct {
	Status       ResolutionStatus `json:"status"`
	Reason       string           `json:"reason"`
	VehicleID    *int64           `json:"vehicle_id,omitempty"`
	AssignmentID *int64           `json:"assignment_id,omitempty"`
	BindingID    *int64           `json:"binding_id,omitempty"`
	DeviceID     *int64           `json:"device_id,omitempty"`
	SensorID     *int64           `json:"sensor_id,omitempty"`
}

type GPSIdentity struct {
	DeviceID int64
	IMEI     string
}

type BLEIdentity struct {
	SensorID         int64
	MAC              string
	ReceiverDeviceID int64
	ReceiverIMEI     string
}

type GPSDevice struct {
	ID   int64
	IMEI string
}

type BLESensor struct {
	ID  int64
	MAC string
}

type VehicleAssignment struct {
	ID        int64
	VehicleID int64
}

type SensorBinding struct {
	ID        int64
	VehicleID *int64
	DeviceID  *int64
}

type Kind string

const (
	GPSPosition Kind = "GPS_POSITION"
	FuelReading Kind = "FUEL_READING"
)

type NormalizedGPS struct {
	Kind             Kind             `json:"kind"`
	PositionID       int64            `json:"position_id"`
	DeviceID         int64            `json:"device_id"`
	IMEI             string           `json:"imei"`
	VehicleID        *int64           `json:"vehicle_id,omitempty"`
	AssignmentID     *int64           `json:"assignment_id,omitempty"`
	Latitude         float64          `json:"latitude"`
	Longitude        float64          `json:"longitude"`
	SpeedKMH         *float64         `json:"speed_kmh,omitempty"`
	EventTime        *time.Time       `json:"event_time,omitempty"`
	ReceivedAt       time.Time        `json:"received_at"`
	FixValid         bool             `json:"fix_valid"`
	ResolutionStatus ResolutionStatus `json:"resolution_status"`
	ResolutionReason string           `json:"resolution_reason"`
	TimestampQuality TimestampQuality `json:"timestamp_quality"`
}

type NormalizedFuel struct {
	Kind             Kind             `json:"kind"`
	FuelReadingID    int64            `json:"fuel_reading_id"`
	TelemetryID      int64            `json:"telemetry_id"`
	RawMessageID     int64            `json:"raw_message_id"`
	SensorID         int64            `json:"sensor_id"`
	MAC              string           `json:"mac"`
	ReceiverDeviceID int64            `json:"receiver_device_id"`
	ReceiverIMEI     string           `json:"receiver_imei"`
	VehicleID        *int64           `json:"vehicle_id,omitempty"`
	BindingID        *int64           `json:"binding_id,omitempty"`
	FuelPercent      *float64         `json:"fuel_percent,omitempty"`
	FuelLiters       *float64         `json:"fuel_liters,omitempty"`
	RawFuel          *float64         `json:"raw_fuel,omitempty"`
	RawUnit          *string          `json:"raw_unit,omitempty"`
	CalibrationID    *int64           `json:"calibration_id,omitempty"`
	Validity         string           `json:"validity"`
	EventTime        *time.Time       `json:"event_time,omitempty"`
	ReceivedAt       time.Time        `json:"received_at"`
	TimestampQuality TimestampQuality `json:"timestamp_quality"`
	TimestampSource  string           `json:"timestamp_source"`
	ResolutionStatus ResolutionStatus `json:"resolution_status"`
	ResolutionReason string           `json:"resolution_reason"`
}

type NormalizedCrash struct {
	Kind                          string `json:"kind"`
	SignalEventID                 int64  `json:"signal_event_id"`
	RawMessageID                  int64  `json:"raw_message_id"`
	DeviceID                      int64  `json:"device_id"`
	IMEI                          string `json:"imei"`
	VehicleID                     *int64 `json:"vehicle_id,omitempty"`
	AssignmentID                  *int64 `json:"assignment_id,omitempty"`
	MessageType                   string `json:"message_type"`
	SequenceKey                   string `json:"sequence_key,omitempty"`
	CrashCounter                  *uint8 `json:"crash_counter,omitempty"`
	DataType                      *uint8 `json:"data_type,omitempty"`
	TotalFrames                   *int   `json:"total_frames,omitempty"`
	FrameNumber                   *int   `json:"frame_number,omitempty"`
	TriggerDegrees                *int   `json:"trigger_degrees,omitempty"`
	EngineStatus                  *bool  `json:"engine_status,omitempty"`
	Latitude, Longitude, SpeedKMH *float64
	EventTime                     *time.Time       `json:"event_time,omitempty"`
	ReceivedAt                    time.Time        `json:"received_at"`
	TimestampQuality              TimestampQuality `json:"timestamp_quality"`
	ResolutionStatus              ResolutionStatus `json:"resolution_status"`
	ResolutionReason              string           `json:"resolution_reason"`
	EvidenceStatus                string           `json:"evidence_status"`
	FirmwareVersion               string           `json:"firmware_version,omitempty"`
	ProtocolVersion               string           `json:"protocol_version"`
	RelatedInputEventIDs          []int64          `json:"related_input_event_ids,omitempty"`
}

type GPSFact struct {
	ID, DeviceID        int64
	IMEI                string
	Latitude, Longitude float64
	SpeedKMH            *float64
	GPSTime             *time.Time
	ReceivedAt          time.Time
	FixStatus           string
}

type FuelFact struct {
	ID, TelemetryID, SensorID, DeviceID, RawMessageID int64
	MAC, IMEI, Validity                               string
	FuelPercent, FuelLiters, RawFuel                  *float64
	RawUnit                                           *string
	CalibrationID                                     *int64
	SensorTime                                        *time.Time
	ReceivedAt                                        time.Time
}
