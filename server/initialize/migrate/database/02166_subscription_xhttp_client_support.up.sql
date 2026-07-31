BEGIN;

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "default" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 1 AND "user_agent" = 'default';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "shadowrocket" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 2 AND "user_agent" = 'Shadowrocket';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
            $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "clash" $proxy) -}}$new$
        ),
        $old$  xhttp-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    host: {{ $proxy.Host }}
    {{- end }}
    {{- if ne (default "" $proxy.XhttpMode) "" }}
    mode: {{ $proxy.XhttpMode }}
    {{- end }}$old$,
        $new$  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 3 AND "user_agent" = 'Clash';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 5 AND "user_agent" = 'Surge';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
            $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "stash" $proxy) -}}$new$
        ),
        $old$network: xhttp, xhttp-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, host: {{ $proxy.Host }}{{- end }}{{- if ne (default "" $proxy.XhttpMode) "" }}, mode: {{ $proxy.XhttpMode }}{{- end }} }$old$,
        $new$network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 6 AND "user_agent" = 'Stash';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "loon" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 8 AND "user_agent" = 'Loon';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "quantumult" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 9 AND "user_agent" = 'Quantumult';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxyType -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxyType) (clientSupportsProxy "quantumult-x" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 10 AND "user_agent" = 'Quantumult%20X';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "egern" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 11 AND "user_agent" = 'Egern';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "hiddify" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 12 AND "user_agent" = 'Hiddify';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$len .Proxies$old$,
            $new$len $supportedProxies$new$
        ),
        $old$range $i, $proxy := .Proxies$old$,
        $new$range $i, $proxy := $supportedProxies$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 13 AND "user_agent" = 'sing-box';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if hasKey $supportSet $proxy.Type -}}$old$,
        $new$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge-xray" $proxy) -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 15 AND "user_agent" = 'surge-xray';

COMMIT;
