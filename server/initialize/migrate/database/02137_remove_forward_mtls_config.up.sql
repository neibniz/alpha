DELETE FROM "system"
WHERE "key" IN (
    'ForwardMTLSEnabled',
    'ForwardMTLSCACert',
    'ForwardMTLSServerCert',
    'ForwardMTLSServerKey',
    'ForwardMTLSClientCert',
    'ForwardMTLSClientKey'
);
