# Vehicle → GPS/BLE/tank registration

## User flow

1. Add vehicle in Vehicles.
2. Select that vehicle and open Calibrate BLE / Sensor Calibration.
3. Click Register Unit.
4. Choose Sensor ID and Tank Profile ID from available server data. GPS gateway is displayed automatically from the sensor's actual reporting association; no manual Device ID is accepted.
5. Save once to connect vehicle + gateway + sensor + tank profile.

Sensors must be active, known to the server through a received report, and not held by a different current/future vehicle assignment. The associated GPS must be active and not held by another vehicle. IDs are revalidated under locks at submit time; hiding options is not the only protection. The form shows last-received time and sensor state; registering a relationship does not claim a live BLE connection.

Tank profiles can be created before assignment (`vehicle_id` NULL, migration 004). The picker includes unowned profiles and profiles already owned by the selected vehicle, excluding other vehicles' profiles and expired profiles. If none exist, the form offers Create tank profile and returns to registration after saving it. Capacity can remain unavailable; it is never assumed to be zero.

The save transaction creates/reuses the GPS vehicle assignment, closes obsolete same-sensor unowned administrative binding intervals, creates/reuses the correct BLE vehicle binding, claims the tank and updates fuel_sensors.tank_profile_id. It records an audit entry and emits realtime invalidation. Duplicate saves do not duplicate active associations. Concurrent competing saves cannot claim the same sensor/gateway/tank for different vehicles. Historical records and physical ignition configuration are not rewritten.

## API

- GET `/api/v1/vehicles/{id}/ble-options`: `data.sensors`, `data.tank_profiles`, selected vehicle. Sensor labels contain requested sensor identity and masked gateway identifier; IMEI is not exposed.
- POST `/api/v1/vehicles/{id}/ble-registration`: `{sensor_id, tank_profile_id}`. Admin/operator only. Server resolves gateway, checks availability in a READ COMMITTED transaction with row locks, returns IDs and `state: registered`. Conflicts return 409, invalid input 400, missing vehicle/profile 404, viewer 403.

Registration does not approve calibration, create liters from raw values, change physical ignition from NOT_INSTALLED, or turn stale/disconnected telemetry into valid current readings. Calibration remains a separate measured/approved process.

## Verification

Go integration tests against isolated MySQL verify available-option filtering, two competing claims, idempotent repeat, transactional rollback, actual fuel sensor tank mapping and unchanged ignition. DOM test opens the imported Register Unit control on the calibration tab and checks select options plus the exact two-ID POST. Existing navigation/design tests remain passing.

Production update changes the web/API binary and applies only migration 004 to the live v2 schema. A consistent private DB backup and previous web executable are retained. Receiver process is not restarted; it uses the same vehicle_assignments/ble_sensor_bindings/fuel_sensors tables for subsequent real reports.
