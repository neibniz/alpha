import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { FormDescription } from "@workspace/ui/components/form";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@workspace/ui/components/hover-card";
import { cn } from "@workspace/ui/lib/utils";
import { AlertTriangle, CircleHelp, ExternalLink } from "lucide-react";
import type { ReactNode } from "react";

type XrayFieldHelpInfo = {
  title: string;
  hint: string;
  detail?: string;
  example?: string;
  warning?: string;
  source: string;
  sourceLabel: string;
};

const SOURCES = {
  config: {
    label: "Xray 配置文件",
    url: "https://xtls.github.io/config/",
  },
  vless: {
    label: "VLESS 入站",
    url: "https://xtls.github.io/config/inbounds/vless.html",
  },
  reality: {
    label: "REALITY",
    url: "https://xtls.github.io/config/transports/reality.html",
  },
  tls: {
    label: "TLS",
    url: "https://xtls.github.io/config/transports/tls.html",
  },
  grpc: {
    label: "gRPC 传输",
    url: "https://xtls.github.io/config/transports/grpc.html",
  },
  xhttp: {
    label: "XHTTP",
    url: "https://xtls.github.io/config/transports/xhttp.html",
  },
  dns: {
    label: "内置 DNS",
    url: "https://xtls.github.io/config/dns.html",
  },
  routing: {
    label: "路由",
    url: "https://xtls.github.io/config/routing.html",
  },
  hysteria: {
    label: "Hysteria 入站",
    url: "https://xtls.github.io/config/inbounds/hysteria.html",
  },
  freedom: {
    label: "Freedom 出站",
    url: "https://xtls.github.io/config/outbounds/freedom.html",
  },
} as const;

function help(
  source: keyof typeof SOURCES,
  data: Omit<XrayFieldHelpInfo, "source" | "sourceLabel">
) {
  return {
    ...data,
    source: SOURCES[source].url,
    sourceLabel: SOURCES[source].label,
  };
}

