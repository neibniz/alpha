-- Device records remain in user_device for subscription limits and online status.
-- Only the legacy device sign-in mapping is removed.
DELETE FROM "user_auth_methods"
WHERE "auth_type" = 'device';

DELETE FROM "auth_method"
WHERE "method" IN ('mobile', 'device');

-- Message logs were consolidated into system_logs by migration 02104.
DELETE FROM "system_logs"
WHERE "type" = 11;
