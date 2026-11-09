CREATE TABLE IF NOT EXISTS dashboard_vehicle_scopes (
  user_id BIGINT UNSIGNED NOT NULL,
  vehicle_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (user_id, vehicle_id),
  INDEX dashboard_scope_vehicle (vehicle_id),
  CONSTRAINT dashboard_scope_user_fk FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT dashboard_scope_vehicle_fk FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS dashboard_replay_events (
  replay_cursor BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_kind VARCHAR(32) NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX dashboard_replay_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
