ALTER TABLE system_logs
    ADD COLUMN IF NOT EXISTS privacy_ref varchar(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS private_data text NOT NULL DEFAULT '';

ALTER TABLE subscribe_access_logs
    ADD COLUMN IF NOT EXISTS privacy_ref varchar(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS private_data text NOT NULL DEFAULT '';

COMMENT ON COLUMN system_logs.privacy_ref IS 'Random reference used as privacy encryption associated data';
COMMENT ON COLUMN system_logs.private_data IS 'Encrypted original sensitive log fields';
COMMENT ON COLUMN subscribe_access_logs.privacy_ref IS 'Random reference used as privacy encryption associated data';
COMMENT ON COLUMN subscribe_access_logs.private_data IS 'Encrypted original client IP and user agent';
