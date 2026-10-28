-- Patch recognized client TLS/REALITY blocks; preserve unrelated and customized content.
BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_old$,
$crypto_new$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if eq $proxy.Security "reality" -}}
    {{- $_ := set $buildSecurityParams "fp" "chrome" -}}
  {{- else if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'base64'
  AND strpos("subscribe_template", 'clientSupportsProxy "default"') > 0
  AND strpos("subscribe_template", $crypto_old$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_old$,
$crypto_new$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") -}}
    {{- $_ := set $buildSecurityParams "fp" "chrome140" -}}
  {{- else if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'base64'
  AND strpos("subscribe_template", 'clientSupportsProxy "shadowrocket"') > 0
  AND strpos("subscribe_template", $crypto_old$    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "vless" }}
  {{- $encryptionStr := "" -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $encryptionStr = "none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.Encryption_Mode) "" -}}
      {{- $encParts = append $encParts $proxy.Encryption_Mode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $encryptionStr = join "." $encParts -}}
  {{- end }}
- name: {{ $proxy.Name | quote }}
  type: vless
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ $password }}
  udp: true
  tfo: true
  encryption: {{ $encryptionStr }}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}
  flow: {{ $proxy.Flow }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "httpupgrade" }}
  network: httpupgrade
  httpupgrade-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "xhttp" }}
  network: xhttp
  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 | trimSuffix "\n" }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "trojan" }}
- name: {{ $proxy.Name | quote }}
  type: trojan
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  {{- if $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- end }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and $proxy.AllowInsecure (not $proxy.PinnedPeerCertSha256) }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "tuic" }}
- name: {{ $proxy.Name | quote }}
  type: tuic
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ default "" $proxy.ServerKey }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if $proxy.DisableSNI }}
  disable-sni: true
  {{- end }}
  {{- if $proxy.ReduceRtt }}
  reduce-rtt: true
  {{- end }}
  {{- if ne (default "" $proxy.UDPRelayMode) "" }}
  udp-relay-mode: {{ $proxy.UDPRelayMode }}
  {{- end }}
  {{- if ne (default "" $proxy.CongestionController) "" }}
  congestion-controller: {{ $proxy.CongestionController }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "wireguard" }}
- name: {{ $proxy.Name | quote }}
  type: wireguard
  server: {{ $server }}
  port: {{ $proxy.Port }}
  private-key: {{ default "" $proxy.ServerKey }}
  public-key: {{ default "" $proxy.RealityPublicKey }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.Path) "" }}
  preshared-key: {{ $proxy.Path }}
  {{- end }}
  {{- if ne (default "" $proxy.RealityServerAddr) "" }}
  ip: {{ $proxy.RealityServerAddr }}
  {{- end }}
  {{- if ne (default 0 $proxy.RealityServerPort) 0 }}
  ipv6: {{ $proxy.RealityServerPort }}
  {{- end }}
{{- else if eq $proxy.Type "anytls" }}
- name: {{ $proxy.Name | quote }}
  type: anytls
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}$crypto_old$,
$crypto_new$  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- else if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if eq $proxy.Security "reality" }}
  client-fingerprint: chrome
  {{- else if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey }}
  reality-opts:
    support-x25519mlkem768: true
    public-key: {{ $proxy.RealityPublicKey | quote }}
    short-id: {{ default "" $proxy.RealityShortId | quote }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "vless" }}
  {{- $encryptionStr := "" -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $encryptionStr = "none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.Encryption_Mode) "" -}}
      {{- $encParts = append $encParts $proxy.Encryption_Mode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $encryptionStr = join "." $encParts -}}
  {{- end }}
- name: {{ $proxy.Name | quote }}
  type: vless
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ $password }}
  udp: true
  tfo: true
  encryption: {{ $encryptionStr }}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}
  flow: {{ $proxy.Flow }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "httpupgrade" }}
  network: httpupgrade
  httpupgrade-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "xhttp" }}
  network: xhttp
  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 | trimSuffix "\n" }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- else if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if eq $proxy.Security "reality" }}
  client-fingerprint: chrome
  {{- else if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    support-x25519mlkem768: true
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "trojan" }}
- name: {{ $proxy.Name | quote }}
  type: trojan
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- else if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if eq $proxy.Security "reality" }}
  client-fingerprint: chrome
  {{- else if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    support-x25519mlkem768: true
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  {{- if $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- end }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and $proxy.AllowInsecure (not $proxy.PinnedPeerCertSha256) }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "tuic" }}
- name: {{ $proxy.Name | quote }}
  type: tuic
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ default "" $proxy.ServerKey }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- else if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if $proxy.DisableSNI }}
  disable-sni: true
  {{- end }}
  {{- if $proxy.ReduceRtt }}
  reduce-rtt: true
  {{- end }}
  {{- if ne (default "" $proxy.UDPRelayMode) "" }}
  udp-relay-mode: {{ $proxy.UDPRelayMode }}
  {{- end }}
  {{- if ne (default "" $proxy.CongestionController) "" }}
  congestion-controller: {{ $proxy.CongestionController }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "wireguard" }}
- name: {{ $proxy.Name | quote }}
  type: wireguard
  server: {{ $server }}
  port: {{ $proxy.Port }}
  private-key: {{ default "" $proxy.ServerKey }}
  public-key: {{ default "" $proxy.RealityPublicKey }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.Path) "" }}
  preshared-key: {{ $proxy.Path }}
  {{- end }}
  {{- if ne (default "" $proxy.RealityServerAddr) "" }}
  ip: {{ $proxy.RealityServerAddr }}
  {{- end }}
  {{- if ne (default 0 $proxy.RealityServerPort) 0 }}
  ipv6: {{ $proxy.RealityServerPort }}
  {{- end }}
{{- else if eq $proxy.Type "anytls" }}
- name: {{ $proxy.Name | quote }}
  type: anytls
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- else if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if eq $proxy.Security "reality" }}
  client-fingerprint: chrome
  {{- else if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'yaml'
  AND strpos("subscribe_template", 'clientSupportsProxy "clash"') > 0
  AND strpos("subscribe_template", $crypto_old$  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "vless" }}
  {{- $encryptionStr := "" -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $encryptionStr = "none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.Encryption_Mode) "" -}}
      {{- $encParts = append $encParts $proxy.Encryption_Mode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $encryptionStr = join "." $encParts -}}
  {{- end }}
- name: {{ $proxy.Name | quote }}
  type: vless
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ $password }}
  udp: true
  tfo: true
  encryption: {{ $encryptionStr }}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}
  flow: {{ $proxy.Flow }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "httpupgrade" }}
  network: httpupgrade
  httpupgrade-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "xhttp" }}
  network: xhttp
  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 | trimSuffix "\n" }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "trojan" }}
