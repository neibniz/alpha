UPDATE "system"
SET "value" = '/favicon.svg',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "key" = 'SiteLogo'
  AND "value" = '/favicon.png';
