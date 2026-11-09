CREATE TABLE IF NOT EXISTS device_fuel_sensors (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 device_id BIGINT UNSIGNED NOT NULL,
 sensor_id VARCHAR(32) NOT NULL,
 model VARCHAR(32) NOT NULL DEFAULT 'CFS300B',
 last_raw_message_id BIGINT UNSIGNED DEFAULT NULL,
 last_seen_at DATETIME(3) NOT NULL,
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE KEY uq_device_sensor (device_id,sensor_id),
 KEY idx_sensor_seen (last_seen_at),
 CONSTRAINT fk_fuel_sensor_device FOREIGN KEY (device_id) REFERENCES gps_devices(id) ON DELETE CASCADE,
 CONSTRAINT fk_fuel_sensor_raw FOREIGN KEY (last_raw_message_id) REFERENCES device_raw_messages(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