- name: {{ $proxy.Name | quote }}
  type: trojan
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  {{- if $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- end }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and $proxy.AllowInsecure (not $proxy.PinnedPeerCertSha256) }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "tuic" }}
- name: {{ $proxy.Name | quote }}
  type: tuic
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ default "" $proxy.ServerKey }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if $proxy.DisableSNI }}
  disable-sni: true
  {{- end }}
  {{- if $proxy.ReduceRtt }}
  reduce-rtt: true
  {{- end }}
  {{- if ne (default "" $proxy.UDPRelayMode) "" }}
  udp-relay-mode: {{ $proxy.UDPRelayMode }}
  {{- end }}
  {{- if ne (default "" $proxy.CongestionController) "" }}
  congestion-controller: {{ $proxy.CongestionController }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "wireguard" }}
- name: {{ $proxy.Name | quote }}
  type: wireguard
  server: {{ $server }}
  port: {{ $proxy.Port }}
  private-key: {{ default "" $proxy.ServerKey }}
  public-key: {{ default "" $proxy.RealityPublicKey }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.Path) "" }}
  preshared-key: {{ $proxy.Path }}
  {{- end }}
  {{- if ne (default "" $proxy.RealityServerAddr) "" }}
  ip: {{ $proxy.RealityServerAddr }}
  {{- end }}
  {{- if ne (default 0 $proxy.RealityServerPort) 0 }}
  ipv6: {{ $proxy.RealityServerPort }}
  {{- end }}
{{- else if eq $proxy.Type "anytls" }}
- name: {{ $proxy.Name | quote }}
  type: anytls
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$  - { name: {{ $proxy.Name | quote }}, type: ss, server: {{ $server }}, port: {{ $proxy.Port }}, cipher: {{ default "aes-128-gcm" $proxy.Method }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.Obfs) "" }}, plugin: obfs, plugin-opts: { mode: {{ $proxy.Obfs }}, host: {{ default "" $proxy.ObfsHost }} }{{- end }} }
  {{- else if eq $proxy.Type "vmess" }}
  - { name: {{ $proxy.Name | quote }}, type: vmess, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, alterId: 0, cipher: auto, {{ $common }}{{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls: true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else if eq $proxy.Type "vless" }}
  - { name: {{ $proxy.Name | quote }}, type: vless, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, {{ $common }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, reality-opts: { public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow: {{ $proxy.Flow }}{{- end }} }
  {{- else if eq $proxy.Type "trojan" }}
  - { name: {{ $proxy.Name | quote }}, type: trojan, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }} }
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, auth: {{ $password | quote }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  {{- else if eq $proxy.Type "tuic" }}
  - { name: {{ $proxy.Name | quote }}, type: tuic, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ default "" $proxy.ServerKey }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni: true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt: true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode: {{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller: {{ $proxy.CongestionController }}{{- end }} }
  {{- else if eq $proxy.Type "wireguard" }}
  - { name: {{ $proxy.Name | quote }}, type: wireguard, server: {{ $server }}, port: {{ $proxy.Port }}, private-key: {{ default "" $proxy.ServerKey }}, public-key: {{ default "" $proxy.RealityPublicKey }}, {{ $common }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key: {{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip: {{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6: {{ $proxy.RealityServerPort }}{{- end }} }
  {{- else if eq $proxy.Type "anytls" }}
  - { name: {{ $proxy.Name | quote }}, type: anytls, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else }}
  - { name: {{ $proxy.Name | quote }}, type: {{ $proxy.Type }}, server: {{ $server }}, port: {{ $proxy.Port }}, {{ $common }} }$crypto_old$,
$crypto_new$  - { name: {{ $proxy.Name | quote }}, type: ss, server: {{ $server }}, port: {{ $proxy.Port }}, cipher: {{ default "aes-128-gcm" $proxy.Method }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.Obfs) "" }}, plugin: obfs, plugin-opts: { mode: {{ $proxy.Obfs }}, host: {{ default "" $proxy.ObfsHost }} }{{- end }} }
  {{- else if eq $proxy.Type "vmess" }}
  - { name: {{ $proxy.Name | quote }}, type: vmess, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, alterId: 0, cipher: auto, {{ $common }}{{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls: true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if eq $proxy.Security "reality" }}, client-fingerprint: chrome{{- else if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey }}, tls: true, reality-opts: { support-x25519mlkem768: true, public-key: {{ $proxy.RealityPublicKey | quote }}, short-id: {{ default "" $proxy.RealityShortId | quote }} }{{- end }} }
  {{- else if eq $proxy.Type "vless" }}
  - { name: {{ $proxy.Name | quote }}, type: vless, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, {{ $common }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if eq $proxy.Security "reality" }}, client-fingerprint: chrome{{- else if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, reality-opts: { support-x25519mlkem768: true, public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow: {{ $proxy.Flow }}{{- end }} }
  {{- else if eq $proxy.Type "trojan" }}
  - { name: {{ $proxy.Name | quote }}, type: trojan, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if eq $proxy.Security "reality" }}, client-fingerprint: chrome{{- else if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey }}, tls: true, reality-opts: { support-x25519mlkem768: true, public-key: {{ $proxy.RealityPublicKey | quote }}, short-id: {{ default "" $proxy.RealityShortId | quote }} }{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }} }
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, auth: {{ $password | quote }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  {{- else if eq $proxy.Type "tuic" }}
  - { name: {{ $proxy.Name | quote }}, type: tuic, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ default "" $proxy.ServerKey }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni: true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt: true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode: {{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller: {{ $proxy.CongestionController }}{{- end }} }
  {{- else if eq $proxy.Type "wireguard" }}
  - { name: {{ $proxy.Name | quote }}, type: wireguard, server: {{ $server }}, port: {{ $proxy.Port }}, private-key: {{ default "" $proxy.ServerKey }}, public-key: {{ default "" $proxy.RealityPublicKey }}, {{ $common }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key: {{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip: {{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6: {{ $proxy.RealityServerPort }}{{- end }} }
  {{- else if eq $proxy.Type "anytls" }}
  - { name: {{ $proxy.Name | quote }}, type: anytls, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if eq $proxy.Security "reality" }}, client-fingerprint: chrome{{- else if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else }}
  - { name: {{ $proxy.Name | quote }}, type: {{ $proxy.Type }}, server: {{ $server }}, port: {{ $proxy.Port }}, {{ $common }} }$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'yaml'
  AND strpos("subscribe_template", 'clientSupportsProxy "stash"') > 0
  AND strpos("subscribe_template", $crypto_old$  - { name: {{ $proxy.Name | quote }}, type: ss, server: {{ $server }}, port: {{ $proxy.Port }}, cipher: {{ default "aes-128-gcm" $proxy.Method }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.Obfs) "" }}, plugin: obfs, plugin-opts: { mode: {{ $proxy.Obfs }}, host: {{ default "" $proxy.ObfsHost }} }{{- end }} }
  {{- else if eq $proxy.Type "vmess" }}
  - { name: {{ $proxy.Name | quote }}, type: vmess, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, alterId: 0, cipher: auto, {{ $common }}{{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls: true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else if eq $proxy.Type "vless" }}
  - { name: {{ $proxy.Name | quote }}, type: vless, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, {{ $common }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, reality-opts: { public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow: {{ $proxy.Flow }}{{- end }} }
  {{- else if eq $proxy.Type "trojan" }}
  - { name: {{ $proxy.Name | quote }}, type: trojan, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }} }
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, auth: {{ $password | quote }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  {{- else if eq $proxy.Type "tuic" }}
  - { name: {{ $proxy.Name | quote }}, type: tuic, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ default "" $proxy.ServerKey }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni: true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt: true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode: {{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller: {{ $proxy.CongestionController }}{{- end }} }
  {{- else if eq $proxy.Type "wireguard" }}
  - { name: {{ $proxy.Name | quote }}, type: wireguard, server: {{ $server }}, port: {{ $proxy.Port }}, private-key: {{ default "" $proxy.ServerKey }}, public-key: {{ default "" $proxy.RealityPublicKey }}, {{ $common }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key: {{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip: {{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6: {{ $proxy.RealityServerPort }}{{- end }} }
  {{- else if eq $proxy.Type "anytls" }}
  - { name: {{ $proxy.Name | quote }}, type: anytls, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else }}
  - { name: {{ $proxy.Name | quote }}, type: {{ $proxy.Type }}, server: {{ $server }}, port: {{ $proxy.Port }}, {{ $common }} }$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old${{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

#!MANAGED-CONFIG {{ .UserInfo.SubscribeURL }} interval=86400 strict=true

[General]
# 日志级别
loglevel = notify

# 外部控制器访问
external-controller-access = perlnk@0.0.0.0:6170

# 网络设置
exclude-simple-hostnames = true
show-error-page-for-reject = true
udp-priority = true
udp-policy-not-supported-behaviour = reject
ipv6 = true
ipv6-vif = auto

# 连接测试
proxy-test-url = http://www.gstatic.com/generate_204
internet-test-url = http://www.gstatic.com/generate_204
test-timeout = 5

# DNS 设置
dns-server = system, 119.29.29.29, 223.5.5.5
encrypted-dns-server = https://dns.alidns.com/dns-query
hijack-dns = 8.8.8.8:53, 8.8.4.4:53, 1.1.1.1:53, 1.0.0.1:53

# 跳过代理
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local

# 真实 IP
always-real-ip = *.lan, lens.l.google.com, *.srv.nintendo.net, *.stun.playstation.net, *.xboxlive.com, xbox.*.*.microsoft.com, *.msftncsi.com, *.msftconnecttest.com

# Surge Mac 参数
http-listen = 0.0.0.0:6088
socks5-listen = 0.0.0.0:6089

# Surge iOS 参数（WiFi 共享）
allow-wifi-access = true
allow-hotspot-access = true
wifi-access-http-port = 6088
wifi-access-socks5-port = 6089

[Panel]
SubscribeInfo = title={{ .SiteName }} - {{ .SubscribeName }}, content=已用流量: {{ $used }} GiB/{{ $total }} GiB \n到期时间: {{ $ExpiredAt}}, style=info

[Proxy]
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp-relay=true, tfo=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
{{ $proxy.Name }} = ss, {{ $server }}, {{ $proxy.Port }}, encrypt-method={{ default "aes-128-gcm" $proxy.Method }}, password={{ $password }}{{- if ne (default "" $proxy.Obfs) "" }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls=true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = tuic, {{ $server }}, {{ $proxy.Port }}, uuid={{ default "" $proxy.ServerKey }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni=true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt=true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode={{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller={{ $proxy.CongestionController }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = wireguard, {{ $server }}, {{ $proxy.Port }}, private-key={{ default "" $proxy.ServerKey }}, public-key={{ default "" $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip={{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6={{ $proxy.RealityServerPort }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}, {{ $common }}
  {{- else }}
{{ $proxy.Name }} = {{ $proxy.Type }}, {{ $server }}, {{ $proxy.Port }}, {{ $common }}$crypto_old$,
$crypto_new${{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

#!MANAGED-CONFIG {{ .UserInfo.SubscribeURL }} interval=86400 strict=true

[General]
# 日志级别
loglevel = notify

# 外部控制器访问
external-controller-access = perlnk@0.0.0.0:6170

# 网络设置
exclude-simple-hostnames = true
show-error-page-for-reject = true
udp-priority = true
udp-policy-not-supported-behaviour = reject
ipv6 = true
ipv6-vif = auto

# 连接测试
proxy-test-url = http://www.gstatic.com/generate_204
internet-test-url = http://www.gstatic.com/generate_204
test-timeout = 5

# DNS 设置
dns-server = system, 119.29.29.29, 223.5.5.5
encrypted-dns-server = https://dns.alidns.com/dns-query
hijack-dns = 8.8.8.8:53, 8.8.4.4:53, 1.1.1.1:53, 1.0.0.1:53

# 跳过代理
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local

# 真实 IP
always-real-ip = *.lan, lens.l.google.com, *.srv.nintendo.net, *.stun.playstation.net, *.xboxlive.com, xbox.*.*.microsoft.com, *.msftncsi.com, *.msftconnecttest.com

# Surge Mac 参数
http-listen = 0.0.0.0:6088
socks5-listen = 0.0.0.0:6089

# Surge iOS 参数（WiFi 共享）
allow-wifi-access = true
allow-hotspot-access = true
wifi-access-http-port = 6088
wifi-access-socks5-port = 6089

[Panel]
SubscribeInfo = title={{ .SiteName }} - {{ .SubscribeName }}, content=已用流量: {{ $used }} GiB/{{ $total }} GiB \n到期时间: {{ $ExpiredAt}}, style=info

[Proxy]
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp-relay=true, tfo=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
{{ $proxy.Name }} = ss, {{ $server }}, {{ $proxy.Port }}, encrypt-method={{ default "aes-128-gcm" $proxy.Method }}, password={{ $password }}{{- if ne (default "" $proxy.Obfs) "" }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls=true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = tuic, {{ $server }}, {{ $proxy.Port }}, uuid={{ default "" $proxy.ServerKey }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni=true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt=true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode={{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller={{ $proxy.CongestionController }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = wireguard, {{ $server }}, {{ $proxy.Port }}, private-key={{ default "" $proxy.ServerKey }}, public-key={{ default "" $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip={{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6={{ $proxy.RealityServerPort }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}, {{ $common }}
  {{- else }}
{{ $proxy.Name }} = {{ $proxy.Type }}, {{ $server }}, {{ $proxy.Port }}, {{ $common }}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", 'clientSupportsProxy "surge"') > 0
  AND strpos("subscribe_template", $crypto_old${{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

#!MANAGED-CONFIG {{ .UserInfo.SubscribeURL }} interval=86400 strict=true

[General]
# 日志级别
loglevel = notify

# 外部控制器访问
external-controller-access = perlnk@0.0.0.0:6170

# 网络设置
exclude-simple-hostnames = true
show-error-page-for-reject = true
udp-priority = true
udp-policy-not-supported-behaviour = reject
ipv6 = true
ipv6-vif = auto

# 连接测试
proxy-test-url = http://www.gstatic.com/generate_204
internet-test-url = http://www.gstatic.com/generate_204
test-timeout = 5

# DNS 设置
dns-server = system, 119.29.29.29, 223.5.5.5
encrypted-dns-server = https://dns.alidns.com/dns-query
hijack-dns = 8.8.8.8:53, 8.8.4.4:53, 1.1.1.1:53, 1.0.0.1:53

# 跳过代理
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local

# 真实 IP
always-real-ip = *.lan, lens.l.google.com, *.srv.nintendo.net, *.stun.playstation.net, *.xboxlive.com, xbox.*.*.microsoft.com, *.msftncsi.com, *.msftconnecttest.com

# Surge Mac 参数
http-listen = 0.0.0.0:6088
socks5-listen = 0.0.0.0:6089

# Surge iOS 参数（WiFi 共享）
allow-wifi-access = true
allow-hotspot-access = true
wifi-access-http-port = 6088
wifi-access-socks5-port = 6089

[Panel]
SubscribeInfo = title={{ .SiteName }} - {{ .SubscribeName }}, content=已用流量: {{ $used }} GiB/{{ $total }} GiB \n到期时间: {{ $ExpiredAt}}, style=info

[Proxy]
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp-relay=true, tfo=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
{{ $proxy.Name }} = ss, {{ $server }}, {{ $proxy.Port }}, encrypt-method={{ default "aes-128-gcm" $proxy.Method }}, password={{ $password }}{{- if ne (default "" $proxy.Obfs) "" }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls=true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = tuic, {{ $server }}, {{ $proxy.Port }}, uuid={{ default "" $proxy.ServerKey }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni=true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt=true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode={{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller={{ $proxy.CongestionController }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = wireguard, {{ $server }}, {{ $proxy.Port }}, private-key={{ default "" $proxy.ServerKey }}, public-key={{ default "" $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip={{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6={{ $proxy.RealityServerPort }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}, {{ $common }}
  {{- else }}
{{ $proxy.Name }} = {{ $proxy.Type }}, {{ $server }}, {{ $proxy.Port }}, {{ $common }}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old${{ $proxy.Name }} = Shadowsocks,{{ $server }},{{ $proxy.Port }},{{ $proxy.Method }},{{ $password }}{{- if $proxy.Transport }},obfs={{ $proxy.Transport }},obfs-host={{ $proxy.SNI }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if or $proxy.SNI $proxy.Fingerprint $proxy.AllowInsecure }},over-tls=true{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = VLESS,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if and $proxy.Flow (ne $proxy.Flow "none") }},flow={{ $proxy.Flow }}{{ end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.RealityPublicKey }},reality-public-key={{ $proxy.RealityPublicKey }}{{- if $proxy.RealityShortId }},reality-short-id={{ $proxy.RealityShortId }}{{ end }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
{{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = TUIC,{{ $proxy.Server }},{{ $proxy.Port }},{{ $proxy.ServerKey }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.DisableSNI }},disable-sni={{ $proxy.DisableSNI }}{{ end }}{{- if $proxy.ReduceRtt }},reduce-rtt={{ $proxy.ReduceRtt }}{{ end }}{{- if $proxy.UDPRelayMode }},udp-relay-mode={{ $proxy.UDPRelayMode }}{{ end }}{{- if $proxy.CongestionController }},congestion-control={{ $proxy.CongestionController }}{{ end }}
{{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = WireGuard,interface-ip={{ $proxy.RealityServerAddr | default "10.0.0.2" }},interface-ipv6={{ $proxy.RealityServerPort | default "::" }},private-key={{ $proxy.ServerKey }},mtu=1280,dns=8.8.8.8,keepalive=25,peers=[{public-key={{ $proxy.RealityPublicKey }},allowed-ips="0.0.0.0/0,::/0",endpoint={{ $proxy.Host }}:{{ $proxy.Port }}{{- if $proxy.Path }},preshared-key={{ $proxy.Path }}{{ end }}}]
{{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}
{{- end }}
{{- end }}$crypto_old$,
$crypto_new${{ $proxy.Name }} = Shadowsocks,{{ $server }},{{ $proxy.Port }},{{ $proxy.Method }},{{ $password }}{{- if $proxy.Transport }},obfs={{ $proxy.Transport }},obfs-host={{ $proxy.SNI }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if or $proxy.SNI $proxy.Fingerprint $proxy.AllowInsecure }},over-tls=true{{ end }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-profile={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = VLESS,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if and $proxy.Flow (ne $proxy.Flow "none") }},flow={{ $proxy.Flow }}{{ end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.RealityPublicKey }},over-tls=true,tls-profile=chrome147,public-key={{ $proxy.RealityPublicKey }}{{- if $proxy.RealityShortId }},short-id={{ $proxy.RealityShortId }}{{ end }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-profile={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
{{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = TUIC,{{ $proxy.Server }},{{ $proxy.Port }},{{ $proxy.ServerKey }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.DisableSNI }},disable-sni={{ $proxy.DisableSNI }}{{ end }}{{- if $proxy.ReduceRtt }},reduce-rtt={{ $proxy.ReduceRtt }}{{ end }}{{- if $proxy.UDPRelayMode }},udp-relay-mode={{ $proxy.UDPRelayMode }}{{ end }}{{- if $proxy.CongestionController }},congestion-control={{ $proxy.CongestionController }}{{ end }}
{{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = WireGuard,interface-ip={{ $proxy.RealityServerAddr | default "10.0.0.2" }},interface-ipv6={{ $proxy.RealityServerPort | default "::" }},private-key={{ $proxy.ServerKey }},mtu=1280,dns=8.8.8.8,keepalive=25,peers=[{public-key={{ $proxy.RealityPublicKey }},allowed-ips="0.0.0.0/0,::/0",endpoint={{ $proxy.Host }}:{{ $proxy.Port }}{{- if $proxy.Path }},preshared-key={{ $proxy.Path }}{{ end }}}]
{{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}
{{- end }}
{{- end }}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", 'clientSupportsProxy "loon"') > 0
  AND strpos("subscribe_template", $crypto_old${{ $proxy.Name }} = Shadowsocks,{{ $server }},{{ $proxy.Port }},{{ $proxy.Method }},{{ $password }}{{- if $proxy.Transport }},obfs={{ $proxy.Transport }},obfs-host={{ $proxy.SNI }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if or $proxy.SNI $proxy.Fingerprint $proxy.AllowInsecure }},over-tls=true{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = VLESS,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if and $proxy.Flow (ne $proxy.Flow "none") }},flow={{ $proxy.Flow }}{{ end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.RealityPublicKey }},reality-public-key={{ $proxy.RealityPublicKey }}{{- if $proxy.RealityShortId }},reality-short-id={{ $proxy.RealityShortId }}{{ end }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
{{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = TUIC,{{ $proxy.Server }},{{ $proxy.Port }},{{ $proxy.ServerKey }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.DisableSNI }},disable-sni={{ $proxy.DisableSNI }}{{ end }}{{- if $proxy.ReduceRtt }},reduce-rtt={{ $proxy.ReduceRtt }}{{ end }}{{- if $proxy.UDPRelayMode }},udp-relay-mode={{ $proxy.UDPRelayMode }}{{ end }}{{- if $proxy.CongestionController }},congestion-control={{ $proxy.CongestionController }}{{ end }}
{{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = WireGuard,interface-ip={{ $proxy.RealityServerAddr | default "10.0.0.2" }},interface-ipv6={{ $proxy.RealityServerPort | default "::" }},private-key={{ $proxy.ServerKey }},mtu=1280,dns=8.8.8.8,keepalive=25,peers=[{public-key={{ $proxy.RealityPublicKey }},allowed-ips="0.0.0.0/0,::/0",endpoint={{ $proxy.Host }}:{{ $proxy.Port }}{{- if $proxy.Path }},preshared-key={{ $proxy.Path }}{{ end }}}]
{{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}
{{- end }}
{{- end }}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.12" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_old$,
$crypto_new${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- $certificateTrust := ne $proxy.Security "reality" -}}
{{- if and $certificateTrust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "TLS certificate pin requires a matching inline public PEM certificate for legacy sing-box" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $certificateTrust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if eq $proxy.Security "reality" -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"chrome\"}" $tlsOpts -}}
  {{- else if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", 'define "AllNodeNames"') > 0
  AND strpos("subscribe_template", $crypto_old${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.12" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.11" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_old$,
$crypto_new${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- $certificateTrust := ne $proxy.Security "reality" -}}
{{- if and $certificateTrust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "TLS certificate pin requires a matching inline public PEM certificate for legacy sing-box" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $certificateTrust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if eq $proxy.Security "reality" -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"chrome\"}" $tlsOpts -}}
  {{- else if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", 'define "AllNodeNames"') > 0
  AND strpos("subscribe_template", $crypto_old${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.11" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$
      {{- $SkipVerify := $proxy.AllowInsecure -}}

{{- if eq $proxy.Type "shadowsocks" -}}
    { "type": "shadowsocks", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "method": {{ $proxy.Method | quote }}, "password": {{ $password | quote }}, {{ $common }} }
{{- else if eq $proxy.Type "vmess" -}}
    { "type": "vmess", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "security": "auto", {{ $common }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "vless" -}}
    { "type": "vless", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, {{ $common }}{{ if and $proxy.Flow (ne $proxy.Flow "none") }}, "flow": {{ $proxy.Flow | quote }}{{ end }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.RealityPublicKey $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}{{ if $proxy.RealityPublicKey }}, "reality": {"enabled": true, "public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.RealityShortId }}, "short_id": {{ $proxy.RealityShortId | quote }}{{ end }}}{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "trojan" -}}
    { "type": "trojan", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    {{- if and $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
      {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for Hiddify" -}}
    {{- end -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}} }
{{- else if eq $proxy.Type "tuic" -}}
    { "type": "tuic", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "password": {{ $password | quote }}{{ if $proxy.DisableSNI }}, "disable_sni": {{ $proxy.DisableSNI }}{{ end }}{{ if $proxy.ReduceRtt }}, "reduce_rtt": {{ $proxy.ReduceRtt }}{{ end }}{{ if $proxy.UDPRelayMode }}, "udp_relay_mode": {{ $proxy.UDPRelayMode | quote }}{{ end }}{{ if $proxy.CongestionController }}, "congestion_control": {{ $proxy.CongestionController | quote }}{{ end }}, {{ $common }}, "alpn": ["h3"], "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if eq $proxy.Type "wireguard" -}}
    { "type": "wireguard", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "private_key": {{ $proxy.ServerKey | quote }}, "peer_public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.Path }}, "pre_shared_key": {{ $proxy.Path | quote }}{{ end }}{{ if $proxy.RealityServerAddr }}, "local_address": [{{ $proxy.RealityServerAddr | quote }}]{{ end }}, {{ $common }} }
{{- else if eq $proxy.Type "anytls" -}}
    { "type": "anytls", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else -}}
    { "type": "direct", "tag": {{ $proxy.Name | quote }}, {{ $common }} }$crypto_old$,
$crypto_new$
      {{- $SkipVerify := $proxy.AllowInsecure -}}
      {{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
        {{- fail "TLS certificate pin requires a matching inline public PEM certificate for Hiddify" -}}
      {{- end -}}

{{- if eq $proxy.Type "shadowsocks" -}}
    { "type": "shadowsocks", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "method": {{ $proxy.Method | quote }}, "password": {{ $password | quote }}, {{ $common }} }
{{- else if eq $proxy.Type "vmess" -}}
    { "type": "vmess", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "security": "auto", {{ $common }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or (eq $proxy.Security "tls") $proxy.TLSCertificate $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if and (ne $proxy.Security "reality") $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}{{ if eq $proxy.Security "reality" }}, "utls": {"enabled": true, "fingerprint": "chrome" }{{ else if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "vless" -}}
    { "type": "vless", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, {{ $common }}{{ if and $proxy.Flow (ne $proxy.Flow "none") }}, "flow": {{ $proxy.Flow | quote }}{{ end }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.RealityPublicKey $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if and (ne $proxy.Security "reality") $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}{{ if eq $proxy.Security "reality" }}, "utls": {"enabled": true, "fingerprint": "chrome" }{{ else if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}{{ if $proxy.RealityPublicKey }}, "reality": {"enabled": true, "public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.RealityShortId }}, "short_id": {{ $proxy.RealityShortId | quote }}{{ end }}}{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "trojan" -}}
    { "type": "trojan", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if and (ne $proxy.Security "reality") $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}{{ if eq $proxy.Security "reality" }}, "utls": {"enabled": true, "fingerprint": "chrome" }{{ else if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    {{- if and $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
      {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for Hiddify" -}}
    {{- end -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}} }
{{- else if eq $proxy.Type "tuic" -}}
    { "type": "tuic", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "password": {{ $password | quote }}{{ if $proxy.DisableSNI }}, "disable_sni": {{ $proxy.DisableSNI }}{{ end }}{{ if $proxy.ReduceRtt }}, "reduce_rtt": {{ $proxy.ReduceRtt }}{{ end }}{{ if $proxy.UDPRelayMode }}, "udp_relay_mode": {{ $proxy.UDPRelayMode | quote }}{{ end }}{{ if $proxy.CongestionController }}, "congestion_control": {{ $proxy.CongestionController | quote }}{{ end }}, {{ $common }}, "alpn": ["h3"], "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if and (ne $proxy.Security "reality") $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}{{ if eq $proxy.Security "reality" }}, "utls": {"enabled": true, "fingerprint": "chrome" }{{ else if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if eq $proxy.Type "wireguard" -}}
    { "type": "wireguard", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "private_key": {{ $proxy.ServerKey | quote }}, "peer_public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.Path }}, "pre_shared_key": {{ $proxy.Path | quote }}{{ end }}{{ if $proxy.RealityServerAddr }}, "local_address": [{{ $proxy.RealityServerAddr | quote }}]{{ end }}, {{ $common }} }
{{- else if eq $proxy.Type "anytls" -}}
    { "type": "anytls", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if and (ne $proxy.Security "reality") $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}{{ if eq $proxy.Security "reality" }}, "utls": {"enabled": true, "fingerprint": "chrome" }{{ else if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else -}}
    { "type": "direct", "tag": {{ $proxy.Name | quote }}, {{ $common }} }$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", 'clientSupportsProxy "hiddify"') > 0
  AND strpos("subscribe_template", $crypto_old$
      {{- $SkipVerify := $proxy.AllowInsecure -}}

{{- if eq $proxy.Type "shadowsocks" -}}
    { "type": "shadowsocks", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "method": {{ $proxy.Method | quote }}, "password": {{ $password | quote }}, {{ $common }} }
{{- else if eq $proxy.Type "vmess" -}}
    { "type": "vmess", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "security": "auto", {{ $common }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "vless" -}}
    { "type": "vless", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, {{ $common }}{{ if and $proxy.Flow (ne $proxy.Flow "none") }}, "flow": {{ $proxy.Flow | quote }}{{ end }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.RealityPublicKey $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}{{ if $proxy.RealityPublicKey }}, "reality": {"enabled": true, "public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.RealityShortId }}, "short_id": {{ $proxy.RealityShortId | quote }}{{ end }}}{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "trojan" -}}
    { "type": "trojan", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    {{- if and $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
      {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for Hiddify" -}}
    {{- end -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}} }
{{- else if eq $proxy.Type "tuic" -}}
    { "type": "tuic", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "password": {{ $password | quote }}{{ if $proxy.DisableSNI }}, "disable_sni": {{ $proxy.DisableSNI }}{{ end }}{{ if $proxy.ReduceRtt }}, "reduce_rtt": {{ $proxy.ReduceRtt }}{{ end }}{{ if $proxy.UDPRelayMode }}, "udp_relay_mode": {{ $proxy.UDPRelayMode | quote }}{{ end }}{{ if $proxy.CongestionController }}, "congestion_control": {{ $proxy.CongestionController | quote }}{{ end }}, {{ $common }}, "alpn": ["h3"], "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if eq $proxy.Type "wireguard" -}}
    { "type": "wireguard", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "private_key": {{ $proxy.ServerKey | quote }}, "peer_public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.Path }}, "pre_shared_key": {{ $proxy.Path | quote }}{{ end }}{{ if $proxy.RealityServerAddr }}, "local_address": [{{ $proxy.RealityServerAddr | quote }}]{{ end }}, {{ $common }} }
{{- else if eq $proxy.Type "anytls" -}}
    { "type": "anytls", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else -}}
    { "type": "direct", "tag": {{ $proxy.Name | quote }}, {{ $common }} }$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$    "security": "reality",
    "realitySettings": {
      "serverName": {{ $sni | quote }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}{{ if $proxy.RealityPublicKey }}, "password": {{ $proxy.RealityPublicKey | quote }}{{ end }}{{ if $proxy.RealityShortId }}, "shortId": {{ $proxy.RealityShortId | quote }}{{ end }}{{ if $proxy.Path }}, "spiderX": {{ $proxy.Path | quote }}{{ end }}
    }
    {{- end }}$crypto_old$,
$crypto_new$    "security": "reality",
    "realitySettings": {
      "serverName": {{ $sni | quote }}, "fingerprint": "chrome"{{ if $proxy.RealityPublicKey }}, "password": {{ $proxy.RealityPublicKey | quote }}{{ end }}{{ if $proxy.RealityShortId }}, "shortId": {{ $proxy.RealityShortId | quote }}{{ end }}{{ if $proxy.Path }}, "spiderX": {{ $proxy.Path | quote }}{{ end }}
    }
    {{- end }}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", 'define "streamSettings"') > 0
  AND strpos("subscribe_template", $crypto_old$    "security": "reality",
    "realitySettings": {
      "serverName": {{ $sni | quote }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}{{ if $proxy.RealityPublicKey }}, "password": {{ $proxy.RealityPublicKey | quote }}{{ end }}{{ if $proxy.RealityShortId }}, "shortId": {{ $proxy.RealityShortId | quote }}{{ end }}{{ if $proxy.Path }}, "spiderX": {{ $proxy.Path | quote }}{{ end }}
    }
    {{- end }}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old${{- end -}}
{{- $tlsOpts := "" -}}
{{- if $proxy.TLS -}}
  {{- $tlsOpts = ", tls=true" -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
  {{- end -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
  {{- end -}}
{{- end -}}
{{ $name }} = vmess, {{ $server }}, {{ $port }}, username={{ $pwd }}, udp-relay=true, ws=true, ws-path={{ $wsPath }}{{ $wsHeaders }}{{ $tlsOpts }}, vmess-aead=true
{{- else if eq $proxy.Type "trojan" -}}
{{- $wsOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $wsOpts = printf ", ws=true, ws-path=%s" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $wsOpts = printf "%s, ws-headers=Host:%s" $wsOpts $proxy.Host -}}
  {{- end -}}
{{- end -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}
{{- if $sni -}}
  {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
{{- end -}}
{{ $name }} = trojan, {{ $server }}, {{ $port }}, password={{ $pwd }}, udp-relay=true{{ $wsOpts }}{{ $tlsOpts }}
{{- else if eq $proxy.Type "http" -}}
{{ $name }} = http, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}
{{- else if eq $proxy.Type "https" -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}$crypto_old$,
$crypto_new${{- end -}}
{{- $tlsOpts := "" -}}
{{- if or $proxy.TLS (eq $proxy.Security "tls") -}}
  {{- $tlsOpts = ", tls=true" -}}
  {{- if $proxy.PinnedPeerCertSha256 -}}
  {{- $tlsOpts = printf "%s, server-cert-fingerprint-sha256=%s" $tlsOpts (first (splitList "," $proxy.PinnedPeerCertSha256)) -}}
{{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
  {{- end -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
  {{- end -}}
{{- end -}}
{{ $name }} = vmess, {{ $server }}, {{ $port }}, username={{ $pwd }}, udp-relay=true, ws=true, ws-path={{ $wsPath }}{{ $wsHeaders }}{{ $tlsOpts }}, vmess-aead=true
{{- else if eq $proxy.Type "trojan" -}}
{{- $wsOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $wsOpts = printf ", ws=true, ws-path=%s" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $wsOpts = printf "%s, ws-headers=Host:%s" $wsOpts $proxy.Host -}}
  {{- end -}}
{{- end -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.PinnedPeerCertSha256 -}}
  {{- $tlsOpts = printf "%s, server-cert-fingerprint-sha256=%s" $tlsOpts (first (splitList "," $proxy.PinnedPeerCertSha256)) -}}
{{- else if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}
{{- if $sni -}}
  {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
{{- end -}}
{{ $name }} = trojan, {{ $server }}, {{ $port }}, password={{ $pwd }}, udp-relay=true{{ $wsOpts }}{{ $tlsOpts }}
{{- else if eq $proxy.Type "http" -}}
{{ $name }} = http, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}
{{- else if eq $proxy.Type "https" -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.PinnedPeerCertSha256 -}}
  {{- $tlsOpts = printf "%s, server-cert-fingerprint-sha256=%s" $tlsOpts (first (splitList "," $proxy.PinnedPeerCertSha256)) -}}
{{- else if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", 'define "SurfboardProxy"') > 0
  AND strpos("subscribe_template", $crypto_old${{- end -}}
{{- $tlsOpts := "" -}}
{{- if $proxy.TLS -}}
  {{- $tlsOpts = ", tls=true" -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
  {{- end -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
  {{- end -}}
{{- end -}}
{{ $name }} = vmess, {{ $server }}, {{ $port }}, username={{ $pwd }}, udp-relay=true, ws=true, ws-path={{ $wsPath }}{{ $wsHeaders }}{{ $tlsOpts }}, vmess-aead=true
{{- else if eq $proxy.Type "trojan" -}}
{{- $wsOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $wsOpts = printf ", ws=true, ws-path=%s" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $wsOpts = printf "%s, ws-headers=Host:%s" $wsOpts $proxy.Host -}}
  {{- end -}}
{{- end -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}
{{- if $sni -}}
  {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
{{- end -}}
{{ $name }} = trojan, {{ $server }}, {{ $port }}, password={{ $pwd }}, udp-relay=true{{ $wsOpts }}{{ $tlsOpts }}
{{- else if eq $proxy.Type "http" -}}
{{ $name }} = http, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}
{{- else if eq $proxy.Type "https" -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}$crypto_old$) > 0;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$crypto_old$    {{- /* obfs enum from constants.ts SECURITY.shadowsocks: 'none','http','tls' */ -}}
    {{- $ssObfs := default "none" $proxy.Obfs }}
shadowsocks={{ $server }}:{{ $proxy.Port }}, method={{ $method }}, password={{ $password }}{{- if ne $ssObfs "none" }}, obfs={{ $ssObfs }}{{- end }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vmess ========== */ -}}
  {{- else if eq $proxyType "vmess" }}
    {{- $vmMethod := default "none" $proxy.Method }}
    {{- /* Transport enum from constants.ts TRANSPORTS.vmess: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum from constants.ts SECURITY.vmess: 'none','tls' */ -}}
    {{- $vmTransport := default "tcp" $proxy.Transport }}
    {{- $vmSecurity := default "none" $proxy.Security }}
    {{- $vmObfs := "" -}}
    {{- if eq $vmTransport "websocket" -}}
      {{- if eq $vmSecurity "tls" -}}
        {{- $vmObfs = "wss" -}}
      {{- else -}}
        {{- $vmObfs = "ws" -}}
      {{- end -}}
    {{- else if eq $vmSecurity "tls" -}}
      {{- $vmObfs = "over-tls" -}}
    {{- end }}
vmess={{ $server }}:{{ $proxy.Port }}, method={{ $vmMethod }}, password={{ $password }}{{- if ne $vmObfs "" }}, obfs={{ $vmObfs }}{{- end }}{{- if and (or (eq $vmObfs "ws") (eq $vmObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vmObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vless ========== */ -}}
  {{- else if eq $proxyType "vless" }}
    {{- /* Transport enum: 'tcp','websocket','grpc','mkcp','httpupgrade','xhttp' */ -}}
    {{- /* Security enum: 'none','tls','reality' */ -}}
    {{- $vlTransport := default "tcp" $proxy.Transport }}
    {{- $vlSecurity := default "none" $proxy.Security }}
    {{- $vlObfs := "" -}}
    {{- if eq $vlTransport "websocket" -}}
      {{- if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
        {{- $vlObfs = "wss" -}}
      {{- else -}}
        {{- $vlObfs = "ws" -}}
      {{- end -}}
    {{- else if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
      {{- $vlObfs = "over-tls" -}}
    {{- end }}
vless={{ $server }}:{{ $proxy.Port }}, method=none, password={{ $password }}{{- if ne $vlObfs "" }}, obfs={{ $vlObfs }}{{- end }}{{- if and (or (eq $vlObfs "ws") (eq $vlObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vlObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, vless-flow={{ $proxy.Flow }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== trojan ========== */ -}}
  {{- else if eq $proxyType "trojan" }}
    {{- /* Transport enum: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum: 'tls' (always tls for trojan) */ -}}
    {{- $trTransport := default "tcp" $proxy.Transport }}
    {{- if eq $trTransport "websocket" }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, obfs=wss{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- else }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- end }}
  {{- /* ========== anytls ========== */ -}}
  {{- else if eq $proxyType "anytls" }}
    {{- /* Security enum: 'tls' (always tls for anytls) */ -}}
anytls={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== http (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "http" }}
    {{- /* Security enum: 'none','tls' */ -}}
    {{- $httpSecurity := default "none" $proxy.Security }}
http={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $httpSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== socks (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "socks" }}
    {{- /* socks has no security/transport in schema; TLS detected via Security field */ -}}
    {{- $socksSecurity := default "none" $proxy.Security }}
socks5={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $socksSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{ end }}
{{ end }}$crypto_old$,
$crypto_new$    {{- /* obfs enum from constants.ts SECURITY.shadowsocks: 'none','http','tls' */ -}}
    {{- $ssObfs := default "none" $proxy.Obfs }}
shadowsocks={{ $server }}:{{ $proxy.Port }}, method={{ $method }}, password={{ $password }}{{- if ne $ssObfs "none" }}, obfs={{ $ssObfs }}{{- end }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vmess ========== */ -}}
  {{- else if eq $proxyType "vmess" }}
    {{- $vmMethod := default "none" $proxy.Method }}
    {{- /* Transport enum from constants.ts TRANSPORTS.vmess: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum from constants.ts SECURITY.vmess: 'none','tls' */ -}}
    {{- $vmTransport := default "tcp" $proxy.Transport }}
    {{- $vmSecurity := default "none" $proxy.Security }}
    {{- $vmObfs := "" -}}
    {{- if eq $vmTransport "websocket" -}}
      {{- if eq $vmSecurity "tls" -}}
        {{- $vmObfs = "wss" -}}
      {{- else -}}
        {{- $vmObfs = "ws" -}}
      {{- end -}}
    {{- else if eq $vmSecurity "tls" -}}
      {{- $vmObfs = "over-tls" -}}
    {{- end }}
vmess={{ $server }}:{{ $proxy.Port }}, method={{ $vmMethod }}, password={{ $password }}{{- if ne $vmObfs "" }}, obfs={{ $vmObfs }}{{- end }}{{- if and (or (eq $vmObfs "ws") (eq $vmObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else if ne $vmObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vless ========== */ -}}
  {{- else if eq $proxyType "vless" }}
    {{- /* Transport enum: 'tcp','websocket','grpc','mkcp','httpupgrade','xhttp' */ -}}
    {{- /* Security enum: 'none','tls','reality' */ -}}
    {{- $vlTransport := default "tcp" $proxy.Transport }}
    {{- $vlSecurity := default "none" $proxy.Security }}
    {{- $vlObfs := "" -}}
    {{- if eq $vlTransport "websocket" -}}
      {{- if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
        {{- $vlObfs = "wss" -}}
      {{- else -}}
        {{- $vlObfs = "ws" -}}
      {{- end -}}
    {{- else if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
      {{- $vlObfs = "over-tls" -}}
    {{- end }}
vless={{ $server }}:{{ $proxy.Port }}, method=none, password={{ $password }}{{- if ne $vlObfs "" }}, obfs={{ $vlObfs }}{{- end }}{{- if and (or (eq $vlObfs "ws") (eq $vlObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else if ne $vlObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, vless-flow={{ $proxy.Flow }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== trojan ========== */ -}}
  {{- else if eq $proxyType "trojan" }}
    {{- /* Transport enum: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum: 'tls' (always tls for trojan) */ -}}
    {{- $trTransport := default "tcp" $proxy.Transport }}
    {{- if eq $trTransport "websocket" }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, obfs=wss{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- else }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- end }}
  {{- /* ========== anytls ========== */ -}}
  {{- else if eq $proxyType "anytls" }}
    {{- /* Security enum: 'tls' (always tls for anytls) */ -}}
anytls={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== http (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "http" }}
    {{- /* Security enum: 'none','tls' */ -}}
    {{- $httpSecurity := default "none" $proxy.Security }}
http={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $httpSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== socks (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "socks" }}
    {{- /* socks has no security/transport in schema; TLS detected via Security field */ -}}
    {{- $socksSecurity := default "none" $proxy.Security }}
socks5={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $socksSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if and (not $hasReality) $proxy.PinnedPeerCertSha256 }}, tls-verification=true, tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{ end }}
{{ end }}$crypto_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", 'clientSupportsProxy "quantumult-x"') > 0
  AND strpos("subscribe_template", $crypto_old$    {{- /* obfs enum from constants.ts SECURITY.shadowsocks: 'none','http','tls' */ -}}
    {{- $ssObfs := default "none" $proxy.Obfs }}
shadowsocks={{ $server }}:{{ $proxy.Port }}, method={{ $method }}, password={{ $password }}{{- if ne $ssObfs "none" }}, obfs={{ $ssObfs }}{{- end }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vmess ========== */ -}}
  {{- else if eq $proxyType "vmess" }}
    {{- $vmMethod := default "none" $proxy.Method }}
    {{- /* Transport enum from constants.ts TRANSPORTS.vmess: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum from constants.ts SECURITY.vmess: 'none','tls' */ -}}
    {{- $vmTransport := default "tcp" $proxy.Transport }}
    {{- $vmSecurity := default "none" $proxy.Security }}
    {{- $vmObfs := "" -}}
    {{- if eq $vmTransport "websocket" -}}
      {{- if eq $vmSecurity "tls" -}}
        {{- $vmObfs = "wss" -}}
      {{- else -}}
        {{- $vmObfs = "ws" -}}
      {{- end -}}
    {{- else if eq $vmSecurity "tls" -}}
      {{- $vmObfs = "over-tls" -}}
    {{- end }}
vmess={{ $server }}:{{ $proxy.Port }}, method={{ $vmMethod }}, password={{ $password }}{{- if ne $vmObfs "" }}, obfs={{ $vmObfs }}{{- end }}{{- if and (or (eq $vmObfs "ws") (eq $vmObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vmObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vless ========== */ -}}
  {{- else if eq $proxyType "vless" }}
    {{- /* Transport enum: 'tcp','websocket','grpc','mkcp','httpupgrade','xhttp' */ -}}
    {{- /* Security enum: 'none','tls','reality' */ -}}
    {{- $vlTransport := default "tcp" $proxy.Transport }}
    {{- $vlSecurity := default "none" $proxy.Security }}
    {{- $vlObfs := "" -}}
    {{- if eq $vlTransport "websocket" -}}
      {{- if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
        {{- $vlObfs = "wss" -}}
      {{- else -}}
        {{- $vlObfs = "ws" -}}
      {{- end -}}
    {{- else if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
      {{- $vlObfs = "over-tls" -}}
    {{- end }}
vless={{ $server }}:{{ $proxy.Port }}, method=none, password={{ $password }}{{- if ne $vlObfs "" }}, obfs={{ $vlObfs }}{{- end }}{{- if and (or (eq $vlObfs "ws") (eq $vlObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vlObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, vless-flow={{ $proxy.Flow }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== trojan ========== */ -}}
  {{- else if eq $proxyType "trojan" }}
    {{- /* Transport enum: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum: 'tls' (always tls for trojan) */ -}}
    {{- $trTransport := default "tcp" $proxy.Transport }}
    {{- if eq $trTransport "websocket" }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, obfs=wss{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- else }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- end }}
  {{- /* ========== anytls ========== */ -}}
  {{- else if eq $proxyType "anytls" }}
    {{- /* Security enum: 'tls' (always tls for anytls) */ -}}
anytls={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== http (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "http" }}
    {{- /* Security enum: 'none','tls' */ -}}
    {{- $httpSecurity := default "none" $proxy.Security }}
http={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $httpSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== socks (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "socks" }}
    {{- /* socks has no security/transport in schema; TLS detected via Security field */ -}}
    {{- $socksSecurity := default "none" $proxy.Security }}
socks5={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $socksSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{ end }}
{{ end }}$crypto_old$) > 0;

COMMIT;
