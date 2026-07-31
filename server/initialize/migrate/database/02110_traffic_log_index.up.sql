CREATE INDEX IF NOT EXISTS "idx_traffic_log_time_user_sub" ON "traffic_log" ("timestamp", "user_id", "subscribe_id");
