ALTER TABLE "user_subscribe_presets"
    ADD COLUMN IF NOT EXISTS "is_default" BOOLEAN NOT NULL DEFAULT FALSE;

WITH ranked AS (
    SELECT "id",
           ROW_NUMBER() OVER (
               PARTITION BY "user_id", "user_subscribe_id"
               ORDER BY CASE WHEN "enabled" THEN 0 ELSE 1 END, "sort", "id"
           ) AS position
    FROM "user_subscribe_presets"
)
UPDATE "user_subscribe_presets" AS preset
SET "is_default" = TRUE,
    "enabled" = TRUE
FROM ranked
WHERE preset."id" = ranked."id"
  AND ranked.position = 1;

ALTER TABLE "user_subscribe_presets"
    ADD CONSTRAINT "chk_user_subscribe_preset_default_enabled"
    CHECK (NOT "is_default" OR "enabled");

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_subscribe_preset_one_default"
    ON "user_subscribe_presets" ("user_id", "user_subscribe_id")
    WHERE "is_default" = TRUE;
