ALTER TABLE device_fuel_sensors
 ADD COLUMN IF NOT EXISTS observed_raw_value INT UNSIGNED NULL,
 ADD COLUMN IF NOT EXISTS observed_at DATETIME(3) NULL,
 ADD COLUMN IF NOT EXISTS calibration_media VARCHAR(20) NOT NULL DEFAULT 'unverified';

CREATE TABLE IF NOT EXISTS fuel_calibration_points (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 sensor_pk BIGINT UNSIGNED NOT NULL,
 level_percent DECIMAL(5,2) NOT NULL,
 raw_value INT UNSIGNED NOT NULL,
 note VARCHAR(200) DEFAULT NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_fuel_sensor_percent (sensor_pk,level_percent),
 KEY idx_fuel_sensor_raw (sensor_pk,raw_value),
 CONSTRAINT fk_fuel_calibration_sensor FOREIGN KEY (sensor_pk) REFERENCES device_fuel_sensors(id) ON DELETE CASCADE,
 CONSTRAINT chk_fuel_level_percent CHECK (level_percent BETWEEN 0 AND 100),
 CONSTRAINT chk_fuel_raw_value CHECK (raw_value BETWEEN 0 AND 65535)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
