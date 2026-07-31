DROP INDEX IF EXISTS ux_subscribe_application_single_default;

ALTER TABLE subscribe_application
    DROP CONSTRAINT IF EXISTS ck_subscribe_application_default_enabled;
