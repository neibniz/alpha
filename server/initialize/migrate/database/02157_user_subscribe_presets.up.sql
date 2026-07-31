CREATE TABLE IF NOT EXISTS "user_subscribe_presets" (
    "id" BIGSERIAL PRIMARY KEY,
    "user_id" BIGINT NOT NULL DEFAULT 0,
    "user_subscribe_id" BIGINT NOT NULL DEFAULT 0,
    "name" VARCHAR(100) NOT NULL DEFAULT '',
    "preset_key" VARCHAR(64) NOT NULL DEFAULT '',
    "mode" VARCHAR(20) NOT NULL DEFAULT 'custom',
    "profile_key" VARCHAR(100) NOT NULL DEFAULT '',
    "items" JSONB NOT NULL DEFAULT '[]',
    "sort" INTEGER NOT NULL DEFAULT 0,
    "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_subscribe_preset_key"
    ON "user_subscribe_presets" ("preset_key");

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_preset_user"
    ON "user_subscribe_presets" ("user_id");

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_preset_subscribe"
    ON "user_subscribe_presets" ("user_subscribe_id");
