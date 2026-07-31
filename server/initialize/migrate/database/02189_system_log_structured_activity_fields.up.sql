ALTER TABLE "system_logs"
    ADD COLUMN IF NOT EXISTS "method" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "identifier" varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "user_agent" varchar(512) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "success" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "event_at" bigint NOT NULL DEFAULT 0;

WITH "activity_source" AS (
    SELECT
        "id",
        "type",
        CASE "type"
            WHEN 20 THEN NULLIF("content"::jsonb->>'client_ip', '')
            WHEN 30 THEN NULLIF("content"::jsonb->>'login_ip', '')
            WHEN 31 THEN NULLIF("content"::jsonb->>'register_ip', '')
            WHEN 35 THEN NULLIF("content"::jsonb->>'logout_ip', '')
            ELSE NULL
        END AS "next_ip",
        CASE "type"
            WHEN 30 THEN NULLIF("content"::jsonb->>'method', '')
            WHEN 31 THEN NULLIF("content"::jsonb->>'auth_method', '')
            ELSE NULL
        END AS "next_method",
        CASE "type"
            WHEN 31 THEN NULLIF("content"::jsonb->>'identifier', '')
            ELSE NULL
        END AS "raw_identifier",
        NULLIF("content"::jsonb->>'user_agent', '') AS "raw_user_agent",
        CASE
            WHEN "type" = 30 THEN NULLIF("content"::jsonb->>'success', '')::boolean
            ELSE NULL
        END AS "next_success",
        CASE
            WHEN "type" IN (30, 31, 35) THEN NULLIF("content"::jsonb->>'timestamp', '')::bigint
            ELSE NULL
        END AS "next_event_at"
    FROM "system_logs"
    WHERE "type" IN (20, 30, 31, 35)
      AND "content" <> ''
),
"activity_display" AS (
    SELECT
        "id",
        "next_ip",
        "next_method",
        CASE
            WHEN "raw_identifier" IS NULL THEN NULL
            WHEN POSITION('@' IN "raw_identifier") > 0 THEN
                CASE
                    WHEN CHAR_LENGTH(SPLIT_PART("raw_identifier", '@', 1)) <= 2 THEN
                        REPEAT('*', CHAR_LENGTH(SPLIT_PART("raw_identifier", '@', 1))) || '@' || SPLIT_PART("raw_identifier", '@', 2)
                    ELSE
                        LEFT(SPLIT_PART("raw_identifier", '@', 1), 1) ||
                        REPEAT('*', GREATEST(CHAR_LENGTH(SPLIT_PART("raw_identifier", '@', 1)) - 2, 0)) ||
                        RIGHT(SPLIT_PART("raw_identifier", '@', 1), 1) ||
                        '@' || SPLIT_PART("raw_identifier", '@', 2)
                END
            WHEN CHAR_LENGTH("raw_identifier") <= 4 THEN REPEAT('*', CHAR_LENGTH("raw_identifier"))
            WHEN CHAR_LENGTH("raw_identifier") <= 8 THEN
                LEFT("raw_identifier", 1) || REPEAT('*', GREATEST(CHAR_LENGTH("raw_identifier") - 2, 0)) || RIGHT("raw_identifier", 1)
            ELSE LEFT("raw_identifier", 4) || '...' || RIGHT("raw_identifier", 4)
        END AS "next_identifier",
        CASE
            WHEN "raw_user_agent" IS NULL THEN NULL
            WHEN POSITION('/' IN SPLIT_PART("raw_user_agent", ' ', 1)) > 0 THEN
                REGEXP_REPLACE(SPLIT_PART("raw_user_agent", ' ', 1), '/.*$', '/***')
            WHEN CHAR_LENGTH(SPLIT_PART("raw_user_agent", ' ', 1)) > 32 THEN
                LEFT(SPLIT_PART("raw_user_agent", ' ', 1), 4) || '...' || RIGHT(SPLIT_PART("raw_user_agent", ' ', 1), 4)
            ELSE SPLIT_PART("raw_user_agent", ' ', 1)
        END AS "next_user_agent",
        "next_success",
        "next_event_at"
    FROM "activity_source"
)
UPDATE "system_logs" AS "logs"
SET
    "ip" = COALESCE(NULLIF("logs"."ip", ''), "activity_display"."next_ip", ''),
    "method" = COALESCE(NULLIF("logs"."method", ''), "activity_display"."next_method", ''),
    "identifier" = COALESCE(NULLIF("logs"."identifier", ''), "activity_display"."next_identifier", ''),
    "user_agent" = COALESCE(NULLIF("logs"."user_agent", ''), "activity_display"."next_user_agent", ''),
    "success" = CASE
        WHEN "activity_display"."next_success" IS NULL THEN "logs"."success"
        ELSE "activity_display"."next_success"
    END,
    "event_at" = COALESCE(NULLIF("logs"."event_at", 0), "activity_display"."next_event_at", 0)
FROM "activity_display"
WHERE "logs"."id" = "activity_display"."id";

CREATE INDEX IF NOT EXISTS "idx_system_logs_activity_recent_id"
    ON "system_logs" ("id" DESC)
    INCLUDE ("type", "object_id", "ip", "country_short", "region", "location", "method", "identifier", "success", "event_at", "created_at")
    WHERE "type" IN (30, 31, 35);
