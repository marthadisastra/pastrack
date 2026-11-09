CREATE TABLE IF NOT EXISTS fuel_event_investigations (
 event_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
 state VARCHAR(24) NOT NULL DEFAULT 'open',
 note TEXT NOT NULL,
 receipt_number VARCHAR(100) NOT NULL DEFAULT '',
 station_name VARCHAR(200) NOT NULL DEFAULT '',
 receipt_liters DECIMAL(12,3) NULL,
 receipt_amount DECIMAL(16,2) NULL,
 row_version BIGINT NOT NULL DEFAULT 1,
 updated_by BIGINT UNSIGNED NOT NULL,
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 CONSTRAINT fk_fuel_investigation_event FOREIGN KEY(event_id) REFERENCES fuel_events(id),
 CONSTRAINT fk_fuel_investigation_user FOREIGN KEY(updated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
