DROP INDEX IF EXISTS "idx_system_logs_activity_recent_id";

ALTER TABLE "system_logs"
    DROP COLUMN IF EXISTS "event_at",
    DROP COLUMN IF EXISTS "success",
    DROP COLUMN IF EXISTS "user_agent",
    DROP COLUMN IF EXISTS "identifier",
    DROP COLUMN IF EXISTS "method";
