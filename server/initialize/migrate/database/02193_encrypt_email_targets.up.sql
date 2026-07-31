ALTER TABLE "task_email_targets"
    ALTER COLUMN "email" TYPE text,
    ADD COLUMN IF NOT EXISTS "email_bidx" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "email_display" varchar(255) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS "idx_task_email_targets_unique";

CREATE UNIQUE INDEX IF NOT EXISTS "idx_task_email_targets_unique_bidx"
    ON "task_email_targets" ("task_id", "email_bidx")
    WHERE "email_bidx" <> '';
