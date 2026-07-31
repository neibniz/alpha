BEGIN;

ALTER TABLE "subscribe_application"
  DROP COLUMN IF EXISTS "enabled";

COMMIT;
