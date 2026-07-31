UPDATE "system"
SET "value" = '/favicon.png',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "key" = 'SiteLogo'
  AND "value" = '/favicon.svg';
