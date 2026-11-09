// Package alertengine provides durable, vehicle-centric alert processing
// infrastructure. Business rule implementations live outside this package.
package alertengine

import (
	"encoding/json"
	"time"

	"pastrack/internal/telemetry"
)

type Event struct {
	ID                int64
	Key               string
	Type              string
	VehicleID         int64
	SourceType        string
	SourceIdentifier  string
	DeviceID          *int64
	SensorID          *int64
	GPSPositionID     *int64
	FuelReadingID     *int64
	DeviceEventID     *int64
	SensorTelemetryID *int64
	RawMessageID      *int64
	EventTime         *time.Time
	ReceivedAt        time.Time
	TimestampQuality  telemetry.TimestampQuality
	ResolutionStatus  telemetry.ResolutionStatus
	Payload           json.RawMessage
	CreatedAt         time.Time
}

func EventFromGPS(value telemetry.NormalizedGPS) Event {
	payload, _ := json.Marshal(value)
	return Event{Key: "gps-position:" + itoa(value.PositionID), Type: string(value.Kind), VehicleID: dereference(value.VehicleID), SourceType: "GPS", SourceIdentifier: value.IMEI, DeviceID: address(value.DeviceID), GPSPositionID: address(value.PositionID), EventTime: value.EventTime, ReceivedAt: value.ReceivedAt, TimestampQuality: value.TimestampQuality, ResolutionStatus: value.ResolutionStatus, Payload: payload}
}

func EventFromFuel(value telemetry.NormalizedFuel) Event {
	payload, _ := json.Marshal(value)
	return Event{Key: "fuel-reading:" + itoa(value.FuelReadingID), Type: string(value.Kind), VehicleID: dereference(value.VehicleID), SourceType: "BLE", SourceIdentifier: value.MAC, DeviceID: address(value.ReceiverDeviceID), SensorID: address(value.SensorID), FuelReadingID: address(value.FuelReadingID), SensorTelemetryID: address(value.TelemetryID), RawMessageID: address(value.RawMessageID), EventTime: value.EventTime, ReceivedAt: value.ReceivedAt, TimestampQuality: value.TimestampQuality, ResolutionStatus: value.ResolutionStatus, Payload: payload}
}

func EventFromCrash(value telemetry.NormalizedCrash) Event {
	payload, _ := json.Marshal(value)
	return Event{Key: "crash-signal:" + itoa(value.SignalEventID), Type: "CRASH_SIGNAL", VehicleID: dereference(value.VehicleID), SourceType: "GPS", SourceIdentifier: value.IMEI, DeviceID: address(value.DeviceID), RawMessageID: address(value.RawMessageID), EventTime: value.EventTime, ReceivedAt: value.ReceivedAt, TimestampQuality: value.TimestampQuality, ResolutionStatus: value.ResolutionStatus, Payload: payload}
}

type JobState string

const (
	JobPending    JobState = "PENDING"
	JobProcessing JobState = "PROCESSING"
	JobSuccess    JobState = "SUCCESS"
	JobRetry      JobState = "RETRY"
	JobFailed     JobState = "FAILED"
)

type Job struct {
	ID, EventID, VehicleID int64
	Type, WorkerID         string
	State                  JobState
	Attempts, MaxAttempts  int
	AvailableAt            time.Time
}

type Action string

const (
	NoAction Action = "NONE"
	Trigger  Action = "TRIGGER"
	Observe  Action = "OBSERVE"
	Recover  Action = "RECOVER"
)

type Evaluation struct {
	Action          Action
	ScopeKey        string
	Severity        string
	Title           string
	Description     string
	Latitude        *float64
	Longitude       *float64
	State           json.RawMessage
	RelatedEventIDs []int64
}

type RuleEvaluator interface {
	RuleID() int64
	RuleCode() string
	RuleVersion() int64
	Accepts(Event) bool
	Validate(Event) error
	Evaluate(Event, json.RawMessage) (Evaluation, error)
}

type Registry struct{ evaluators []RuleEvaluator }

func NewRegistry(evaluators ...RuleEvaluator) Registry {
	return Registry{evaluators: append([]RuleEvaluator(nil), evaluators...)}
}

func (r Registry) Matching(event Event) []RuleEvaluator {
	values := []RuleEvaluator{}
	for _, evaluator := range r.evaluators {
		if evaluator != nil && evaluator.Accepts(event) {
			values = append(values, evaluator)
		}
	}
	return values
}

type AlertState string

const (
	AlertActive        AlertState = "active"
	AlertAcknowledged  AlertState = "acknowledged"
	AlertResolved      AlertState = "resolved"
	AlertFalsePositive AlertState = "false_positive"
)

type Actor struct {
	UserID *int64
	Type   string
}

var SystemActor = Actor{Type: "SYSTEM"}

func address[T any](value T) *T { return &value }
func dereference(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
