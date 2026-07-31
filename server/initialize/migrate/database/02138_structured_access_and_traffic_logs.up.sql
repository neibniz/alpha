CREATE TABLE IF NOT EXISTS "subscribe_access_logs" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL DEFAULT 0,
  "user_subscribe_id" bigint NOT NULL DEFAULT 0,
  "token_hash" varchar(64) NOT NULL DEFAULT '',
  "token_prefix" varchar(12) NOT NULL DEFAULT '',
  "user_agent" varchar(512) NOT NULL DEFAULT '',
  "client_ip" varchar(64) NOT NULL DEFAULT '',
  "success" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "idx_subscribe_access_created" ON "subscribe_access_logs" ("created_at");
CREATE INDEX IF NOT EXISTS "idx_subscribe_access_user_created" ON "subscribe_access_logs" ("user_id", "created_at");
CREATE INDEX IF NOT EXISTS "idx_subscribe_access_user_sub_created" ON "subscribe_access_logs" ("user_subscribe_id", "created_at");

CREATE TABLE IF NOT EXISTS "daily_user_traffic" (
  "id" bigserial PRIMARY KEY,
  "date" varchar(20) NOT NULL,
  "user_id" bigint NOT NULL,
  "subscribe_id" bigint NOT NULL,
  "download" bigint NOT NULL DEFAULT 0,
  "upload" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "idx_daily_user_traffic_date_total" ON "daily_user_traffic" ("date", "total");
CREATE INDEX IF NOT EXISTS "idx_daily_user_traffic_user_sub_date" ON "daily_user_traffic" ("user_id", "subscribe_id", "date");

CREATE TABLE IF NOT EXISTS "daily_server_traffic" (
  "id" bigserial PRIMARY KEY,
  "date" varchar(20) NOT NULL,
  "server_id" bigint NOT NULL,
  "download" bigint NOT NULL DEFAULT 0,
  "upload" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "idx_daily_server_traffic_date_total" ON "daily_server_traffic" ("date", "total");
CREATE INDEX IF NOT EXISTS "idx_daily_server_traffic_server_date" ON "daily_server_traffic" ("server_id", "date");

CREATE TABLE IF NOT EXISTS "daily_traffic_stat" (
  "id" bigserial PRIMARY KEY,
  "date" varchar(20) NOT NULL UNIQUE,
  "download" bigint NOT NULL DEFAULT 0,
  "upload" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "idx_traffic_log_user_sub_timestamp" ON "traffic_log" ("user_id", "subscribe_id", "timestamp");
CREATE INDEX IF NOT EXISTS "idx_traffic_log_server_timestamp" ON "traffic_log" ("server_id", "timestamp");
CREATE INDEX IF NOT EXISTS "idx_system_logs_type_date_object_id" ON "system_logs" ("type", "date", "object_id", "id");
