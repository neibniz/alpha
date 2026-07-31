CREATE INDEX IF NOT EXISTS "idx_user_subscribe_presets_user_subscribe_sort_id"
    ON "user_subscribe_presets" ("user_id", "user_subscribe_id", "sort", "id");
