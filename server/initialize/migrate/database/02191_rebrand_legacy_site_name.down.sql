UPDATE "system"
SET "value" = 'Perfect Panel',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "key" = 'SiteName'
  AND "value" = 'Alpha Panel';
