-- 02133_add_xray_core_subscription_templates.down.sql
-- Remove built-in templates added by 02133.

BEGIN;
DELETE FROM "subscribe_application" WHERE "id" IN (14, 15) AND "user_agent" IN ('xray-core', 'surge-xray');
COMMIT;
