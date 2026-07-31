BEGIN;

WITH ranked_defaults AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS position
    FROM subscribe_application
    WHERE is_default = TRUE
)
UPDATE subscribe_application
SET is_default = FALSE
WHERE id IN (
    SELECT id
    FROM ranked_defaults
    WHERE position > 1
);

UPDATE subscribe_application
SET is_default = TRUE,
    enabled = TRUE
WHERE id = (SELECT MIN(id) FROM subscribe_application)
  AND NOT EXISTS (
      SELECT 1
      FROM subscribe_application
      WHERE is_default = TRUE
  );

UPDATE subscribe_application
SET enabled = TRUE
WHERE is_default = TRUE;

ALTER TABLE subscribe_application
    ADD CONSTRAINT ck_subscribe_application_default_enabled
    CHECK (NOT is_default OR enabled);

CREATE UNIQUE INDEX IF NOT EXISTS ux_subscribe_application_single_default
    ON subscribe_application (is_default)
    WHERE is_default = TRUE;

COMMIT;
