CREATE INDEX IF NOT EXISTS "idx_user_referer_id_id"
    ON "user" ("referer_id", "id");

CREATE INDEX IF NOT EXISTS "idx_system_logs_type_object_id_id"
    ON "system_logs" ("type", "object_id", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_system_logs_object_created_ip"
    ON "system_logs" ("object_id", "created_at" DESC, "id" DESC)
    WHERE "ip" <> '';

CREATE INDEX IF NOT EXISTS "idx_traffic_log_subscribe_timestamp_id"
    ON "traffic_log" ("subscribe_id", "timestamp", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_subscribe_access_user_created_ip"
    ON "subscribe_access_logs" ("user_id", "created_at" DESC, "id" DESC)
    WHERE "client_ip" <> '';

CREATE INDEX IF NOT EXISTS "idx_user_auth_methods_user_type"
    ON "user_auth_methods" ("user_id", "auth_type");

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_status_id"
    ON "user_subscribe" ("status", "id");

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_quota_filter"
    ON "user_subscribe" ("subscribe_id", "status", "start_time", "expire_time", "id");

CREATE INDEX IF NOT EXISTS "idx_user_device_user_id_id"
    ON "user_device" ("user_id", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_task_type_status_created_at"
    ON "task" ("type", "status", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_task_type_created_at"
    ON "task" ("type", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_subscribe_nodes_gin"
    ON "subscribe" USING GIN (string_to_array(COALESCE("nodes", ''), ','));

CREATE INDEX IF NOT EXISTS "idx_subscribe_node_tags_gin"
    ON "subscribe" USING GIN (string_to_array(COALESCE("node_tags", ''), ','));

CREATE INDEX IF NOT EXISTS "idx_nodes_tags_gin"
    ON "nodes" USING GIN (string_to_array(COALESCE("tags", ''), ','));

CREATE INDEX IF NOT EXISTS "idx_document_tags_gin"
    ON "document" USING GIN (string_to_array(COALESCE("tags", ''), ','));

CREATE INDEX IF NOT EXISTS "idx_coupon_subscribe_gin"
    ON "coupon" USING GIN (string_to_array(COALESCE("subscribe", ''), ','));
