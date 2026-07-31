-- 02168_clash_xhttp_whitespace.up.sql
-- Remove the extra blank line emitted between Clash XHTTP options and TLS
-- settings for existing databases.

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '{{ xhttpMihomoOpts $proxy 4 -}}',
    '{{ xhttpMihomoOpts $proxy 4 | trimSuffix "\n" }}'
  ),
  "updated_at" = NOW()
WHERE "id" = 3
  AND "subscribe_template" LIKE '%{{ xhttpMihomoOpts $proxy 4 -}}%';

COMMIT;
