CREATE TABLE IF NOT EXISTS sensor_reported_names (
 sensor_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
 reported_name VARCHAR(80) NOT NULL,
 reported_at DATETIME(3) NOT NULL,
 received_at DATETIME(3) NOT NULL,
 raw_message_id BIGINT UNSIGNED NOT NULL,
 CONSTRAINT fk_sensor_reported_name_sensor FOREIGN KEY(sensor_id) REFERENCES ble_sensors(id),
 CONSTRAINT fk_sensor_reported_name_raw FOREIGN KEY(raw_message_id) REFERENCES device_raw_messages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
