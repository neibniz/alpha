ALTER TABLE subscribe_access_logs
    DROP COLUMN IF EXISTS private_data,
    DROP COLUMN IF EXISTS privacy_ref;

ALTER TABLE system_logs
    DROP COLUMN IF EXISTS private_data,
    DROP COLUMN IF EXISTS privacy_ref;
