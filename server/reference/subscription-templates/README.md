# Subscription Templates

This directory is a reference-only copy of the built-in client subscription templates stored by database migrations. The server runtime still reads templates from `subscribe_application.subscribe_template`; files here are not loaded by the application.

Generated from the effective built-in migrations:

- `initialize/migrate/database/02132_update_subscription_templates.up.sql`
- `initialize/migrate/database/02135_refresh_subscription_protocol_fields.up.sql`
- `initialize/migrate/database/02162_refresh_xray_core_hysteria_template.up.sql`
- `initialize/migrate/database/02219_hy2_certificate_trust.up.sql`

## Hysteria 2 Certificate Trust

HY2 certificate pins are SHA-256 hashes of the entire DER certificate, not TLS
ClientHello fingerprints (such as `chrome`) or public-key/SPKI hashes. Colon- and
hyphen-separated hex inputs are normalized and checked for an exact 32-byte hash.

- Default/Shadowrocket URI: `pinSHA256` plus pinned-certificate mode (`insecure=1`).
  The pin still authenticates the server; clients must retain it when importing.
- Mihomo/Clash: `fingerprint`.
- Stash: `server-cert-fingerprint` (HY2 authentication uses `auth`).
- Loon: `tls-cert-sha256`.
- Surge: `server-cert-fingerprint-sha256`.
- Egern: `hysteria2.fingerprint_sha256`.
- Xray Core: `tlsSettings.pinnedPeerCertSha256`.
- SingBox 1.11/1.12 and Hiddify: `tls.certificate`, populated with matching public
  PEM certificates. These legacy templates do not use newer certificate-pin fields.

Inline serving certificates from the HY2 Xray template are made available as
`TLSCertificate` in subscription templates. Only parsed public `CERTIFICATE` PEM
blocks are exported; private keys are never included. When pins are configured,
only certificates matching those pins are trusted. For certificate-file deployments,
the server does not read files from agent paths: supply public PEM certificate(s)
through the binding's `tls_certificate` subscription variable. Legacy sing-box
templates fail with an explicit error if a pin has no matching public certificate;
they do not silently fall back to disabled verification.

Single-pin clients use the first configured pin; rotate their subscription data
together with the serving certificate. Xray can retain multiple configured pins.
The migration patches recognized blocks, including matching custom copies, without
overwriting other content. Already-customized blocks are left unchanged and should
be reviewed manually. Deploy the updated server and run its migrations together.

## Files

- `01-default.gotmpl.txt` - Default (base64)
- `02-shadowrocket.gotmpl.txt` - Shadowrocket (base64)
- `03-clash.gotmpl.yaml` - Clash (yaml)
- `04-singbox.gotmpl.json` - SingBox (json)
- `05-surge.gotmpl.conf` - Surge (conf)
- `06-stash.gotmpl.yaml` - Stash (yaml)
- `07-surfboard.gotmpl.conf` - Surfboard (conf)
- `08-loon.gotmpl.conf` - Loon (conf)
- `09-quantumult.gotmpl.conf` - Quantumult (conf)
- `10-quantumult-x.gotmpl.conf` - Quantumult X (conf)
- `11-egern.gotmpl.yaml` - Egern (yaml)
- `12-hiddify.gotmpl.json` - Hiddify (json)
- `13-singbox-1-11.gotmpl.json` - SingBox 1.11 (json)
- `14-xray-core.gotmpl.json` - Xray Core (json)
- `15-surge-xray.gotmpl.conf` - Surge Xray (conf)
