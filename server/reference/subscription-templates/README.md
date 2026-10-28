# Subscription Templates

This directory is a reference-only copy of the built-in client subscription templates stored by database migrations. The server runtime still reads templates from `subscribe_application.subscribe_template`; files here are not loaded by the application.

Generated from the effective built-in migrations:

- `initialize/migrate/database/02132_update_subscription_templates.up.sql`
- `initialize/migrate/database/02135_refresh_subscription_protocol_fields.up.sql`
- `initialize/migrate/database/02162_refresh_xray_core_hysteria_template.up.sql`
- `initialize/migrate/database/02219_hy2_certificate_trust.up.sql`
- `initialize/migrate/database/02220_client_tls_reality_compatibility.up.sql`

## REALITY and X25519MLKEM768

ClientHello key exchange and certificate pinning are independent settings. Do not
put `X25519MLKEM768` into VLESS `encryption` or a certificate fingerprint field.

| Client | Subscription behavior and compatibility |
| --- | --- |
| Shadowrocket | TLS/REALITY URIs use `fp=chrome140`, including when the node has no fingerprint or an older fingerprint. Use a release supporting this profile; verify the imported profile in the app. |
| Default URI | REALITY uses `fp=chrome`; its precise meaning depends on the importing client. For Shadowrocket select its dedicated template. |
| Mihomo (Clash) / Stash | REALITY uses `client-fingerprint: chrome` and `reality-opts.support-x25519mlkem768: true`. Requires a core/app implementing that option. Classic Clash does not implement REALITY. |
| Xray Core / Surge Xray | Xray REALITY uses `fingerprint: chrome`. Surge Xray is a SOCKS bridge to the separately running Xray Core configuration; native Surge has no equivalent REALITY switch. |
| Loon | VLESS REALITY uses the documented `public-key`, `short-id`, `over-tls`, and `tls-profile=chrome147` parameters. Requires a release implementing that profile; the app controls its key-share behavior. |
| Quantumult X | Its current REALITY implementation automatically uses the iOS 26 Safari profile with X25519MLKEM768. Keep `fast-open=false`; no extra curve flag is needed. |
| SingBox 1.11 / 1.12, Hiddify | Templates default REALITY uTLS to `chrome`, but legacy sing-box removes the ML-KEM key share in code. No template option repairs that. Use a compatible client core or Xray Core for new REALITY servers requiring ML-KEM. |
| Egern / Quantumult / Surfboard | No verified portable ML-KEM switch is inserted. Client-native support, protocol and transport limitations still apply; these templates cannot upgrade the client's TLS implementation. |

Upstream references:
[Mihomo TLS](https://wiki.metacubex.one/config/proxies/tls/),
[Stash protocols](https://stash.wiki/en/proxy-protocols/proxy-types),
[Xray REALITY](https://xtls.github.io/en/config/transports/reality.html),
[Loon nodes](https://nsloon.app/docs/Node/),
[Quantumult X sample](https://github.com/crossutility/Quantumult-X/blob/master/sample.conf),
[sing-box 1.12 REALITY source](https://github.com/SagerNet/sing-box/blob/v1.12.0/common/tls/reality_client.go).

## TLS and Hysteria 2 Certificate Trust

Certificate pins are SHA-256 hashes of the entire DER certificate, not TLS
ClientHello fingerprints (such as `chrome`) or public-key/SPKI hashes. Colon- and
hyphen-separated hex inputs are normalized and checked for an exact 32-byte hash.

- Default/Shadowrocket URI: `pinSHA256` plus pinned-certificate mode (`insecure=1`).
  The pin still authenticates the server; clients must retain it when importing.
- Mihomo/Clash: `fingerprint`.
- Stash: `server-cert-fingerprint` (HY2 authentication uses `auth`).
- Loon: `tls-cert-sha256`.
- Surge: `server-cert-fingerprint-sha256`.
- Surfboard: `server-cert-fingerprint-sha256` for supported TLS protocols.
- Quantumult X: `tls-cert-sha256` with `tls-verification=true`; disabling
  verification would also disable the pin check.
- Egern: `hysteria2.fingerprint_sha256`.
- Xray Core: `tlsSettings.pinnedPeerCertSha256`.
- SingBox 1.11/1.12 and Hiddify: `tls.certificate`, populated with matching public
  PEM certificates. These legacy templates do not use newer certificate-pin fields.

Inline serving certificates from TLS Xray templates are made available as
`TLSCertificate` in subscription templates. Only parsed public `CERTIFICATE` PEM
blocks are exported; private keys are never included. When pins are configured,
only certificates matching those pins are trusted. For certificate-file deployments,
the server does not read files from agent paths: supply public PEM certificate(s)
through the binding's `tls_certificate` subscription variable. Legacy sing-box
templates fail with an explicit error if a pin has no matching public certificate;
they do not silently fall back to disabled verification.

Mihomo, Surge, Stash and Loon now also export pins for ordinary TLS nodes, including
VMess/Trojan and other supported TLS protocols. In Mihomo, `fingerprint` is the
certificate hash and `client-fingerprint` is the ClientHello profile: do not confuse
them. Pin configuration takes priority over an insecure-verification toggle.
Do not put the server certificate/key into Mihomo `certificate`/`private-key`:
those fields configure mutual TLS client authentication, not server trust.

Pin field references:
[Surge TLS](https://manual.nssurge.com/policies/tls.html),
[Surfboard TLS](https://getsurfboard.com/docs/profile-format/proxy/external-proxy/trojan/).

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
