package telemetry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func isolatedDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated integration database not configured")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("integration test requires isolated pastrack_test_local")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMySQLHistoricalResolversAndNormalization(t *testing.T) {
	db := isolatedDB(t)
	ctx := context.Background()
	repo := MySQLRepository{DB: db}
	if err := repo.Validate(); err != nil {
		t.Fatal(err)
	}
	resolver := Resolver{Repository: repo}
	normalizer := Normalizer{Facts: repo, Resolver: resolver}
	suffix := fmt.Sprint(time.Now().UnixNano())
	insert := func(query string, args ...any) int64 {
		t.Helper()
		r, e := db.Exec(query, args...)
		if e != nil {
			t.Fatal(e)
		}
		id, e := r.LastInsertId()
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	model := insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('QA',?,'queclink')", "resolver-"+suffix)
	makeDevice := func(prefix string) (int64, string) {
		imei := prefix + suffix[len(suffix)-14:]
		return insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(?,?,1)", imei, model), imei
	}
	makeVehicle := func(prefix string) int64 {
		return insert("INSERT INTO vehicles(plate_number,vehicle_name,row_version) VALUES(?,'Resolver QA',1)", prefix+suffix)
	}
	d1, imei1 := makeDevice("1")
	d2, imei2 := makeDevice("2")
	unassignedDevice, _ := makeDevice("3")
	overlapDevice, _ := makeDevice("4")
	v1, v2, v3 := makeVehicle("RA-"), makeVehicle("RB-"), makeVehicle("RC-")
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cut := t0.Add(time.Hour)
	after := cut.Add(time.Minute)
	before := t0.Add(30 * time.Minute)
	a1 := insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,ends_at,provenance) VALUES(?,?,?,?, 'resolver_qa')", v1, d1, t0, cut)
	insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'resolver_qa')", v2, d1, cut)
	insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'resolver_qa')", v2, d2, cut)
	insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'resolver_overlap')", v1, overlapDevice, t0)
	insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(?,?,?,'resolver_overlap')", v3, overlapDevice, t0)

	assertGPS := func(identity GPSIdentity, at *time.Time, status ResolutionStatus, vehicle int64) {
		t.Helper()
		got, e := resolver.ResolveGPS(ctx, identity, at)
		if e != nil || got.Status != status {
			t.Fatalf("GPS got %#v error %v", got, e)
		}
		if vehicle > 0 && (got.VehicleID == nil || *got.VehicleID != vehicle) {
			t.Fatalf("GPS vehicle %#v want %d", got, vehicle)
		}
	}
	assertGPS(GPSIdentity{IMEI: imei1}, &before, Resolved, v1)
	assertGPS(GPSIdentity{DeviceID: d1}, &after, Resolved, v2)
	assertGPS(GPSIdentity{DeviceID: d1}, &before, Resolved, v1) // late arrival remains historical
	assertGPS(GPSIdentity{DeviceID: unassignedDevice}, &after, Unassigned, 0)
	assertGPS(GPSIdentity{DeviceID: overlapDevice}, &after, Ambiguous, 0)
	assertGPS(GPSIdentity{IMEI: "000000000000000"}, &after, Invalid, 0)
	assertGPS(GPSIdentity{DeviceID: d1}, nil, Invalid, 0)

	mac := fmt.Sprintf("%012X", time.Now().UnixNano()&0xffffffffffff)
	sensor := insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1)", mac, mac)
	b1 := insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,ends_at,binding_source) VALUES(?,?,?,?,?,'resolver_qa')", sensor, v1, d1, t0, cut)
	insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(?,?,?,?,'resolver_qa')", sensor, v2, d2, cut)
	unknownMAC := fmt.Sprintf("%012X", (time.Now().UnixNano()+1)&0xffffffffffff)
	noBindingSensor := insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1)", unknownMAC, unknownMAC)
	overlapMAC := fmt.Sprintf("%012X", (time.Now().UnixNano()+2)&0xffffffffffff)
	overlapSensor := insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(?,?,'CFS300B',1)", overlapMAC, overlapMAC)
	insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(?,?,?,?,'resolver_overlap')", overlapSensor, v1, d1, t0)
	insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(?,?,?,?,'resolver_overlap')", overlapSensor, v1, d1, t0)
	assertBLE := func(identity BLEIdentity, at *time.Time, quality TimestampQuality, status ResolutionStatus, vehicle int64) {
		t.Helper()
		got, e := resolver.ResolveBLE(ctx, identity, at, quality)
		if e != nil || got.Status != status {
			t.Fatalf("BLE got %#v error %v", got, e)
		}
		if vehicle > 0 && (got.VehicleID == nil || *got.VehicleID != vehicle) {
			t.Fatalf("BLE vehicle %#v want %d", got, vehicle)
		}
	}
	assertBLE(BLEIdentity{SensorID: sensor, MAC: mac, ReceiverDeviceID: d1, ReceiverIMEI: imei1}, &before, OriginalDeviceTime, Resolved, v1)
	assertBLE(BLEIdentity{SensorID: sensor, MAC: mac, ReceiverDeviceID: d2, ReceiverIMEI: imei2}, &after, CorrelatedTime, Resolved, v2)
	assertBLE(BLEIdentity{MAC: "FFFFFFFFFFFF", ReceiverDeviceID: d1}, &before, OriginalDeviceTime, Invalid, 0)
	assertBLE(BLEIdentity{SensorID: noBindingSensor, MAC: unknownMAC, ReceiverDeviceID: d1}, &before, OriginalDeviceTime, Unassigned, 0)
	assertBLE(BLEIdentity{SensorID: overlapSensor, MAC: overlapMAC, ReceiverDeviceID: d1}, &before, OriginalDeviceTime, Ambiguous, 0)
	assertBLE(BLEIdentity{SensorID: sensor, MAC: mac, ReceiverDeviceID: d2}, &before, OriginalDeviceTime, Ambiguous, 0)
	assertBLE(BLEIdentity{SensorID: sensor, MAC: mac, ReceiverDeviceID: d1}, nil, UnknownTime, Unassigned, 0)

	// A correlated GPS timestamp is exposed as correlated; reading_at remains NULL.
	payload := []byte("resolver-fixture-" + suffix)
	sum := sha256.Sum256(payload)
	raw := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,?,?)", d1, payload, sum[:], before.Add(2*time.Hour))
	position := insert("INSERT INTO gps_positions(device_id,vehicle_id,assignment_id,raw_message_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,fix_status,validation_state) VALUES(?,?,?,?,0,?,?,?,?,?,'valid','validated')", d1, v1, a1, raw, before, before.Add(2*time.Hour), -6.2, 106.8, 45)
	telemetry := insert("INSERT INTO sensor_telemetry(sensor_id,device_id,binding_id,raw_message_id,accessory_index,sensor_time,received_at,validation_state,decoder_version) VALUES(?,?,?,?,0,NULL,?,'valid_raw','resolver-qa')", sensor, d1, b1, raw, before.Add(2*time.Hour))
	fuelSensor := insert("INSERT INTO fuel_sensors(sensor_id,decoding_state) VALUES(?,'calibrated')", sensor)
	percent := 42.5
	reading := insert("INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,reading_at,received_at,raw_measurement,raw_unit,fuel_liters,fuel_percent,validity) VALUES(?,?,?,NULL,?,500,'fuel_height_raw',42.5,?,'valid')", fuelSensor, telemetry, v1, before.Add(2*time.Hour), percent)
	var positionsBefore, readingsBefore, assignmentsBefore int
	db.QueryRow("SELECT COUNT(*) FROM gps_positions").Scan(&positionsBefore)
	db.QueryRow("SELECT COUNT(*) FROM fuel_readings").Scan(&readingsBefore)
	db.QueryRow("SELECT COUNT(*) FROM vehicle_assignments").Scan(&assignmentsBefore)
	ng, e := normalizer.GPS(ctx, position)
	if e != nil || ng.VehicleID == nil || *ng.VehicleID != v1 || ng.TimestampQuality != OriginalDeviceTime {
		t.Fatalf("normalized GPS %#v %v", ng, e)
	}
	nf, e := normalizer.Fuel(ctx, reading)
	if e != nil || nf.VehicleID == nil || *nf.VehicleID != v1 || nf.TimestampQuality != CorrelatedTime || nf.EventTime == nil || !nf.EventTime.Equal(before) || nf.FuelPercent == nil || *nf.FuelPercent != percent {
		t.Fatalf("normalized fuel %#v %v", nf, e)
	}
	var readingAt sql.NullTime
	var storedPercent float64
	if e = db.QueryRow("SELECT reading_at,fuel_percent FROM fuel_readings WHERE id=?", reading).Scan(&readingAt, &storedPercent); e != nil || readingAt.Valid || storedPercent != percent {
		t.Fatalf("historical fuel changed %v %v %v", readingAt, storedPercent, e)
	}
	var positionsAfter, readingsAfter, assignmentsAfter int
	db.QueryRow("SELECT COUNT(*) FROM gps_positions").Scan(&positionsAfter)
	db.QueryRow("SELECT COUNT(*) FROM fuel_readings").Scan(&readingsAfter)
	db.QueryRow("SELECT COUNT(*) FROM vehicle_assignments").Scan(&assignmentsAfter)
	if positionsBefore != positionsAfter || readingsBefore != readingsAfter || assignmentsBefore != assignmentsAfter {
		t.Fatal("read-only resolver changed historical data")
	}

	// Same source without a device/correlated timestamp remains received-time-only.
	raw2Payload := []byte("resolver-no-time-" + suffix)
	sum2 := sha256.Sum256(raw2Payload)
	raw2 := insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at) VALUES(?,?,?,?)", d2, raw2Payload, sum2[:], after)
	telemetry2 := insert("INSERT INTO sensor_telemetry(sensor_id,device_id,binding_id,raw_message_id,accessory_index,received_at,validation_state,decoder_version) VALUES(?,?,?,?,0,?,'valid_raw','resolver-qa')", sensor, d2, nil, raw2, after)
	reading2 := insert("INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,reading_at,received_at,fuel_percent,validity) VALUES(?,?,?,NULL,?,50,'valid')", fuelSensor, telemetry2, v2, after)
	nf, e = normalizer.Fuel(ctx, reading2)
	if e != nil || nf.EventTime != nil || nf.TimestampQuality != ReceivedTimeOnly || nf.ResolutionStatus != Unassigned {
		t.Fatalf("received time promoted: %#v %v", nf, e)
	}

	// Queries used by the resolver advertise the existing history indexes.
	for _, check := range []struct {
		query string
		args  []any
		index string
	}{
		{"EXPLAIN SELECT id,vehicle_id FROM vehicle_assignments WHERE device_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC,id DESC LIMIT 2", []any{d1, before, before}, "ix_vehicle_assignments_history"},
		{"EXPLAIN SELECT id,vehicle_id,device_id FROM ble_sensor_bindings WHERE sensor_id=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?) ORDER BY starts_at DESC,id DESC LIMIT 2", []any{sensor, before, before}, "ix_ble_sensor_bindings_history"},
	} {
		rows, e := db.Query(check.query, check.args...)
		if e != nil {
			t.Fatal(e)
		}
		cols, _ := rows.Columns()
		values := make([]any, len(cols))
		pointers := make([]any, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}
		found := false
		for rows.Next() {
			if e = rows.Scan(pointers...); e != nil {
				t.Fatal(e)
			}
			for _, v := range values {
				if b, ok := v.([]byte); ok && strings.Contains(string(b), check.index) {
					found = true
				}
				if s, ok := v.(string); ok && strings.Contains(s, check.index) {
					found = true
				}
			}
		}
		rows.Close()
		if !found {
			t.Fatalf("index %s absent from EXPLAIN", check.index)
		}
	}

	// Concurrent reads return one immutable historical mapping.
	var group sync.WaitGroup
	errors := make(chan error, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			got, e := resolver.ResolveGPS(ctx, GPSIdentity{DeviceID: d1}, &before)
			if e != nil || got.VehicleID == nil || *got.VehicleID != v1 {
				errors <- fmt.Errorf("concurrent mapping %#v: %w", got, e)
			}
		}()
	}
	group.Wait()
	close(errors)
	for e := range errors {
		t.Error(e)
	}

	// Soft deletion/inactivation affects inventory, not historical resolution.
	if _, e = db.Exec("UPDATE gps_devices SET is_active=0,deleted_at=UTC_TIMESTAMP(3) WHERE id IN (?,?)", d1, d2); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("UPDATE ble_sensors SET is_active=0,deleted_at=UTC_TIMESTAMP(3) WHERE id=?", sensor); e != nil {
		t.Fatal(e)
	}
	assertGPS(GPSIdentity{DeviceID: d1, IMEI: imei1}, &before, Resolved, v1)
	assertBLE(BLEIdentity{SensorID: sensor, MAC: mac, ReceiverDeviceID: d1, ReceiverIMEI: imei1}, &before, OriginalDeviceTime, Resolved, v1)
}
