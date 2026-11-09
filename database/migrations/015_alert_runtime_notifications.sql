CREATE TABLE IF NOT EXISTS alert_inbox_deliveries (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 alert_id BIGINT UNSIGNED NOT NULL,
 user_id BIGINT UNSIGNED NOT NULL,
 notification_id BIGINT UNSIGNED NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_alert_inbox_user(alert_id,user_id),
 UNIQUE KEY uq_alert_inbox_notification(notification_id),
 CONSTRAINT fk_alert_inbox_alert FOREIGN KEY(alert_id) REFERENCES alert_events(id),
 CONSTRAINT fk_alert_inbox_user FOREIGN KEY(user_id) REFERENCES users(id),
 CONSTRAINT fk_alert_inbox_notification FOREIGN KEY(notification_id) REFERENCES notifications(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX ix_alert_inbox_user_created ON alert_inbox_deliveries(user_id,created_at,id);
