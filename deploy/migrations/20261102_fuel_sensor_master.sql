-- Fuel sensors are inventory assets. Association to a GPS is optional and explicit.
ALTER TABLE device_fuel_sensors
  MODIFY device_id BIGINT(20) UNSIGNED NULL,
  MODIFY last_seen_at DATETIME(3) NULL,
  ADD COLUMN reported_device_id BIGINT(20) UNSIGNED NULL AFTER device_id,
  ADD COLUMN is_active TINYINT(1) NOT NULL DEFAULT 1 AFTER calibration_media,
  ADD COLUMN deleted_at DATETIME(3) NULL AFTER is_active,
  ADD COLUMN row_version BIGINT(20) UNSIGNED NOT NULL DEFAULT 1 AFTER deleted_at,
  ADD UNIQUE KEY uq_fuel_sensor_id (sensor_id),
  ADD KEY idx_reported_device (reported_device_id),
  ADD CONSTRAINT fk_fuel_sensor_reported_device FOREIGN KEY (reported_device_id) REFERENCES gps_devices(id) ON DELETE SET NULL;
UPDATE device_fuel_sensors SET reported_device_id=device_id WHERE reported_device_id IS NULL;
