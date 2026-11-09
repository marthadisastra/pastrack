package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type IdentityRepository interface {
	GPSDevice(context.Context, GPSIdentity) (GPSDevice, error)
	VehicleAssignmentsAt(context.Context, int64, time.Time) ([]VehicleAssignment, error)
	BLESensor(context.Context, BLEIdentity) (BLESensor, error)
	SensorBindingsAt(context.Context, int64, time.Time) ([]SensorBinding, error)
}

type Resolver struct{ Repository IdentityRepository }

func (r Resolver) ResolveGPS(ctx context.Context, identity GPSIdentity, eventTime *time.Time) (Resolution, error) {
	if r.Repository == nil {
		return Resolution{}, fmt.Errorf("telemetry identity repository is required")
	}
	device, err := r.Repository.GPSDevice(ctx, identity)
	if err != nil {
		if err == sql.ErrNoRows {
			return Resolution{Status: Invalid, Reason: "DEVICE_NOT_FOUND"}, nil
		}
		return Resolution{}, err
	}
	result := Resolution{Status: Invalid, Reason: "EVENT_TIME_INVALID", DeviceID: pointer(device.ID)}
	if eventTime == nil || eventTime.IsZero() {
		return result, nil
	}
	assignments, err := r.Repository.VehicleAssignmentsAt(ctx, device.ID, eventTime.UTC())
	if err != nil {
		return Resolution{}, err
	}
	switch len(assignments) {
	case 0:
		result.Status, result.Reason = Unassigned, "ASSIGNMENT_NOT_FOUND"
	case 1:
		result.Status, result.Reason = Resolved, "HISTORICAL_ASSIGNMENT"
		result.VehicleID = pointer(assignments[0].VehicleID)
		result.AssignmentID = pointer(assignments[0].ID)
	default:
		result.Status, result.Reason = Ambiguous, "ASSIGNMENT_OVERLAP"
	}
	return result, nil
}

func (r Resolver) ResolveBLE(ctx context.Context, identity BLEIdentity, eventTime *time.Time, quality TimestampQuality) (Resolution, error) {
	if r.Repository == nil {
		return Resolution{}, fmt.Errorf("telemetry identity repository is required")
	}
	sensor, err := r.Repository.BLESensor(ctx, identity)
	if err != nil {
		if err == sql.ErrNoRows {
			return Resolution{Status: Invalid, Reason: "SENSOR_NOT_FOUND"}, nil
		}
		return Resolution{}, err
	}
	result := Resolution{Status: Unassigned, Reason: "EVENT_TIME_UNAVAILABLE", SensorID: pointer(sensor.ID)}
	if eventTime == nil || eventTime.IsZero() || quality == ReceivedTimeOnly || quality == UnknownTime {
		return result, nil
	}
	bindings, err := r.Repository.SensorBindingsAt(ctx, sensor.ID, eventTime.UTC())
	if err != nil {
		return Resolution{}, err
	}
	if len(bindings) == 0 {
		result.Reason = "BINDING_NOT_FOUND"
		return result, nil
	}
	if len(bindings) > 1 {
		result.Status, result.Reason = Ambiguous, "BINDING_OVERLAP"
		return result, nil
	}
	binding := bindings[0]
	result.BindingID = pointer(binding.ID)
	if binding.VehicleID == nil {
		result.Reason = "BINDING_HAS_NO_VEHICLE"
		return result, nil
	}

	// The receiver validates the direct binding; it never supplies the vehicle.
	receiver, receiverErr := r.Repository.GPSDevice(ctx, GPSIdentity{DeviceID: identity.ReceiverDeviceID, IMEI: identity.ReceiverIMEI})
	if receiverErr != nil {
		if receiverErr == sql.ErrNoRows {
			result.Status, result.Reason = Ambiguous, "RECEIVER_NOT_FOUND"
			return result, nil
		}
		return Resolution{}, receiverErr
	}
	result.DeviceID = pointer(receiver.ID)
	if binding.DeviceID != nil && *binding.DeviceID != receiver.ID {
		result.Status, result.Reason = Ambiguous, "BINDING_RECEIVER_MISMATCH"
		return result, nil
	}
	receiverAssignments, err := r.Repository.VehicleAssignmentsAt(ctx, receiver.ID, eventTime.UTC())
	if err != nil {
		return Resolution{}, err
	}
	if len(receiverAssignments) > 1 {
		result.Status, result.Reason = Ambiguous, "RECEIVER_ASSIGNMENT_OVERLAP"
		return result, nil
	}
	if len(receiverAssignments) == 1 && receiverAssignments[0].VehicleID != *binding.VehicleID {
		result.Status, result.Reason = Ambiguous, "BINDING_VEHICLE_MISMATCH"
		return result, nil
	}
	result.Status, result.Reason = Resolved, "HISTORICAL_SENSOR_BINDING"
	result.VehicleID = pointer(*binding.VehicleID)
	return result, nil
}

func NormalizeMAC(value string) (string, bool) {
	value = strings.ToUpper(strings.NewReplacer(":", "", "-", "", " ", "").Replace(value))
	if len(value) != 12 || strings.Trim(value, "0123456789ABCDEF") != "" {
		return "", false
	}
	return value, true
}

func pointer[T any](value T) *T { return &value }
