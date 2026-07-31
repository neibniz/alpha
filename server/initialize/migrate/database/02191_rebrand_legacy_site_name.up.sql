UPDATE "system"
SET "value" = 'Alpha Panel',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "key" = 'SiteName'
  AND "value" IN ('Perfect Panel', 'perfect-panel', 'PPanel', 'ppanel');
