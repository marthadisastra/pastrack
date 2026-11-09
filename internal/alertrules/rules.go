// Package alertrules contains vehicle business rule evaluators for the
// centralized alert engine. It never resolves identity or persists alerts.
package alertrules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"pastrack/internal/alertengine"
	"pastrack/internal/telemetry"
)

type Clock func() time.Time

type definition struct {
	ID, Version   int64
	Code          string
	Configuration json.RawMessage
}

// LoadEnabled loads the centrally configured Phase 5 evaluators. FUEL_DROP is
// registered defensively but only ORIGINAL_DEVICE_TIME samples are eligible.
func LoadEnabled(ctx context.Context, db *sql.DB, clock Clock) (alertengine.Registry, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,code,version,configuration FROM alert_rules WHERE enabled=1 AND code IN ('FUEL_LOW','FUEL_DROP','OVER_SPEED','CRASH') ORDER BY id`)
	if err != nil {
		return alertengine.Registry{}, err
	}
	defer rows.Close()
	var evaluators []alertengine.RuleEvaluator
	for rows.Next() {
		var d definition
		if err = rows.Scan(&d.ID, &d.Code, &d.Version, &d.Configuration); err != nil {
			return alertengine.Registry{}, err
		}
		switch d.Code {
		case "FUEL_LOW":
			var c FuelLowConfig
			defaultsFuelLow(&c)
			if err = json.Unmarshal(d.Configuration, &c); err != nil {
				return alertengine.Registry{}, err
			}
			evaluators = append(evaluators, NewFuelLow(d.ID, d.Version, c, clock))
		case "FUEL_DROP":
			var c FuelDropConfig
			defaultsFuelDrop(&c)
			if err = json.Unmarshal(d.Configuration, &c); err != nil {
				return alertengine.Registry{}, err
			}
			evaluators = append(evaluators, NewFuelDrop(d.ID, d.Version, c, clock))
		case "OVER_SPEED":
			var c OverSpeedConfig
			defaultsOverSpeed(&c)
			if err = json.Unmarshal(d.Configuration, &c); err != nil {
				return alertengine.Registry{}, err
			}
			evaluators = append(evaluators, NewOverSpeed(d.ID, d.Version, c, clock))
		case "CRASH":
			var c CrashConfig
			if err = json.Unmarshal(d.Configuration, &c); err != nil {
				return alertengine.Registry{}, err
			}
			evaluators = append(evaluators, NewCrash(d.ID, d.Version, c))
		}
	}
	if err = rows.Err(); err != nil {
		return alertengine.Registry{}, err
	}
	return alertengine.NewRegistry(evaluators...), nil
}

type CrashConfig struct {
	GTECREnabled bool `json:"gtecr_enabled"`
}
type Crash struct {
	id, version int64
	config      CrashConfig
}

func NewCrash(id, version int64, c CrashConfig) *Crash {
	return &Crash{id: id, version: version, config: c}
}
func (r *Crash) RuleID() int64                    { return r.id }
func (r *Crash) RuleCode() string                 { return "CRASH" }
func (r *Crash) RuleVersion() int64               { return r.version }
func (r *Crash) Accepts(e alertengine.Event) bool { return e.Type == "CRASH_SIGNAL" }
func (r *Crash) Validate(e alertengine.Event) error {
	var value telemetry.NormalizedCrash
	if json.Unmarshal(e.Payload, &value) != nil || e.DeviceID == nil || value.SequenceKey == "" && value.MessageType != "GTACC" || e.EventTime == nil || e.TimestampQuality != telemetry.OriginalDeviceTime || value.ResolutionStatus != telemetry.Resolved {
		return alertengine.ValidationError{Code: "CRASH_INPUT_INVALID"}
	}
	return nil
}
func (r *Crash) Evaluate(e alertengine.Event, _ json.RawMessage) (alertengine.Evaluation, error) {
	var value telemetry.NormalizedCrash
	_ = json.Unmarshal(e.Payload, &value)
	out := alertengine.Evaluation{Action: alertengine.Observe, ScopeKey: "crash:" + value.SequenceKey, Severity: "CRITICAL", Title: "Indikasi Kecelakaan Kendaraan", Description: "Perangkat GPS kendaraan mengirimkan sinyal indikasi kecelakaan. Data rekonstruksi kecelakaan sedang dilengkapi.", Latitude: value.Latitude, Longitude: value.Longitude, RelatedEventIDs: value.RelatedInputEventIDs}
	switch value.MessageType {
	case "GTCRA":
		out.Action = alertengine.Trigger
	case "GTECR":
		if r.config.GTECREnabled {
			out.Action = alertengine.Trigger
		}
	case "GTCRD":
		out.Action = alertengine.Observe
	case "GTACC":
		out.Action = alertengine.NoAction
		out.ScopeKey = "acceleration"
	default:
		return alertengine.Evaluation{}, alertengine.ValidationError{Code: "CRASH_SIGNAL_UNSUPPORTED"}
	}
	if value.EvidenceStatus == "COMPLETE" {
		out.Description = "Perangkat GPS kendaraan mengirimkan sinyal indikasi kecelakaan. Data rekonstruksi kecelakaan telah diterima lengkap."
	}
	state, _ := json.Marshal(map[string]any{"sequence_key": value.SequenceKey, "crash_counter": value.CrashCounter, "evidence_status": value.EvidenceStatus, "message_type": value.MessageType})
	out.State = state
	return out, nil
}

func now(clock Clock) time.Time {
	if clock != nil {
		return clock().UTC()
	}
	return time.Now().UTC()
}
func fresh(received time.Time, seconds int, clock Clock) bool {
	age := now(clock).Sub(received.UTC())
	return age >= -time.Minute && age <= time.Duration(seconds)*time.Second
}
func scope(sensorOrDevice *int64, prefix string) string {
	if sensorOrDevice == nil {
		return prefix + ":missing"
	}
	return fmt.Sprintf("%s:%d", prefix, *sensorOrDevice)
}
func decodeFuel(event alertengine.Event) (telemetry.NormalizedFuel, error) {
	var v telemetry.NormalizedFuel
	if event.Type != string(telemetry.FuelReading) {
		return v, fmt.Errorf("not fuel telemetry")
	}
	if err := json.Unmarshal(event.Payload, &v); err != nil {
		return v, err
	}
	return v, nil
}
func decodeGPS(event alertengine.Event) (telemetry.NormalizedGPS, error) {
	var v telemetry.NormalizedGPS
	if event.Type != string(telemetry.GPSPosition) {
		return v, fmt.Errorf("not GPS telemetry")
	}
	if err := json.Unmarshal(event.Payload, &v); err != nil {
		return v, err
	}
	return v, nil
}
func validPercent(v *float64) bool {
	return v != nil && !math.IsNaN(*v) && !math.IsInf(*v, 0) && *v >= 0 && *v <= 100
}

type FuelLowConfig struct {
	Threshold        float64 `json:"threshold_percent"`
	Recovery         float64 `json:"recovery_percent"`
	MinimumSamples   int     `json:"minimum_samples"`
	RecoverySamples  int     `json:"recovery_samples"`
	FreshnessSeconds int     `json:"freshness_seconds"`
}

func defaultsFuelLow(c *FuelLowConfig) { *c = FuelLowConfig{10, 15, 2, 2, 300} }

type fuelLowState struct {
	LastEventID   int64 `json:"last_event_id"`
	LowCount      int   `json:"low_count"`
	RecoveryCount int   `json:"recovery_count"`
	Triggered     bool  `json:"triggered"`
}
type FuelLow struct {
	id, version int64
	config      FuelLowConfig
	clock       Clock
}

func NewFuelLow(id, version int64, c FuelLowConfig, clock Clock) *FuelLow {
	return &FuelLow{id: id, version: version, config: c, clock: clock}
}
func (r *FuelLow) RuleID() int64                    { return r.id }
func (r *FuelLow) RuleCode() string                 { return "FUEL_LOW" }
func (r *FuelLow) RuleVersion() int64               { return r.version }
func (r *FuelLow) Accepts(e alertengine.Event) bool { return e.Type == string(telemetry.FuelReading) }
func (r *FuelLow) Validate(e alertengine.Event) error {
	v, err := decodeFuel(e)
	if err != nil || e.SensorID == nil || v.SensorID <= 0 || !validPercent(v.FuelPercent) || v.Validity != "valid" || !fresh(e.ReceivedAt, r.config.FreshnessSeconds, r.clock) {
		return alertengine.ValidationError{Code: "FUEL_LOW_INPUT_INVALID"}
	}
	return nil
}
func (r *FuelLow) Evaluate(e alertengine.Event, raw json.RawMessage) (alertengine.Evaluation, error) {
	v, _ := decodeFuel(e)
	s := fuelLowState{}
	_ = json.Unmarshal(raw, &s)
	out := alertengine.Evaluation{Action: alertengine.Observe, ScopeKey: scope(e.SensorID, "sensor"), Severity: "CRITICAL", Title: "BBM Kendaraan Hampir Habis", Description: "Persentase BBM kendaraan berada di bawah batas minimum 10%."}
	if s.LastEventID == e.ID {
		out.State = raw
		return out, nil
	}
	s.LastEventID = e.ID
	if *v.FuelPercent < r.config.Threshold {
		s.LowCount++
		s.RecoveryCount = 0
		if s.LowCount >= r.config.MinimumSamples {
			s.Triggered = true
			out.Action = alertengine.Trigger
		}
	} else if *v.FuelPercent >= r.config.Recovery {
		s.LowCount = 0
		s.RecoveryCount++
		if s.Triggered && s.RecoveryCount >= r.config.RecoverySamples {
			s.Triggered = false
			s.RecoveryCount = 0
			out.Action = alertengine.Recover
		}
	} else {
		s.LowCount = 0
		s.RecoveryCount = 0
	}
	out.State, _ = json.Marshal(s)
	return out, nil
}

type FuelDropConfig struct {
	MinimumDrop                  float64 `json:"minimum_drop_points"`
	WindowSeconds                int     `json:"window_seconds"`
	MinimumSamples               int     `json:"minimum_samples"`
	FreshnessSeconds             int     `json:"freshness_seconds"`
	StableRange                  float64 `json:"stable_range_points"`
	RecoverySamples              int     `json:"recovery_samples"`
	MaximumGapSeconds            int     `json:"maximum_gap_seconds"`
	MaximumCorrelationLagSeconds int     `json:"maximum_correlation_lag_seconds"`
}

func defaultsFuelDrop(c *FuelDropConfig) {
	// A drop is evaluated against the immediately previous valid reading. One
	// baseline plus the incoming sample is sufficient; waiting for three samples
	// delays detection unnecessarily for sensors reporting every five minutes.
	*c = FuelDropConfig{MinimumDrop: 8, WindowSeconds: 900, MinimumSamples: 2, FreshnessSeconds: 300, StableRange: 2, RecoverySamples: 3, MaximumGapSeconds: 600, MaximumCorrelationLagSeconds: 300}
}

type dropSample struct {
	EventID          int64                      `json:"event_id"`
	TelemetryID      int64                      `json:"telemetry_id"`
	RawMessageID     int64                      `json:"raw_message_id"`
	At               time.Time                  `json:"at"`
	Percent          float64                    `json:"percent"`
	TimestampQuality telemetry.TimestampQuality `json:"timestamp_quality"`
}
type fuelDropState struct {
	Samples     []dropSample `json:"samples"`
	Triggered   bool         `json:"triggered"`
	StableCount int          `json:"stable_count"`
	Confidence  string       `json:"confidence"`
}
type FuelDrop struct {
	id, version int64
	config      FuelDropConfig
	clock       Clock
}

func NewFuelDrop(id, version int64, c FuelDropConfig, clock Clock) *FuelDrop {
	return &FuelDrop{id: id, version: version, config: c, clock: clock}
}
func (r *FuelDrop) RuleID() int64                    { return r.id }
func (r *FuelDrop) RuleCode() string                 { return "FUEL_DROP" }
func (r *FuelDrop) RuleVersion() int64               { return r.version }
func (r *FuelDrop) Accepts(e alertengine.Event) bool { return e.Type == string(telemetry.FuelReading) }
func (r *FuelDrop) Validate(e alertengine.Event) error {
	v, err := decodeFuel(e)
	if err != nil || e.SensorID == nil || e.DeviceID == nil || v.Validity != "valid" || v.CalibrationID == nil || !validPercent(v.FuelPercent) || !fresh(e.ReceivedAt, r.config.FreshnessSeconds, r.clock) {
		return alertengine.ValidationError{Code: "FUEL_DROP_INPUT_INVALID"}
	}
	return nil
}
func (r *FuelDrop) Evaluate(e alertengine.Event, raw json.RawMessage) (alertengine.Evaluation, error) {
	v, _ := decodeFuel(e)
	out := alertengine.Evaluation{Action: alertengine.Observe, ScopeKey: scope(e.SensorID, "sensor"), Severity: "WARNING", Title: "Anomali Penurunan BBM", Description: "Terdeteksi penurunan persentase BBM yang signifikan dalam interval waktu singkat."}
	s := fuelDropState{}
	_ = json.Unmarshal(raw, &s)
	if !r.eligibleTime(e, v) {
		s.Confidence = "INSUFFICIENT"
		out.State, _ = json.Marshal(s)
		return out, nil
	}
	for _, x := range s.Samples {
		if x.EventID == e.ID || (e.RawMessageID != nil && x.RawMessageID == *e.RawMessageID) {
			out.State = raw
			return out, nil
		}
	}
	if len(s.Samples) > 0 && !e.EventTime.After(s.Samples[len(s.Samples)-1].At) {
		out.State = raw
		return out, nil
	}
	if len(s.Samples) > 0 && e.EventTime.Sub(s.Samples[len(s.Samples)-1].At) > time.Duration(r.config.MaximumGapSeconds)*time.Second {
		s.Samples = nil
		s.StableCount = 0
	}
	sample := dropSample{EventID: e.ID, TelemetryID: v.TelemetryID, At: e.EventTime.UTC(), Percent: *v.FuelPercent, TimestampQuality: e.TimestampQuality}
	if e.RawMessageID != nil {
		sample.RawMessageID = *e.RawMessageID
	}
	s.Samples = append(s.Samples, sample)
	cut := e.EventTime.Add(-time.Duration(r.config.WindowSeconds) * time.Second)
	kept := s.Samples[:0]
	for _, x := range s.Samples {
		if !x.At.Before(cut) {
			kept = append(kept, x)
		}
	}
	s.Samples = kept
	if len(s.Samples) >= r.config.MinimumSamples {
		s.Confidence = "HIGH"
		for _, x := range s.Samples {
			if x.TimestampQuality == telemetry.CorrelatedTime {
				s.Confidence = "MEDIUM"
			}
		}
		min, max := s.Samples[0].Percent, s.Samples[0].Percent
		for _, x := range s.Samples {
			if x.Percent < min {
				min = x.Percent
			}
			if x.Percent > max {
				max = x.Percent
			}
		}
		previous := s.Samples[len(s.Samples)-2].Percent
		current := s.Samples[len(s.Samples)-1].Percent
		if r.config.MinimumSamples > 2 {
			previous = s.Samples[0].Percent
		}
		drop := previous - current
		// Fuel increase is refuelling, never a fuel-drop alert. Compare the
		// newest sample only with its immediate predecessor to avoid a prior
		// refuel contaminating the direction of this incident.
		if current >= previous {
			drop = 0
		}
		if !s.Triggered && drop >= r.config.MinimumDrop {
			s.Triggered = true
			s.StableCount = 0
			out.Action = alertengine.Trigger
			if s.Confidence == "MEDIUM" {
				out.Description = fmt.Sprintf("Indikasi penurunan BBM berdasarkan waktu GPS yang berkorelasi dengan pembacaan sensor. Penurunan %.1f poin (%.1f%% ke %.1f%%) dalam %s. Confidence MEDIUM; sensor %d; device %d; vehicle %d; waktu %s sampai %s.", drop, s.Samples[0].Percent, s.Samples[len(s.Samples)-1].Percent, e.EventTime.Sub(s.Samples[0].At), v.SensorID, v.ReceiverDeviceID, e.VehicleID, s.Samples[0].At.Format(time.RFC3339), e.EventTime.Format(time.RFC3339))
			} else {
				out.Description = fmt.Sprintf("Indikasi penurunan BBM %.1f poin (%.1f%% ke %.1f%%) dalam %s berdasarkan waktu asli sensor. Confidence HIGH; sensor %d; device %d; vehicle %d; waktu %s sampai %s.", drop, s.Samples[0].Percent, s.Samples[len(s.Samples)-1].Percent, e.EventTime.Sub(s.Samples[0].At), v.SensorID, v.ReceiverDeviceID, e.VehicleID, s.Samples[0].At.Format(time.RFC3339), e.EventTime.Format(time.RFC3339))
			}
			for _, x := range s.Samples {
				out.RelatedEventIDs = append(out.RelatedEventIDs, x.EventID)
			}
		} else if s.Triggered && max-min <= r.config.StableRange {
			s.StableCount++
			if s.StableCount >= r.config.RecoverySamples {
				s.Triggered = false
				s.StableCount = 0
				out.Action = alertengine.Recover
			}
		} else if s.Triggered {
			s.StableCount = 0
		}
	}
	out.State, _ = json.Marshal(s)
	return out, nil
}

func (r *FuelDrop) eligibleTime(e alertengine.Event, v telemetry.NormalizedFuel) bool {
	if e.EventTime == nil {
		return false
	}
	switch e.TimestampQuality {
	case telemetry.OriginalDeviceTime:
		return v.TimestampSource == "SENSOR_TIME"
	case telemetry.CorrelatedTime:
		if v.TimestampSource != "SAME_RAW_GPS_TIME" || e.RawMessageID == nil || *e.RawMessageID <= 0 || v.RawMessageID != *e.RawMessageID || e.SensorTelemetryID == nil || *e.SensorTelemetryID != v.TelemetryID || e.FuelReadingID == nil || *e.FuelReadingID != v.FuelReadingID || e.DeviceID == nil || *e.DeviceID != v.ReceiverDeviceID {
			return false
		}
		lag := e.ReceivedAt.Sub(e.EventTime.UTC())
		return lag >= -time.Minute && lag <= time.Duration(r.config.MaximumCorrelationLagSeconds)*time.Second
	default:
		return false
	}
}

type OverSpeedConfig struct {
	Threshold              float64 `json:"threshold_kmh"`
	Recovery               float64 `json:"recovery_kmh"`
	MinimumDurationSeconds int     `json:"minimum_duration_seconds"`
	MaximumGapSeconds      int     `json:"maximum_gap_seconds"`
	FreshnessSeconds       int     `json:"freshness_seconds"`
	MaximumSpeed           float64 `json:"maximum_speed_kmh"`
}

func defaultsOverSpeed(c *OverSpeedConfig) { *c = OverSpeedConfig{80, 75, 10, 15, 300, 250} }

type overSpeedState struct {
	LastEventID int64      `json:"last_event_id"`
	Start       *time.Time `json:"start"`
	Last        *time.Time `json:"last"`
	MaxSpeed    float64    `json:"max_speed"`
	Triggered   bool       `json:"triggered"`
}
type OverSpeed struct {
	id, version int64
	config      OverSpeedConfig
	clock       Clock
}

func NewOverSpeed(id, version int64, c OverSpeedConfig, clock Clock) *OverSpeed {
	return &OverSpeed{id: id, version: version, config: c, clock: clock}
}
func (r *OverSpeed) RuleID() int64                    { return r.id }
func (r *OverSpeed) RuleCode() string                 { return "OVER_SPEED" }
func (r *OverSpeed) RuleVersion() int64               { return r.version }
func (r *OverSpeed) Accepts(e alertengine.Event) bool { return e.Type == string(telemetry.GPSPosition) }
func (r *OverSpeed) Validate(e alertengine.Event) error {
	v, err := decodeGPS(e)
	if err != nil || e.DeviceID == nil || !v.FixValid || v.SpeedKMH == nil || *v.SpeedKMH < 0 || *v.SpeedKMH > r.config.MaximumSpeed || v.Latitude == 0 && v.Longitude == 0 || v.Latitude < -90 || v.Latitude > 90 || v.Longitude < -180 || v.Longitude > 180 || e.TimestampQuality != telemetry.OriginalDeviceTime || e.EventTime == nil || !fresh(e.ReceivedAt, r.config.FreshnessSeconds, r.clock) {
		return alertengine.ValidationError{Code: "OVER_SPEED_INPUT_INVALID"}
	}
	return nil
}
func (r *OverSpeed) Evaluate(e alertengine.Event, raw json.RawMessage) (alertengine.Evaluation, error) {
	v, _ := decodeGPS(e)
	s := overSpeedState{}
	_ = json.Unmarshal(raw, &s)
	out := alertengine.Evaluation{Action: alertengine.Observe, ScopeKey: scope(e.DeviceID, "device"), Severity: "WARNING", Title: "Peringatan Kendaraan Melebihi Batas Kecepatan", Description: "Kendaraan terdeteksi melaju melebihi 80 km/h selama minimal 10 detik.", Latitude: &v.Latitude, Longitude: &v.Longitude}
	if s.LastEventID == e.ID {
		out.State = raw
		return out, nil
	}
	s.LastEventID = e.ID
	at := e.EventTime.UTC()
	if s.Last != nil && (!at.After(*s.Last) || at.Sub(*s.Last) > time.Duration(r.config.MaximumGapSeconds)*time.Second) {
		s.Start = nil
		s.Triggered = false
		s.MaxSpeed = 0
	}
	if *v.SpeedKMH > r.config.Threshold {
		if s.Start == nil {
			s.Start = &at
		}
		s.Last = &at
		if *v.SpeedKMH > s.MaxSpeed {
			s.MaxSpeed = *v.SpeedKMH
		}
		if at.Sub(*s.Start) >= time.Duration(r.config.MinimumDurationSeconds)*time.Second {
			s.Triggered = true
			out.Action = alertengine.Trigger
		}
	} else if *v.SpeedKMH <= r.config.Recovery {
		if s.Triggered {
			out.Action = alertengine.Recover
		}
		s = overSpeedState{LastEventID: e.ID}
	} else {
		if s.Triggered {
			s.Last = &at
		} else {
			s.Start = nil
			s.Last = &at
			s.MaxSpeed = 0
		}
	}
	out.State, _ = json.Marshal(s)
	return out, nil
}
