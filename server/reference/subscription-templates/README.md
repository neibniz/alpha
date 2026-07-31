# Subscription Templates

This directory is a reference-only copy of the built-in client subscription templates stored by database migrations. The server runtime still reads templates from `subscribe_application.subscribe_template`; files here are not loaded by the application.

Generated from the effective built-in migrations:

- `initialize/migrate/database/02132_update_subscription_templates.up.sql`
- `initialize/migrate/database/02135_refresh_subscription_protocol_fields.up.sql`
- `initialize/migrate/database/02162_refresh_xray_core_hysteria_template.up.sql`

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