export const XRAY_FIELD_HELP: Record<string, XrayFieldHelpInfo> = {
  protocol: help("config", {
    title: "Protocol",
    hint: "入站和出站的协议类型，决定 settings 的结构。",
    detail:
      "切换协议后，只显示该协议相关字段。高级字段仍可放入 Settings JSON。",
  }),
  tag: help("config", {
    title: "Tag",
    hint: "用于路由、统计和模板引用的唯一标识。",
    detail:
      "路由规则中的 inboundTag/outboundTag 会引用这个值，同类型模板内建议保持唯一。",
    example: "vless-reality-grpc",
  }),
  listen: help("config", {
    title: "Listen",
    hint: "入站监听地址。服务端常用 0.0.0.0。",
    example: "0.0.0.0",
  }),
  port: help("config", {
    title: "Port",
    hint: "入站监听端口，范围 1-65535。",
    warning: "同一台服务器上的多个 inbound 不能监听同一端口。",
    example: "443",
  }),
  settings_json: help("config", {
    title: "Settings JSON",
    hint: "协议 settings 的高级覆盖字段。",
    detail:
      "表单已支持的字段会自动生成，只有官方文档确认但表单暂未覆盖的字段才建议放这里。",
  }),
  network: help("config", {
    title: "Network",
    hint: "streamSettings.network 选择传输方式，例如 raw、grpc、xhttp。",
    warning: "不同 network 对应不同 settings 对象，切换后请检查下方传输字段。",
  }),
  security: help("config", {
    title: "Security",
    hint: "streamSettings.security 选择传输安全层，例如 none、tls、reality。",
    warning: "REALITY 只适用于官方文档支持的传输组合。",
  }),
  host: help("config", {
    title: "Host",
    hint: "传输层 Host/headers 字段，常用于 WebSocket、XHTTP 或反代场景。",
    detail: "没有反代、CDN 或 Host 覆写需求时可以留空。",
  }),
  path: help("config", {
    title: "Path",
    hint: "传输层路径字段，常用于 WebSocket、XHTTP 或 HTTPUpgrade。",
    example: "/ray",
  }),
  raw_settings_json: help("config", {
    title: "Raw Settings",
    hint: "RAW/TCP 传输高级对象。",
    detail: "表单没有覆盖的官方字段可以放在这里。",
  }),
  kcp_settings_json: help("config", {
    title: "KCP Settings",
    hint: "mKCP 传输高级对象。",
    warning: "mKCP 参数较多，建议只配置明确需要的字段。",
  }),
  hysteria_settings_json: help("hysteria", {
    title: "Hysteria Settings",
    hint: "streamSettings.hysteriaSettings 高级对象。",
    detail: "Hysteria 入站/出站协议字段与 Hysteria 传输字段需要区分。",
  }),
  sockopt_json: help("config", {
    title: "Sockopt",
    hint: "底层 socket 行为的高级配置对象。",
    warning:
      "涉及 tproxy、mark、dialerProxy 等系统行为，确认运行环境支持后再启用。",
  }),
  finalmask_json: help("config", {
    title: "FinalMask",
    hint: "传输层 FinalMask 高级对象。",
    detail: "没有明确需求时建议保持空对象。",
  }),
  variables_schema_json: help("config", {
    title: "Variables Schema",
    hint: "定义服务器绑定时可填写的变量结构。",
    detail: "只描述变量，不直接进入 Xray 配置。绑定页会据此生成可编辑字段。",
  }),
  default_variables_json: help("config", {
    title: "Default Variables",
    hint: "仅用于 config_template 中无法从 config 推断的默认变量。",
    warning: "不要与模板配置中的同名变量写成不同值，否则保存时会被阻止。",
  }),
  vless_decryption: help("vless", {
    title: "VLESS Decryption",
    hint: "VLESS decryption 不能留空；禁用加密时也要显式填写 none。",
    example: "none",
  }),
  flow: help("vless", {
    title: "Flow",
    hint: "VLESS 流控参数，常见值为 xtls-rprx-vision。",
    detail:
      "只在 VLESS + TCP/RAW + TLS/REALITY 这类直连传输中填写。XHTTP、gRPC、WebSocket、HTTPUpgrade 等传输不要填写 flow。",
    warning:
      "如果 XHTTP 节点填写 xtls-rprx-vision，客户端和服务端即使配置一致也可能无法建立连接。",
    example: "xtls-rprx-vision",
  }),
  out_flow: help("vless", {
    title: "Flow",
    hint: "VLESS 出站流控参数，常见值为 xtls-rprx-vision。",
    detail:
      "只在 VLESS + TCP/RAW + TLS/REALITY 这类直连传输中填写。XHTTP、gRPC、WebSocket、HTTPUpgrade 等传输不要填写 flow。",
    warning:
      "传输方式不是 TCP/RAW 时通常应留空，避免客户端生成不可连接的配置。",
    example: "xtls-rprx-vision",
  }),
  vless_clients_json: help("vless", {
    title: "VLESS Users",
    hint: "服务端认可的 VLESS 用户数组。",
    detail:
      "xray-agent 会把真实订阅用户注入 settings.clients；模板里保留默认用户用于提供 flow 等默认值。flow 仅适用于 VLESS TCP/RAW + TLS/REALITY，XHTTP/gRPC/WS 场景请留空。",
  }),
  vmess_clients_json: help("config", {
    title: "VMess Users",
    hint: "服务端认可的 VMess 用户数组。",
    detail: "xray-agent 会把真实订阅用户注入 settings.clients。",
  }),
  trojan_clients_json: help("config", {
    title: "Trojan Users",
    hint: "服务端认可的 Trojan 用户数组。",
    detail: "xray-agent 会把真实订阅用户注入 settings.clients。",
  }),
  ss_clients_json: help("config", {
    title: "Shadowsocks Users",
    hint: "多用户 Shadowsocks 用户数组。",
    detail:
      "单用户模板可只填写 method/password；多用户时 users 会由 agent 注入。",
  }),
  fallbacks_json: help("vless", {
    title: "Fallbacks",
    hint: "按需配置 TLS/REALITY 回落目标。",
    warning: "没有明确回落需求时建议留空，减少配置复杂度。",
  }),
  allow_insecure: help("reality", {
    title: "Allow Insecure",
    hint: "TLS 客户端是否允许不安全证书。",
    warning:
      "Xray-core 将在 2026-06-01 后移除该字段；请改用 pinnedPeerCertSha256 或 verifyPeerCertByName。",
  }),
  pinned_peer_cert_sha256: help("tls", {
    title: "Pinned Peer Cert SHA-256",
    hint: "TLS 出站用于固定服务端证书的 SHA-256 指纹。",
    detail:
      "用于替代 allowInsecure。可通过 xray tls hash --cert cert.pem 或 openssl x509 -noout -fingerprint -sha256 -in cert.pem 获取。",
    warning: "仅在需要连接自签名或私有 CA 证书时填写。",
    example: "AA:BB:CC:DD:...",
  }),
  verify_peer_cert_by_name: help("tls", {
    title: "Verify Peer Cert By Name",
    hint: "TLS 出站用于指定证书 SAN 校验名称。",
    detail:
      "多个名称可用英文逗号分隔。普通公网证书通常只需要填写 SNI，不需要额外配置。",
    example: "example.com",
  }),
  reality_target: help("reality", {
    title: "REALITY Target",
    hint: "服务端必填，目标站点地址与端口。",
    warning: "这是服务端字段；缺失时容易被误当成客户端 REALITY 配置。",
    example: "www.ebay.com:443",
  }),
  reality_server_names: help("reality", {
    title: "REALITY Server Names",
    hint: "服务端允许的 SNI 列表。",
    detail: "客户端 serverName 必须匹配列表中的值。多个值用英文逗号分隔。",
    example: "www.ebay.com, ebay.com",
  }),
  reality_private_key: help("reality", {
    title: "REALITY Private Key",
    hint: "服务端私钥，使用 xray x25519 生成。",
    warning: "不要填写客户端 public key。",
  }),
  reality_mldsa65_seed: help("reality", {
    title: "ML-DSA-65 Seed",
    hint: "服务端用于添加抗量子证书签名的私有种子。",
    detail: "使用 xray mldsa65 生成；仅在明确启用 ML-DSA-65 时填写。",
    warning: "这是服务端私密材料，不要放入客户端配置或日志。",
  }),
  reality_mldsa65_verify: help("reality", {
    title: "ML-DSA-65 Verify",
    hint: "客户端用于验证服务端 ML-DSA-65 签名的公钥。",
    detail: "必须与服务端 mldsa65Seed 生成的公钥配对。",
  }),
  reality_public_key: help("reality", {
    title: "REALITY Public Key",
    hint: "客户端连接服务端时使用的 public key。",
    warning: "出站配置使用 public key；入站配置使用 private key。",
  }),
  reality_short_ids: help("reality", {
    title: "REALITY Short IDs",
    hint: "服务端可接受的 shortId 列表。",
    detail: "非空值应为十六进制，位数必须是偶数，最多 16 位。",
    example: "0123456789abcdef",
  }),
  reality_short_id: help("reality", {
    title: "REALITY Short ID",
    hint: "客户端 shortId，需要匹配服务端 shortIds 中的一项。",
    detail: "非空值应为十六进制，位数必须是偶数，最多 16 位。",
  }),
  fingerprint: help("reality", {
    title: "Fingerprint",
    hint: "REALITY/TLS 客户端指纹。",
    warning: "REALITY 不支持 unsafe 禁用 utls。",
    example: "chrome",
  }),
  sni: help("reality", {
    title: "SNI / Server Name",
    hint: "TLS 或 REALITY 客户端握手使用的 serverName。",
    detail: "REALITY 出站需与服务端 serverNames 匹配。",
  }),
  service_name: help("grpc", {
    title: "gRPC Service Name",
    hint: "gRPC serviceName，用于区分 gRPC 路径。",
    example: "grpc",
  }),
  grpc_authority: help("grpc", {
    title: "gRPC Authority",
    hint: "gRPC authority，常用于反代或 CDN 场景。",
    detail: "没有明确需求可以留空。",
  }),
  grpc_multi_mode: help("grpc", {
    title: "gRPC Multi Mode",
    hint: "实验性选项，默认 false。",
    warning: "官方文档标记为 BETA，不保证长期兼容。服务端一般不需要启用。",
  }),
  grpc_user_agent: help("grpc", {
    title: "gRPC User-Agent",
    hint: "出站客户端发送给上游的 User-Agent。",
    detail: "只对出站生效，可用于避免部分 CDN 拒绝 gRPC 请求。",
  }),
  grpc_idle_timeout: help("grpc", {
    title: "gRPC Idle Timeout",
    hint: "空闲多久后进行健康检查，单位秒。",
    warning: "小于 10 会按 10 处理；部分反代场景下过低可能触发连接关闭。",
  }),
  grpc_health_check_timeout: help("grpc", {
    title: "gRPC Health Check Timeout",
    hint: "健康检查超时时间，单位秒。",
    detail: "没有开启健康检查需求可以留空。",
  }),
  grpc_permit_without_stream: help("grpc", {
    title: "Permit Without Stream",
    hint: "没有活动子连接时仍允许发送健康检查。",
    detail: "只对出站生效，通常与 Idle Timeout 一起配置。",
  }),
  grpc_initial_windows_size: help("grpc", {
    title: "Initial Window Size",
    hint: "HTTP/2 Stream 的初始窗口大小。",
    detail: "0 表示关闭；大于 65535 时会禁用动态窗口机制。只对出站生效。",
  }),
  xhttp_mode: help("xhttp", {
    title: "XHTTP Mode",
    hint: "XHTTP 传输模式。",
    detail: "常用 auto 或 stream-one；需要严格限制连接数时配合 extra.xmux。",
  }),
  xhttp_extra_json: help("xhttp", {
    title: "XHTTP Extra",
    hint: "XHTTP 高级 extra 对象。",
    detail: "例如通过 xmux.maxConnections 控制复用连接行为。",
    example: '{"xmux":{"maxConnections":1}}',
  }),
  hysteria_version: help("hysteria", {
    title: "Hysteria Version",
    hint: "官方要求必须为 2。",
    warning: "不是 2 的值会导致配置无效。",
  }),
  hysteria_auth: help("hysteria", {
    title: "Hysteria Auth",
    hint: "Hysteria 传输层认证字符串。",
    detail:
      "作为出站客户端密码使用；入站存在 settings.clients 时会被用户列表覆盖。",
  }),
  udp_idle_timeout: help("hysteria", {
    title: "UDP Idle Timeout",
    hint: "QUIC UDP 连接空闲等待时间，单位秒。",
    detail: "官方默认 60，通常不需要调整。",
  }),
  hysteria_masquerade_json: help("hysteria", {
    title: "Masquerade",
    hint: "HTTP/3 页面伪装对象。",
    detail: "支持 string、file、proxy 三种 type；留空则使用默认 404 页面。",
  }),
  hysteria_users_json: help("hysteria", {
    title: "Hysteria Users",
    hint: "服务端认可的 Hysteria 用户数组。",
    detail: "users 只在搭配 hysteria 传输层时生效；auth 是认证字符串。",
  }),
  sniffing: help("config", {
    title: "Sniffing",
    hint: "入站流量探测，可用于识别 HTTP/TLS/QUIC 等目标信息。",
    detail: "常与路由配合使用，开启后可基于探测结果做分流。",
  }),
  dest_override: help("config", {
    title: "Dest Override",
    hint: "Sniffing 可覆盖目标的协议列表。",
    example: "http, tls, quic",
  }),
  sniffing_metadata_only: help("config", {
    title: "Metadata Only",
    hint: "只使用连接元数据进行探测。",
  }),
  sniffing_route_only: help("config", {
    title: "Route Only",
    hint: "探测结果仅用于路由，不改写连接目标。",
  }),
  sniffing_domains_excluded: help("config", {
    title: "Domains Excluded",
    hint: "Sniffing 排除的域名列表。",
    example: "geosite:cn, domain:example.com",
  }),
  routing_domain_strategy: help("routing", {
    title: "Routing Domain Strategy",
    hint: "域名解析策略：AsIs、IPIfNonMatch、IPOnDemand。",
    detail:
      "AsIs 不额外解析；IPIfNonMatch 未命中后解析；IPOnDemand 匹配前解析。",
  }),
  routing_rules_json: help("routing", {
    title: "Routing Rules",
    hint: "RuleObject 数组，用于把流量分到不同 outbound。",
    warning: "outboundTag 必须引用已绑定的出站模板 tag。",
  }),
  routing_balancers_json: help("routing", {
    title: "Routing Balancers",
    hint: "BalancerObject 数组，用于多个出站之间做选择。",
    detail: "路由规则使用 balancerTag 时，需要在这里定义对应 balancer。",
  }),
  servers: help("dns", {
    title: "DNS Servers",
    hint: "一行一个 DNS 服务器，支持 IP、localhost、tcp://、https:// 等形式。",
    warning: "localhost 不受 Xray 控制，若需要转发系统 DNS 需额外配置。",
  }),
  dns_servers_json: help("dns", {
    title: "DNS Server Objects",
    hint: "DNS Server Object 数组，可配置 domains、expectedIPs、skipFallback 等条件。",
    detail: "适合为不同域名使用不同 DNS 服务器。",
  }),
  query_strategy: help("dns", {
    title: "DNS Query Strategy",
    hint: "UseIP 查询 A+AAAA，UseIPv4 只查 A，UseIPv6 只查 AAAA，UseSystem 跟随系统环境。",
    warning:
      "全局 queryStrategy 与子项 queryStrategy 冲突时，子项可能返回空响应。",
  }),
  hosts_json: help("dns", {
    title: "Hosts",
    hint: "静态 hosts 映射，值可以是 IP、域名或数组。",
    detail: "匹配格式同路由 domain，未写前缀时默认类似 full:。",
  }),
  client_ip: help("dns", {
    title: "Client IP",
    hint: "EDNS Client Subnet 使用的 IP 地址。",
    detail: "必须是有效 IPv4 或 IPv6，发送时会自动抹掉最后几位。",
  }),
  disable_cache: help("dns", {
    title: "Disable Cache",
    hint: "禁用 DNS 缓存，默认 false。",
    warning: "对 localhost DNS 不生效。",
  }),
  serve_stale: help("dns", {
    title: "Serve Stale",
    hint: "启用 DNS 乐观缓存，默认 false。",
    warning: "只有在缓存未被禁用时才有效。",
  }),
  serve_expired_ttl: help("dns", {
    title: "Serve Expired TTL",
    hint: "乐观缓存有效期，单位秒；0 表示不过期。",
  }),
  disable_fallback: help("dns", {
    title: "Disable Fallback",
    hint: "禁用 DNS fallback 查询，默认 false。",
  }),
  disable_fallback_if_match: help("dns", {
    title: "Disable Fallback If Match",
    hint: "当优先匹配域名列表命中时禁用 fallback 查询，默认 false。",
  }),
  enable_parallel_query: help("dns", {
    title: "Enable Parallel Query",
    hint: "启用并行 DNS 查询，默认 false。",
  }),
  domain_strategy: help("freedom", {
    title: "Freedom Domain Strategy",
    hint: "Freedom 出站发起连接前的域名解析策略。",
    example: "AsIs / UseIP / UseIPv4 / UseIPv6",
  }),
  redirect: help("freedom", {
    title: "Freedom Redirect",
    hint: "Freedom 出站重定向目标地址。",
    detail: "常用于透明代理或特定转发场景，没有需求时留空。",
  }),
  fragment_json: help("freedom", {
    title: "Freedom Fragment",
    hint: "Freedom fragment 高级对象。",
    detail: "仅在明确需要分片行为时配置。",
  }),
  noises_json: help("freedom", {
    title: "Freedom Noises",
    hint: "Freedom noises 高级列表。",
    detail: "仅在明确需要额外噪声行为时配置。",
  }),
  response_type: help("config", {
    title: "Blackhole Response",
    hint: "Blackhole 出站响应类型。",
    detail: "通常使用 none；需要模拟 HTTP 响应时才配置 http。",
  }),
};

