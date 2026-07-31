-- 02167_xray_core_tls_verification_fields.up.sql
-- Stop emitting Xray-core allowInsecure in the built-in Xray Core client
-- template. Newer Xray-core removes allowInsecure after 2026-06-01; pinned
-- certificate and verify-by-name fields are the supported replacements.

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '"serverName": {{ $sni | quote }}{{ if $proxy.AllowInsecure }}, "allowInsecure": true{{ end }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}',
    '"serverName": {{ $sni | quote }}{{ if $proxy.PinnedPeerCertSha256 }}, "pinnedPeerCertSha256": {{ $proxy.PinnedPeerCertSha256 | quote }}{{ end }}{{ if $proxy.VerifyPeerCertByName }}, "verifyPeerCertByName": {{ $proxy.VerifyPeerCertByName | quote }}{{ end }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}'
  ),
  "updated_at" = NOW()
WHERE "id" = 14
  AND "subscribe_template" LIKE '%"allowInsecure": true%';

COMMIT;
