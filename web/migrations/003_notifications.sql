CREATE TABLE IF NOT EXISTS web_push_tokens (
 token_hash CHAR(64) NOT NULL PRIMARY KEY,
 user_id BIGINT UNSIGNED NOT NULL,
 session_hash CHAR(64) NOT NULL,
 client_id VARCHAR(64) NOT NULL,
 token TEXT NOT NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 KEY idx_push_user(user_id,client_id),
 KEY idx_push_session(session_hash),
 CONSTRAINT fk_push_user FOREIGN KEY(user_id) REFERENCES web_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS web_notifications (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT UNSIGNED NOT NULL,
 title VARCHAR(120) NOT NULL,
 body VARCHAR(1000) NOT NULL,
 target VARCHAR(200) NOT NULL DEFAULT '/#overview',
 read_at DATETIME(3) NULL,
 created_at DATETIME(3) NOT NULL,
 KEY idx_notification_user(user_id,id),
 CONSTRAINT fk_notification_user FOREIGN KEY(user_id) REFERENCES web_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS web_push_jobs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 notification_id BIGINT UNSIGNED NOT NULL,
 token_hash CHAR(64) NOT NULL,
 state VARCHAR(20) NOT NULL DEFAULT 'pending',
 attempts INT NOT NULL DEFAULT 0,
 next_attempt DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_push_job(notification_id,token_hash),
 KEY idx_push_pending(state,next_attempt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
