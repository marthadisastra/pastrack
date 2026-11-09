CREATE TABLE IF NOT EXISTS web_users (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 username VARCHAR(64) NOT NULL,
 full_name VARCHAR(100) NOT NULL,
 email VARCHAR(190) NULL,
 phone VARCHAR(32) NULL,
 job_title VARCHAR(100) NULL,
 department VARCHAR(100) NULL,
 role ENUM('admin','operator','viewer') NOT NULL DEFAULT 'viewer',
 password_hash VARCHAR(255) NOT NULL,
 password_algo VARCHAR(30) NOT NULL DEFAULT 'bcrypt',
 is_active TINYINT(1) NOT NULL DEFAULT 1,
 auth_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
 row_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
 password_changed_at DATETIME(3) NULL,
 last_login_at DATETIME(3) NULL,
 deleted_at DATETIME(3) NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_web_username (username), UNIQUE KEY uq_web_email (email),
 KEY idx_web_user_role (role,is_active,deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS web_sessions (
 token_hash CHAR(64) NOT NULL PRIMARY KEY,
 user_id BIGINT UNSIGNED NOT NULL,
 auth_version BIGINT UNSIGNED NOT NULL,
 expires_at DATETIME(3) NOT NULL,
 created_at DATETIME(3) NOT NULL,
 KEY idx_web_session_user (user_id), KEY idx_web_session_expiry (expires_at),
 CONSTRAINT fk_web_session_user FOREIGN KEY (user_id) REFERENCES web_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
