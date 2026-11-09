package telemetry

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"
)

type fakeRepository struct {
	devices     map[int64]GPSDevice
	devicesIMEI map[string]GPSDevice
	sensors     map[int64]BLESensor
	sensorsMAC  map[string]BLESensor
	assignments map[int64][]VehicleAssignment
	bindings    map[int64][]SensorBinding
}

func (f fakeRepository) GPSDevice(_ context.Context, identity GPSIdentity) (GPSDevice, error) {
	if identity.DeviceID > 0 {
		value, ok := f.devices[identity.DeviceID]
		if ok && (identity.IMEI == "" || identity.IMEI == value.IMEI) {
			return value, nil
		}
	}
	if value, ok := f.devicesIMEI[identity.IMEI]; ok {
		return value, nil
	}
	return GPSDevice{}, sql.ErrNoRows
}
func (f fakeRepository) VehicleAssignmentsAt(_ context.Context, id int64, _ time.Time) ([]VehicleAssignment, error) {
	return append([]VehicleAssignment(nil), f.assignments[id]...), nil
}
func (f fakeRepository) BLESensor(_ context.Context, identity BLEIdentity) (BLESensor, error) {
	if identity.SensorID > 0 {
		value, ok := f.sensors[identity.SensorID]
		if ok && (identity.MAC == "" || identity.MAC == value.MAC) {
			return value, nil
		}
	}
	mac, _ := NormalizeMAC(identity.MAC)
	if value, ok := f.sensorsMAC[mac]; ok {
		return value, nil
	}
	return BLESensor{}, sql.ErrNoRows
}
func (f fakeRepository) SensorBindingsAt(_ context.Context, id int64, _ time.Time) ([]SensorBinding, error) {
	return append([]SensorBinding(nil), f.bindings[id]...), nil
}

func fixtureRepository() fakeRepository {
	d := GPSDevice{ID: 10, IMEI: "123456789012345"}
	s := BLESensor{ID: 20, MAC: "AABBCCDDEEFF"}
	return fakeRepository{devices: map[int64]GPSDevice{10: d}, devicesIMEI: map[string]GPSDevice{d.IMEI: d}, sensors: map[int64]BLESensor{20: s}, sensorsMAC: map[string]BLESensor{s.MAC: s}, assignments: map[int64][]VehicleAssignment{10: {{ID: 30, VehicleID: 40}}}, bindings: map[int64][]SensorBinding{20: {{ID: 50, VehicleID: pointer(int64(40)), DeviceID: pointer(int64(10))}}}}
}

func TestGPSResolutionStates(t *testing.T) {
	at := time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		mutate   func(*fakeRepository)
		identity GPSIdentity
		at       *time.Time
		status   ResolutionStatus
		reason   string
	}{
		{"resolved by IMEI", nil, GPSIdentity{IMEI: "123456789012345"}, &at, Resolved, "HISTORICAL_ASSIGNMENT"},
		{"unknown device", nil, GPSIdentity{IMEI: "000000000000000"}, &at, Invalid, "DEVICE_NOT_FOUND"},
		{"invalid timestamp", nil, GPSIdentity{DeviceID: 10}, nil, Invalid, "EVENT_TIME_INVALID"},
		{"unassigned", func(f *fakeRepository) { f.assignments[10] = nil }, GPSIdentity{DeviceID: 10}, &at, Unassigned, "ASSIGNMENT_NOT_FOUND"},
		{"overlap", func(f *fakeRepository) {
			f.assignments[10] = append(f.assignments[10], VehicleAssignment{ID: 31, VehicleID: 41})
		}, GPSIdentity{DeviceID: 10}, &at, Ambiguous, "ASSIGNMENT_OVERLAP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := fixtureRepository()
			if tc.mutate != nil {
				tc.mutate(&f)
			}
			got, err := (Resolver{Repository: f}).ResolveGPS(context.Background(), tc.identity, tc.at)
			if err != nil || got.Status != tc.status || got.Reason != tc.reason {
				t.Fatalf("got %#v error %v", got, err)
			}
		})
	}
}

