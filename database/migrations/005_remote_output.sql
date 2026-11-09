-- Additive remote output command ledger. No physical state is inferred from ACK.
CREATE TABLE IF NOT EXISTS remote_output_commands (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 device_id BIGINT UNSIGNED NOT NULL,
 vehicle_id BIGINT UNSIGNED NOT NULL,
 requested_by BIGINT UNSIGNED NOT NULL,
 serial CHAR(4) NOT NULL,
 request_key CHAR(64) NOT NULL,
 desired_active BOOLEAN NOT NULL,
 state VARCHAR(24) NOT NULL,
 sent_at DATETIME(3) NULL,
 acknowledged_at DATETIME(3) NULL,
 acknowledged_raw_id BIGINT UNSIGNED NULL,
 expires_at DATETIME(3) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_remote_output_serial(device_id,serial),
 UNIQUE KEY uq_remote_output_request(request_key),
 KEY ix_remote_output_device(device_id,id),
 CONSTRAINT fk_remote_output_device FOREIGN KEY(device_id) REFERENCES gps_devices(id),
 CONSTRAINT fk_remote_output_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT fk_remote_output_user FOREIGN KEY(requested_by) REFERENCES users(id),
 CONSTRAINT fk_remote_output_ack FOREIGN KEY(acknowledged_raw_id) REFERENCES device_raw_messages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
