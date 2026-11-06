BEGIN;

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "registration_method" VARCHAR(32) NOT NULL DEFAULT '';

-- A linked email/GitHub identity does not prove how an account was created.
-- Registration events are written only after account creation succeeds.
-- Legacy events did not populate the success flag, so do not filter by it.
UPDATE "user" AS u
SET "registration_method" = r.method
FROM (
    SELECT DISTINCT ON (object_id) object_id, method
    FROM "system_logs"
    WHERE type = 31 AND object_id > 0
    ORDER BY object_id, id ASC
) AS r
WHERE u.id = r.object_id
  AND u.registration_method = ''
  AND r.method IN ('email', 'github');

COMMIT;
