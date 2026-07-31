DROP INDEX IF EXISTS "idx_user_subscribe_preset_one_default";

ALTER TABLE "user_subscribe_presets"
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_preset_default_enabled";

ALTER TABLE "user_subscribe_presets"
    DROP COLUMN IF EXISTS "is_default";