export function getXrayFieldHelp(fieldKey?: string) {
  return fieldKey ? XRAY_FIELD_HELP[fieldKey] : undefined;
}

export function XrayFieldHelp({
  className,
  fieldKey,
}: {
  className?: string;
  fieldKey?: string;
}) {
  const info = getXrayFieldHelp(fieldKey);
  if (!info) return null;

  return (
    <HoverCard closeDelay={120} openDelay={120}>
      <HoverCardTrigger asChild>
        <Button
          aria-label={`${info.title} help`}
          className={cn(className)}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <CircleHelp />
        </Button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="flex w-80 flex-col gap-3">
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between gap-2">
            <div className="font-medium text-sm">{info.title}</div>
            <Badge className="shrink-0" variant="outline">
              Xray
            </Badge>
          </div>
          <p className="text-muted-foreground text-xs leading-5">{info.hint}</p>
          {info.detail ? (
            <p className="text-muted-foreground text-xs leading-5">
              {info.detail}
            </p>
          ) : null}
        </div>
        {info.example ? (
          <div className="rounded-md bg-muted px-2 py-1.5 font-mono text-xs">
            {info.example}
          </div>
        ) : null}
        {info.warning ? (
          <Alert>
            <AlertTriangle />
            <AlertDescription>{info.warning}</AlertDescription>
          </Alert>
        ) : null}
        <Button asChild className="self-start px-0" size="sm" variant="link">
          <a href={info.source} rel="noreferrer" target="_blank">
            {info.sourceLabel}
            <ExternalLink data-icon="inline-end" />
          </a>
        </Button>
      </HoverCardContent>
    </HoverCard>
  );
}

export function XrayFieldLabel({
  children,
}: {
  children: ReactNode;
  fieldKey?: string;
}) {
  return <span className="truncate">{children}</span>;
}

export function XrayFieldDescription({
  description,
  fieldKey,
}: {
  description?: ReactNode;
  fieldKey?: string;
}) {
  const info = getXrayFieldHelp(fieldKey);
  if (!(description || info?.hint || info?.warning)) return null;
  const primaryDescription = description || info?.hint;

  return (
    <FormDescription className="space-y-1 text-xs">
      {primaryDescription ? (
        <span className="block">{primaryDescription}</span>
      ) : null}
      {info?.warning ? (
        <span className="block text-amber-600">{info.warning}</span>
      ) : null}
    </FormDescription>
  );
}
