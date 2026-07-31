DROP INDEX IF EXISTS "idx_task_email_targets_unique_bidx";

ALTER TABLE "task_email_targets"
    DROP COLUMN IF EXISTS "email_display",
    DROP COLUMN IF EXISTS "email_bidx";

CREATE UNIQUE INDEX IF NOT EXISTS "idx_task_email_targets_unique"
    ON "task_email_targets" ("task_id", "email");
