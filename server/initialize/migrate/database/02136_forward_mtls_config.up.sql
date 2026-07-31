INSERT INTO "system" ("category", "key", "value", "type", "desc", "created_at", "updated_at")
VALUES
    ('server', 'ForwardMTLSEnabled', 'false', 'bool', 'Forward mTLS enabled', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('server', 'ForwardMTLSCACert', '', 'string', 'Forward mTLS CA certificate', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('server', 'ForwardMTLSServerCert', '', 'string', 'Forward mTLS server certificate', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('server', 'ForwardMTLSServerKey', '', 'string', 'Forward mTLS server private key', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('server', 'ForwardMTLSClientCert', '', 'string', 'Forward mTLS client certificate', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('server', 'ForwardMTLSClientKey', '', 'string', 'Forward mTLS client private key', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("key") DO NOTHING;
