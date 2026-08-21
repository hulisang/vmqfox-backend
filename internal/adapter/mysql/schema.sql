-- Go-only 初始 Schema（MySQL 5.7+）。
-- 这是全新部署脚本，不执行旧 PHP 四表迁移，也不包含运行时建表逻辑。
-- 所有业务状态转换依赖 InnoDB 事务与行锁。

CREATE TABLE IF NOT EXISTS settings (
  setting_key VARCHAR(64) NOT NULL,
  setting_value TEXT NOT NULL,
  PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS admin_credentials (
  id TINYINT UNSIGNED NOT NULL,
  username VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  password_hash VARBINARY(255) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  singleton_guard TINYINT UNSIGNED GENERATED ALWAYS AS (1) STORED,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_admin_credentials_singleton (singleton_guard)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS qrcodes (
  id BIGINT NOT NULL AUTO_INCREMENT,
  pay_url TEXT NOT NULL,
  price_cent BIGINT NOT NULL,
  payment_type TINYINT NOT NULL,
  state TINYINT NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_qrcodes_lookup (payment_type, state, price_cent, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT NOT NULL AUTO_INCREMENT,
  order_id VARBINARY(128) NOT NULL,
  public_token CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  pay_id VARBINARY(255) NOT NULL,
  payment_type TINYINT NOT NULL,
  price_cent BIGINT NOT NULL,
  really_price_cent BIGINT NOT NULL,
  price_text VARCHAR(64) NOT NULL,
  really_price_text VARCHAR(64) NOT NULL,
  state TINYINT NOT NULL DEFAULT 0,
  param VARCHAR(255) NOT NULL DEFAULT '',
  pay_url TEXT NOT NULL,
  is_auto TINYINT(1) NOT NULL DEFAULT 0,
  notify_url TEXT NOT NULL,
  return_url TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  paid_at DATETIME(6) NULL,
  closed_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_orders_order_id (order_id),
  UNIQUE KEY uq_orders_public_token (public_token),
  UNIQUE KEY uq_orders_pay_id (pay_id),
  KEY idx_orders_payment_state_amount (payment_type, state, really_price_cent, id),
  KEY idx_orders_state_created (state, created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS price_locks (
  payment_type TINYINT NOT NULL,
  amount_cent BIGINT NOT NULL,
  order_id VARBINARY(128) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (payment_type, amount_cent),
  UNIQUE KEY uq_price_locks_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS payment_events (
  event_key CHAR(64) CHARACTER SET ascii NOT NULL,
  payment_type TINYINT NOT NULL,
  price_cent BIGINT NOT NULL,
  event_time VARCHAR(64) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (event_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notification_outbox (
  id CHAR(64) CHARACTER SET ascii NOT NULL,
  topic VARCHAR(64) CHARACTER SET ascii NOT NULL,
  aggregate_id VARBINARY(128) NOT NULL,
  event_key CHAR(64) CHARACTER SET ascii NOT NULL,
  payload LONGTEXT NOT NULL,
  status TINYINT NOT NULL DEFAULT 0,
  attempts INT UNSIGNED NOT NULL DEFAULT 0,
  next_attempt_at DATETIME(6) NOT NULL,
  lease_token CHAR(64) CHARACTER SET ascii NOT NULL DEFAULT '',
  locked_until DATETIME(6) NULL,
  last_error TEXT NULL,
  created_at DATETIME(6) NOT NULL,
  delivered_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_outbox_event_topic (event_key, topic),
  KEY idx_outbox_claim (status, next_attempt_at, locked_until, created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO settings (setting_key, setting_value) VALUES
  ('key', ''),
  ('notifyUrl', ''),
  ('returnUrl', ''),
  ('lastheart', '0'),
  ('lastpay', '0'),
  ('jkstate', '-1'),
  ('close', '5'),
  ('payQf', '1'),
  ('wxpay', ''),
  ('zfbpay', '');
