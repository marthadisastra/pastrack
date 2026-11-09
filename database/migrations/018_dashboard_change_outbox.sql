CREATE TABLE IF NOT EXISTS dashboard_change_outbox (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  raw_message_id BIGINT UNSIGNED NOT NULL,
  event_kind VARCHAR(32) NOT NULL,
  aggregate_id BIGINT UNSIGNED NOT NULL,
  payload JSON NOT NULL,
  attempts INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  delivered_at DATETIME(3) NULL,
  UNIQUE KEY dashboard_outbox_raw (raw_message_id),
  INDEX dashboard_outbox_pending (delivered_at, id),
  CONSTRAINT dashboard_outbox_raw_fk FOREIGN KEY (raw_message_id) REFERENCES device_raw_messages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
