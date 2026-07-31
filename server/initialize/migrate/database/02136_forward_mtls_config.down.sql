DELETE FROM "system"
WHERE "category" = 'server'
  AND "key" IN (
      'ForwardMTLSEnabled',
      'ForwardMTLSCACert',
      'ForwardMTLSServerCert',
      'ForwardMTLSServerKey',
      'ForwardMTLSClientCert',
      'ForwardMTLSClientKey'
  );