func TestBLEResolutionRequiresDirectBindingAndValidatesReceiver(t *testing.T) {
	at := time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		mutate   func(*fakeRepository)
		identity BLEIdentity
		at       *time.Time
		quality  TimestampQuality
		status   ResolutionStatus
		reason   string
	}{
		{"resolved direct binding", nil, BLEIdentity{MAC: "AA:BB:CC:DD:EE:FF", ReceiverDeviceID: 10}, &at, OriginalDeviceTime, Resolved, "HISTORICAL_SENSOR_BINDING"},
		{"unknown sensor", nil, BLEIdentity{MAC: "000000000000", ReceiverDeviceID: 10}, &at, OriginalDeviceTime, Invalid, "SENSOR_NOT_FOUND"},
		{"unknown time", nil, BLEIdentity{SensorID: 20, ReceiverDeviceID: 10}, nil, UnknownTime, Unassigned, "EVENT_TIME_UNAVAILABLE"},
		{"received time is not event time", nil, BLEIdentity{SensorID: 20, ReceiverDeviceID: 10}, &at, ReceivedTimeOnly, Unassigned, "EVENT_TIME_UNAVAILABLE"},
		{"binding missing", func(f *fakeRepository) { f.bindings[20] = nil }, BLEIdentity{SensorID: 20, ReceiverDeviceID: 10}, &at, CorrelatedTime, Unassigned, "BINDING_NOT_FOUND"},
		{"binding overlap", func(f *fakeRepository) {
			f.bindings[20] = append(f.bindings[20], SensorBinding{ID: 51, VehicleID: pointer(int64(40))})
		}, BLEIdentity{SensorID: 20, ReceiverDeviceID: 10}, &at, CorrelatedTime, Ambiguous, "BINDING_OVERLAP"},
		{"receiver mismatch", func(f *fakeRepository) {
			f.devices[11] = GPSDevice{ID: 11, IMEI: "223456789012345"}
			f.assignments[11] = []VehicleAssignment{{ID: 31, VehicleID: 40}}
		}, BLEIdentity{SensorID: 20, ReceiverDeviceID: 11}, &at, CorrelatedTime, Ambiguous, "BINDING_RECEIVER_MISMATCH"},
		{"receiver vehicle mismatch", func(f *fakeRepository) { f.assignments[10] = []VehicleAssignment{{ID: 30, VehicleID: 41}} }, BLEIdentity{SensorID: 20, ReceiverDeviceID: 10}, &at, CorrelatedTime, Ambiguous, "BINDING_VEHICLE_MISMATCH"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := fixtureRepository()
			if tc.mutate != nil {
				tc.mutate(&f)
			}
			got, err := (Resolver{Repository: f}).ResolveBLE(context.Background(), tc.identity, tc.at, tc.quality)
			if err != nil || got.Status != tc.status || got.Reason != tc.reason {
				t.Fatalf("got %#v error %v", got, err)
			}
		})
	}
}

func TestConcurrentResolutionIsStable(t *testing.T) {
	f := fixtureRepository()
	resolver := Resolver{Repository: f}
	at := time.Now().UTC()
	const count = 32
	results := make(chan Resolution, count)
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			v, e := resolver.ResolveGPS(context.Background(), GPSIdentity{DeviceID: 10}, &at)
			if e != nil {
				t.Errorf("resolve: %v", e)
				return
			}
			results <- v
		}()
	}
	group.Wait()
	close(results)
	for result := range results {
		if result.Status != Resolved || result.VehicleID == nil || *result.VehicleID != 40 {
			t.Fatalf("unstable result %#v", result)
		}
	}
}

func TestNormalizeMAC(t *testing.T) {
	if got, ok := NormalizeMAC("aa:bb:cc:dd:ee:ff"); !ok || got != "AABBCCDDEEFF" {
		t.Fatal(got, ok)
	}
	if _, ok := NormalizeMAC("not-a-mac"); ok {
		t.Fatal("invalid MAC accepted")
	}
}
