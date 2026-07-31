"use client";

import { Link } from "@tanstack/react-router";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Progress } from "@workspace/ui/components/progress";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@workspace/ui/components/chart";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@workspace/ui/components/table";
import { Tabs, TabsList, TabsTrigger } from "@workspace/ui/components/tabs";
import { getAuthorizationToken } from "@workspace/ui/lib/auth-token";
import { getApiBaseURL, getApiPrefix } from "@workspace/ui/lib/runtime-config";
import { cn } from "@workspace/ui/lib/utils";
import { formatBytes } from "@workspace/ui/utils/formatting";
import {
  Activity,
  CheckCircle2,
  CircleDollarSign,
  Cpu,
  ExternalLink,
  HardDrive,
  MemoryStick,
  Server,
  ShieldAlert,
  ShieldCheck,
  ShoppingBag,
  TicketCheck,
  UserPlus,
  Users,
  Wifi,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { Display } from "@/components/display";
import {
  aggregateNetworkRateSeries,
  appendRealtimeTrafficPoint,
  appendRealtimeValuePoint,
  mergeRealtimeNetworkPoints,
} from "./statistics-data";

type NetworkRange = "1h" | "6h" | "24h";
type NetworkResolution = "5s" | "1m";
type NetworkSource = "all" | "system" | "xray";
type RankingPeriod = "today" | "month";

type DashboardRealtimePatch = Partial<
  Omit<
    API.DashboardRealtimeResponse,
    | "business"
    | "connections"
    | "network"
    | "network_series"
    | "online_user_series"
    | "resources"
    | "servers"
    | "traffic"
  >
> & {
  business?: Partial<API.DashboardRealtimeResponse["business"]>;
  connections?: Partial<API.DashboardRealtimeResponse["connections"]>;
  full?: boolean;
  network?: Partial<API.DashboardRealtimeResponse["network"]>;
  network_series?: API.DashboardRealtimeResponse["network_series"];
  online_user_series?: Partial<
    API.DashboardRealtimeResponse["online_user_series"]
  >;
  resources?: Partial<API.DashboardRealtimeResponse["resources"]>;
  servers?: Partial<API.DashboardRealtimeResponse["servers"]>;
  traffic?: Partial<API.DashboardRealtimeResponse["traffic"]>;
};

type NetworkChartDatum = {
  systemRxBps: number;
  systemTxBps: number;
  timestamp: number;
  xrayRxBps: number;
  xrayTxBps: number;
};

type OperationalIssue = {
  key: string;
  level: string;
  message?: string;
  serverId?: number;
  title: string;
};

const NETWORK_RANGE_MS: Record<NetworkRange, number> = {
  "1h": 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
};
const NETWORK_SERIES_MS = NETWORK_RANGE_MS["24h"];

const RATE_COLORS = {
  systemRxBps: "#4F83FF",
  systemTxBps: "#3DD6B4",
  xrayRxBps: "#F1A33B",
  xrayTxBps: "#3DBBD8",
};

const ALERT_TITLE_KEYS: Record<string, string> = {
  "Config apply failed": "alerts.titles.configApplyFailed",
  "High resource usage": "alerts.titles.highResourceUsage",
  "Server offline": "alerts.titles.serverOffline",
  "Xray stats error": "alerts.titles.xrayStatsError",
  "Xray stopped": "alerts.titles.xrayStopped",
};

function buildDashboardRealtimeWsUrl() {
  const base = getApiBaseURL() || window.location.origin;
  const prefix = getApiPrefix();
  const url = new URL(`${prefix}/v1/admin/console/realtime/ws`, base);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

function formatBitrate(value?: number) {
  const bitrate = Math.max(0, Number(value) || 0);
  if (bitrate === 0) return "0 bps";

  const units = ["bps", "Kbps", "Mbps", "Gbps", "Tbps"];
  let scaled = bitrate;
  let unitIndex = 0;
  while (scaled >= 1000 && unitIndex < units.length - 1) {
    scaled /= 1000;
    unitIndex += 1;
  }
  const digits = unitIndex === 0 ? 0 : scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
  return `${scaled.toFixed(digits)} ${units[unitIndex]}`;
}

function formatPercent(value?: number) {
  return `${Math.max(0, Number(value) || 0).toFixed(1)}%`;
}

function hasPatchField(patch: DashboardRealtimePatch, key: string) {
  return Object.hasOwn(patch, key);
}

function networkPointFromNetwork(
  timestamp: number,
  network: API.DashboardRealtimeResponse["network"]
): API.DashboardRealtimeResponse["network_series"][number] {
  return {
    timestamp,
    system_rx_bps: network.system_rx_bps || 0,
    system_tx_bps: network.system_tx_bps || 0,
    xray_rx_bps: network.xray_rx_bps || 0,
    xray_tx_bps: network.xray_tx_bps || 0,
  };
}

function mergeDashboardRealtime(
  previous: API.DashboardRealtimeResponse,
  patch: DashboardRealtimePatch
): API.DashboardRealtimeResponse {
  const updatedAt = patch.updated_at || Date.now();
  const dayStart = new Date(updatedAt);
  dayStart.setHours(0, 0, 0, 0);
  const nextOnlineUsers = patch.online_users ?? previous.online_users;
  const nextTraffic = { ...previous.traffic, ...(patch.traffic || {}) };
  const nextNetwork = { ...previous.network, ...(patch.network || {}) };
  const nextNetworkSeries =
    patch.network_series && patch.network_series.length > 0
      ? patch.network_series
      : [networkPointFromNetwork(updatedAt, nextNetwork)];

  return {
    ...previous,
    ...patch,
    activities: hasPatchField(patch, "activities")
      ? patch.activities || []
      : previous.activities,
    alerts: hasPatchField(patch, "alerts")
      ? patch.alerts || []
      : previous.alerts,
    business: { ...previous.business, ...(patch.business || {}) },
    connections: { ...previous.connections, ...(patch.connections || {}) },
    network: nextNetwork,
    network_series: mergeRealtimeNetworkPoints(
      previous.network_series,
      nextNetworkSeries,
      updatedAt - NETWORK_SERIES_MS
    ),
    online_user_series: {
      last_7_days: appendRealtimeValuePoint(
        patch.online_user_series?.last_7_days ??
          previous.online_user_series?.last_7_days,
        { timestamp: updatedAt, value: nextOnlineUsers },
        updatedAt - 7 * 24 * 60 * 60 * 1000
      ),
      last_day: appendRealtimeValuePoint(
        patch.online_user_series?.last_day ??
          previous.online_user_series?.last_day,
        { timestamp: updatedAt, value: nextOnlineUsers },
        updatedAt - 24 * 60 * 60 * 1000
      ),
    },
    online_users: nextOnlineUsers,
    resources: { ...previous.resources, ...(patch.resources || {}) },
    servers: { ...previous.servers, ...(patch.servers || {}) },
    traffic: {
      ...nextTraffic,
      today_series: appendRealtimeTrafficPoint(
        patch.traffic?.today_series ?? previous.traffic?.today_series,
        {
          download: nextTraffic.today_download,
          timestamp: updatedAt,
          total: nextTraffic.today_upload + nextTraffic.today_download,
          upload: nextTraffic.today_upload,
        },
        dayStart.getTime()
      ),
    },
    updated_at: updatedAt,
  };
}

function ensureNetworkRateSeries(
  points: API.DashboardRealtimeResponse["network_series"] | undefined,
  network: API.DashboardRealtimeResponse["network"] | undefined,
  updatedAt: number | undefined,
  spanMs: number
): NetworkChartDatum[] {
  const now = updatedAt || Date.now();
  const cutoff = now - spanMs;
  const data =
    points
      ?.filter((item) => item.timestamp >= cutoff)
      .map((item) => ({
        systemRxBps: item.system_rx_bps,
        systemTxBps: item.system_tx_bps,
        timestamp: item.timestamp,
        xrayRxBps: item.xray_rx_bps,
        xrayTxBps: item.xray_tx_bps,
      })) || [];
  if (data.length > 1) return data;
  const current = data[0] || {
    systemRxBps: network?.system_rx_bps || 0,
    systemTxBps: network?.system_tx_bps || 0,
    timestamp: now,
    xrayRxBps: network?.xray_rx_bps || 0,
    xrayTxBps: network?.xray_tx_bps || 0,
  };
  return [
    {
      systemRxBps: 0,
      systemTxBps: 0,
      timestamp: Math.max(0, current.timestamp - spanMs),
      xrayRxBps: 0,
      xrayTxBps: 0,
    },
    current,
  ];
}

function issueCount(realtime?: API.DashboardRealtimeResponse) {
  const serverIssues =
    (realtime?.servers.offline || 0) +
    (realtime?.servers.config_failed || 0) +
    (realtime?.servers.xray_stopped || 0) +
    (realtime?.servers.stats_error || 0);
  return Math.max(realtime?.alerts.length || 0, serverIssues);
}

function StatusMetric({
  icon: Icon,
  label,
  secondary,
  value,
}: {
  icon: LucideIcon;
  label: string;
  secondary: string;
  value: string | number;
}) {
  return (
    <div className="dashboard-status-metric min-w-0 border-t p-4 lg:border-t-0 lg:border-l">
      <div className="flex items-center gap-2 text-muted-foreground text-xs">
        <Icon aria-hidden />
        <span className="truncate">{label}</span>
      </div>
      <div className="mt-2 truncate font-semibold text-xl tabular-nums">
        {value}
      </div>
      <div className="mt-1 truncate text-muted-foreground text-xs">
        {secondary}
      </div>
    </div>
  );
}

function HealthSummary({
  connected,
  realtime,
}: {
  connected: boolean;
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t } = useTranslation("dashboard");
  const issues = issueCount(realtime);
  const hasRisk = issues > 0;
  const totalServers = realtime?.servers.total || 0;
  const onlineServers = realtime?.servers.online || 0;
  const onlineRate =
    totalServers > 0 ? (onlineServers / totalServers) * 100 : 0;
  const network = realtime?.network;
  const totalRate =
    (network?.system_rx_bps || 0) +
    (network?.system_tx_bps || 0) +
    (network?.xray_rx_bps || 0) +
    (network?.xray_tx_bps || 0);
  const todayUpload = realtime?.traffic.today_upload || 0;
  const todayDownload = realtime?.traffic.today_download || 0;

  return (
    <Card className="dashboard-health-band gap-0 overflow-hidden py-0">
      <div className="dashboard-health-grid grid min-h-28 grid-cols-2 lg:grid-cols-[minmax(18rem,1.35fr)_repeat(4,minmax(0,1fr))]">
        <div className="col-span-2 flex min-w-0 flex-col justify-between gap-4 p-4 sm:flex-row sm:items-center lg:col-span-1">
          <div className="flex min-w-0 items-center gap-3">
            <div
              className={cn(
                "flex size-10 shrink-0 items-center justify-center rounded-md",
                hasRisk
                  ? "bg-amber-500/10 text-amber-600 dark:text-amber-300"
                  : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-300"
              )}
            >
              {hasRisk ? (
                <ShieldAlert aria-hidden />
              ) : (
                <ShieldCheck aria-hidden />
              )}
            </div>
            <div className="min-w-0">
              <div className="font-semibold text-base">
                {hasRisk
                  ? t("serviceNeedsAttention", "Service needs attention")
                  : t("serviceHealthy", "Service is healthy")}
              </div>
              <div className="mt-1 truncate text-muted-foreground text-xs">
                {hasRisk
                  ? t("issuesDetected", {
                      count: issues,
                      defaultValue: "Detected {{count}} issues",
                    })
                  : connected
                    ? t(
                        "allSystemsNominal",
                        "All monitored services are normal"
                      )
                    : t("reconnecting", "Reconnecting")}
              </div>
            </div>
          </div>
          <Button asChild className="shrink-0" size="sm">
            <Link to="/dashboard/servers">
              {hasRisk
                ? t("resolveIssues", "Resolve issues")
                : t("inspectServices", "Inspect services")}
            </Link>
          </Button>
        </div>
        <StatusMetric
          icon={Server}
          label={t("onlineNodes", "Online nodes")}
          secondary={t("onlineRate", {
            defaultValue: "Online rate {{value}}",
            value: formatPercent(onlineRate),
          })}
          value={`${onlineServers}/${totalServers}`}
        />
        <StatusMetric
          icon={Users}
          label={t("onlineNow", "Online now")}
          secondary={t("activeConnections", "Current active users")}
          value={realtime?.online_users || 0}
        />
        <StatusMetric
          icon={Activity}
          label={t("currentRate", "Current rate")}
          secondary={t("combinedThroughput", "Combined host and Xray")}
          value={formatBitrate(totalRate)}
        />
        <StatusMetric
          icon={Wifi}
          label={t("trafficToday", "Traffic today")}
          secondary={`${t("uploadShort", "Up")} ${formatBytes(todayUpload)} · ${t("downloadShort", "Down")} ${formatBytes(todayDownload)}`}
          value={formatBytes(todayUpload + todayDownload)}
        />
      </div>
    </Card>
  );
}

function RateLegendItem({
  color,
  label,
  value,
}: {
  color: string;
  label: string;
  value: string;
}) {
  return (
    <div className="min-w-0 border-t px-4 py-3 sm:border-t-0 sm:border-l sm:first:border-l-0">
      <div className="flex min-w-0 items-center gap-2 text-muted-foreground text-xs">
        <span
          className="size-2 shrink-0 rounded-full"
          style={{ backgroundColor: color }}
        />
        <span className="truncate">{label}</span>
      </div>
      <div className="mt-1 truncate font-semibold text-sm tabular-nums">
        {value}
      </div>
    </div>
  );
}

function NetworkChartCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t, i18n } = useTranslation("dashboard");
  const [range, setRange] = useState<NetworkRange>("1h");
  const [resolution, setResolution] = useState<NetworkResolution>("5s");
  const [source, setSource] = useState<NetworkSource>("all");
  const aggregatedSeries = useMemo(
    () =>
      resolution === "1m"
        ? aggregateNetworkRateSeries(realtime?.network_series)
        : realtime?.network_series,
    [realtime?.network_series, resolution]
  );
  const data = useMemo(
    () =>
      ensureNetworkRateSeries(
        aggregatedSeries,
        realtime?.network,
        realtime?.updated_at,
        NETWORK_RANGE_MS[range]
      ),
    [aggregatedSeries, range, realtime?.network, realtime?.updated_at]
  );
  const network = realtime?.network;
  const showSystem = source !== "xray";
  const showXray = source !== "system";

  const timeFormatter = (value: number) => {
    const date = new Date(Number(value));
    if (range === "1h" || range === "6h") {
      return date.toLocaleTimeString(i18n.language, {
        hour: "2-digit",
        minute: "2-digit",
      });
    }
    return date.toLocaleDateString(i18n.language, {
      day: "numeric",
      hour: "2-digit",
      month: "short",
    });
  };

  return (
    <Card className="dashboard-panel h-full overflow-hidden">
      <CardHeader className="gap-3 border-b py-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="min-w-0">
            <CardTitle className="text-base">
              {t("realtimeTraffic", "Realtime traffic")}
            </CardTitle>
            <p className="mt-1 text-muted-foreground text-xs">
              {t(
                "realtimeTrafficDescription",
                "Observe host and Xray throughput in one timeline"
              )}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Select
              onValueChange={(value) => setSource(value as NetworkSource)}
              value={source}
            >
              <SelectTrigger
                aria-label={t("trafficSource", "Traffic source")}
                className="h-8 w-32"
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="all">
                    {t("allTraffic", "All traffic")}
                  </SelectItem>
                  <SelectItem value="system">
                    {t("hostTraffic", "Host traffic")}
                  </SelectItem>
                  <SelectItem value="xray">
                    {t("xrayTraffic", "Xray traffic")}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Tabs
              onValueChange={(value) => setRange(value as NetworkRange)}
              value={range}
            >
              <TabsList>
                <TabsTrigger value="1h">1h</TabsTrigger>
                <TabsTrigger value="6h">6h</TabsTrigger>
                <TabsTrigger value="24h">24h</TabsTrigger>
              </TabsList>
            </Tabs>
            <Tabs
              onValueChange={(value) =>
                setResolution(value as NetworkResolution)
              }
              value={resolution}
            >
              <TabsList>
                <TabsTrigger value="5s">5s</TabsTrigger>
                <TabsTrigger value="1m">1m</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-0 pt-0">
        <div className="h-[18rem] px-3 py-4 sm:h-[20rem] sm:px-5">
          <ChartContainer
            className="h-full w-full"
            config={{
              systemRxBps: {
                color: RATE_COLORS.systemRxBps,
                label: t("rate.systemRxBps", "Host down"),
              },
              systemTxBps: {
                color: RATE_COLORS.systemTxBps,
                label: t("rate.systemTxBps", "Host up"),
              },
              xrayRxBps: {
                color: RATE_COLORS.xrayRxBps,
                label: t("rate.xrayRxBps", "Xray down"),
              },
              xrayTxBps: {
                color: RATE_COLORS.xrayTxBps,
                label: t("rate.xrayTxBps", "Xray up"),
              },
            }}
          >
            <LineChart
              data={data}
              margin={{ bottom: 0, left: 0, right: 14, top: 8 }}
            >
              <CartesianGrid
                stroke="var(--border)"
                strokeDasharray="4 4"
                vertical={false}
              />
              <XAxis
                axisLine={false}
                dataKey="timestamp"
                height={28}
                minTickGap={30}
                tickFormatter={(value) => timeFormatter(Number(value))}
                tickLine={false}
                tickMargin={10}
              />
              <YAxis
                axisLine={false}
                tickFormatter={(value) => formatBitrate(Number(value) || 0)}
                tickLine={false}
                width={66}
              />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    formatter={(value, name) => [
                      formatBitrate(Number(value) || 0),
                      t(`rate.${name}`, String(name)),
                    ]}
                    labelFormatter={(value) =>
                      new Date(Number(value)).toLocaleString(i18n.language)
                    }
                  />
                }
                cursor={{ stroke: "var(--border)" }}
              />
              {showSystem ? (
                <Line
                  dataKey="systemTxBps"
                  dot={false}
                  name="systemTxBps"
                  stroke={RATE_COLORS.systemTxBps}
                  strokeWidth={2}
                  type="monotone"
                />
              ) : null}
              {showSystem ? (
                <Line
                  dataKey="systemRxBps"
                  dot={false}
                  name="systemRxBps"
                  stroke={RATE_COLORS.systemRxBps}
                  strokeWidth={2}
                  type="monotone"
                />
              ) : null}
              {showXray ? (
                <Line
                  dataKey="xrayTxBps"
                  dot={false}
                  name="xrayTxBps"
                  stroke={RATE_COLORS.xrayTxBps}
                  strokeWidth={2}
                  type="monotone"
                />
              ) : null}
              {showXray ? (
                <Line
                  dataKey="xrayRxBps"
                  dot={false}
                  name="xrayRxBps"
                  stroke={RATE_COLORS.xrayRxBps}
                  strokeWidth={2}
                  type="monotone"
                />
              ) : null}
            </LineChart>
          </ChartContainer>
        </div>
        <div className="grid border-t sm:grid-cols-2 xl:grid-cols-4">
          <RateLegendItem
            color={RATE_COLORS.systemTxBps}
            label={t("rate.systemTxBps", "Host up")}
            value={formatBitrate(network?.system_tx_bps)}
          />
          <RateLegendItem
            color={RATE_COLORS.systemRxBps}
            label={t("rate.systemRxBps", "Host down")}
            value={formatBitrate(network?.system_rx_bps)}
          />
          <RateLegendItem
            color={RATE_COLORS.xrayTxBps}
            label={t("rate.xrayTxBps", "Xray up")}
            value={formatBitrate(network?.xray_tx_bps)}
          />
          <RateLegendItem
            color={RATE_COLORS.xrayRxBps}
            label={t("rate.xrayRxBps", "Xray down")}
            value={formatBitrate(network?.xray_rx_bps)}
          />
        </div>
      </CardContent>
    </Card>
  );
}

function buildOperationalIssues(
  realtime: API.DashboardRealtimeResponse | undefined,
  translate: (key: string, fallback: string) => string
): OperationalIssue[] {
  if (realtime?.alerts.length) {
    return realtime.alerts.slice(0, 4).map((alert, index) => ({
      key: `${alert.server_id || "global"}-${alert.title}-${index}`,
      level: alert.level,
      message: alert.message,
      serverId: alert.server_id,
      title: ALERT_TITLE_KEYS[alert.title]
        ? translate(ALERT_TITLE_KEYS[alert.title]!, alert.title)
        : alert.title,
    }));
  }

  const issues: OperationalIssue[] = [];
  if ((realtime?.servers.offline || 0) > 0) {
    issues.push({
      key: "offline",
      level: "critical",
      message: translate(
        "alerts.messages.inspectOffline",
        "Inspect offline servers"
      ),
      title: translate("alerts.titles.serverOffline", "Server offline"),
    });
  }
  if ((realtime?.servers.config_failed || 0) > 0) {
    issues.push({
      key: "config-failed",
      level: "warning",
      message: translate(
        "alerts.messages.retryConfig",
        "Retry failed configuration delivery"
      ),
      title: translate(
        "alerts.titles.configApplyFailed",
        "Configuration delivery failed"
      ),
    });
  }
  if ((realtime?.servers.xray_stopped || 0) > 0) {
    issues.push({
      key: "xray-stopped",
      level: "critical",
      message: translate(
        "alerts.messages.inspectXray",
        "Inspect the Xray process"
      ),
      title: translate("alerts.titles.xrayStopped", "Xray stopped"),
    });
  }
  return issues;
}

function AlertQueueCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t } = useTranslation("dashboard");
  const issues = buildOperationalIssues(realtime, (key, fallback) =>
    t(key, fallback)
  );

  return (
    <Card className="dashboard-panel min-h-[14.75rem] overflow-hidden">
      <CardHeader className="flex-row items-center justify-between border-b py-4">
        <div className="flex items-center gap-2">
          <CardTitle>{t("pendingItems", "Pending items")}</CardTitle>
          {issues.length > 0 ? (
            <span className="inline-flex min-w-5 items-center justify-center rounded-full bg-red-500/10 px-1.5 py-0.5 font-semibold text-red-600 text-xs dark:text-red-300">
              {issues.length}
            </span>
          ) : null}
        </div>
        <Button asChild className="h-7 gap-1 px-2" size="sm" variant="ghost">
          <Link to="/dashboard/servers">
            {t("viewAll", "View all")}
            <ExternalLink className="size-3.5" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="px-0">
        {issues.length > 0 ? (
          <div className="divide-y">
            {issues.map((issue) => {
              const critical = issue.level === "critical";
              return (
                <div
                  className="flex items-start gap-3 px-4 py-3"
                  key={issue.key}
                >
                  <span
                    className={cn(
                      "mt-1.5 size-2 shrink-0 rounded-full",
                      critical ? "bg-red-500" : "bg-amber-500"
                    )}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium text-sm">
                      {issue.title}
                    </div>
                    <div className="mt-1 line-clamp-2 text-muted-foreground text-xs">
                      {issue.message ||
                        t(
                          "detectedRealtime",
                          "Detected by realtime monitoring"
                        )}
                    </div>
                  </div>
                  <Button
                    asChild
                    className={cn(
                      "h-8 shrink-0 px-2.5",
                      critical &&
                        "border-red-500/40 text-red-600 dark:text-red-300"
                    )}
                    size="sm"
                    variant="outline"
                  >
                    <Link to="/dashboard/servers">
                      {critical
                        ? t("resolveNow", "Resolve now")
                        : t("inspect", "Inspect")}
                    </Link>
                  </Button>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="flex min-h-32 items-center gap-3 px-4 py-6">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-300">
              <CheckCircle2 className="size-5" />
            </div>
            <div>
              <div className="font-medium text-sm">
                {t("noPendingIssues", "No pending issues")}
              </div>
              <div className="mt-1 text-muted-foreground text-xs">
                {t("allSystemsNominal", "All monitored services are normal")}
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ResourceGauge({
  color,
  icon: Icon,
  label,
  maximum,
  value,
}: {
  color: string;
  icon: LucideIcon;
  label: string;
  maximum: number;
  value: number;
}) {
  const safeValue = Math.min(100, Math.max(0, value || 0));
  return (
    <div
      aria-label={`${label} ${formatPercent(safeValue)}`}
      aria-valuemax={100}
      aria-valuemin={0}
      aria-valuenow={safeValue}
      className="min-w-0 px-4 py-3"
      role="meter"
    >
      <div className="flex items-center justify-between gap-3 text-xs">
        <div className="flex min-w-0 items-center gap-2 text-muted-foreground">
          <Icon aria-hidden />
          <span className="truncate">{label}</span>
        </div>
        <strong className="font-semibold tabular-nums">
          {Math.round(safeValue)}%
        </strong>
      </div>
      <Progress
        aria-label={label}
        className="mt-2 h-1.5"
        style={{ "--progress-color": color } as React.CSSProperties}
        value={safeValue}
      />
      <div className="mt-1.5 truncate text-muted-foreground text-[11px]">
        {maximum > 0 ? `Max ${formatPercent(maximum)}` : "--"}
      </div>
    </div>
  );
}

function ResourceLevelsCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t } = useTranslation("dashboard");
  return (
    <Card className="dashboard-panel overflow-hidden">
      <CardHeader className="border-b py-4">
        <CardTitle>{t("resourceLevels", "Resource levels")}</CardTitle>
      </CardHeader>
      <CardContent className="px-0">
        <div className="grid divide-y">
          <ResourceGauge
            color="#3DD6B4"
            icon={Cpu}
            label={t("cpuUsage", "CPU")}
            maximum={realtime?.resources.max_cpu || 0}
            value={realtime?.resources.avg_cpu || 0}
          />
          <ResourceGauge
            color="#4F83FF"
            icon={MemoryStick}
            label={t("memoryUsage", "Memory")}
            maximum={realtime?.resources.max_mem || 0}
            value={realtime?.resources.avg_mem || 0}
          />
          <ResourceGauge
            color="#3DBBD8"
            icon={HardDrive}
            label={t("diskUsage", "Disk")}
            maximum={realtime?.resources.max_disk || 0}
            value={realtime?.resources.avg_disk || 0}
          />
        </div>
      </CardContent>
    </Card>
  );
}

function BusinessMetric({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="min-w-0 p-4">
      <div className="flex items-center gap-2 text-muted-foreground text-xs">
        <Icon className="size-4" />
        <span className="truncate">{label}</span>
      </div>
      <div className="mt-3 truncate font-semibold text-2xl tabular-nums">
        {value}
      </div>
    </div>
  );
}

function BusinessSnapshotCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t } = useTranslation("dashboard");
  const orderCount =
    (realtime?.business.new_order_count || 0) +
    (realtime?.business.renewal_order_count || 0) +
    (realtime?.business.membership_order_count || 0);
  return (
    <Card className="dashboard-panel min-h-[15rem] overflow-hidden">
      <CardHeader className="border-b py-4">
        <CardTitle>{t("todayBusiness", "Today's business")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 px-0">
        <div className="grid flex-1 grid-cols-2 divide-x divide-y sm:grid-cols-4 sm:divide-y-0">
          <BusinessMetric
            icon={CircleDollarSign}
            label={t("todayRevenue", "Today's revenue")}
            value={
              <Display
                type="currency"
                value={realtime?.business.revenue_today}
              />
            }
          />
          <BusinessMetric
            icon={ShoppingBag}
            label={t("orders", "Orders")}
            value={orderCount}
          />
          <BusinessMetric
            icon={UserPlus}
            label={t("newUsers", "New users")}
            value={realtime?.business.register_today || 0}
          />
          <BusinessMetric
            icon={TicketCheck}
            label={t("pendingTickets", "Pending tickets")}
            value={realtime?.pending_tickets || 0}
          />
        </div>
      </CardContent>
    </Card>
  );
}

function TrafficRankingCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t } = useTranslation("dashboard");
  const [period, setPeriod] = useState<RankingPeriod>("today");
  const rows =
    period === "today"
      ? realtime?.traffic.server_ranking_today || []
      : realtime?.traffic.server_ranking_monthly || [];
  const rankedRows = [...rows]
    .sort((a, b) => b.upload + b.download - (a.upload + a.download))
    .slice(0, 5);

  return (
    <Card className="dashboard-panel min-h-[15rem] overflow-hidden">
      <CardHeader className="flex-row items-center justify-between gap-3 border-b py-4">
        <div className="min-w-0">
          <CardTitle className="truncate">
            {t("nodeTrafficRanking", "Node traffic ranking")}
          </CardTitle>
          <p className="mt-1 truncate text-muted-foreground text-xs">
            {t("cumulativeTraffic", "Cumulative upload and download")}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Tabs
            onValueChange={(value) => setPeriod(value as RankingPeriod)}
            value={period}
          >
            <TabsList>
              <TabsTrigger value="today">{t("today", "Today")}</TabsTrigger>
              <TabsTrigger value="month">
                {t("thisMonth", "This month")}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <Button
            asChild
            className="hidden h-8 gap-1 px-2 sm:inline-flex"
            size="sm"
            variant="ghost"
          >
            <Link to="/dashboard/log/server-traffic">
              {t("viewAll", "View all")}
              <ExternalLink className="size-3.5" />
            </Link>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14 text-center">
                {t("rank", "Rank")}
              </TableHead>
              <TableHead>{t("nodeName", "Node")}</TableHead>
              <TableHead className="text-right">
                {t("uploadShort", "Up")}
              </TableHead>
              <TableHead className="text-right">
                {t("downloadShort", "Down")}
              </TableHead>
              <TableHead className="text-right">
                {t("total", "Total")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rankedRows.length > 0 ? (
              rankedRows.map((row, index) => (
                <TableRow key={`${period}-${row.server_id}`}>
                  <TableCell className="text-center text-muted-foreground tabular-nums">
                    {index + 1}
                  </TableCell>
                  <TableCell>
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="size-2 shrink-0 rounded-full bg-emerald-500" />
                      <span className="truncate font-medium">{row.name}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatBytes(row.upload)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatBytes(row.download)}
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    {formatBytes(row.upload + row.download)}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell className="h-40 text-center" colSpan={5}>
                  <div className="text-muted-foreground text-sm">
                    {t("empty.noTrafficData", "No traffic data yet.")}
                  </div>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

export default function Statistics() {
  const [realtime, setRealtime] = useState<API.DashboardRealtimeResponse>();
  const [realtimeConnected, setRealtimeConnected] = useState(false);

  useEffect(() => {
    if (!getAuthorizationToken()) return;
    let closed = false;
    let ws: WebSocket | undefined;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let hasFullSnapshot = false;
    let lastSeq = 0;

    const connect = () => {
      hasFullSnapshot = false;
      lastSeq = 0;
      const token = getAuthorizationToken();
      ws = new WebSocket(
        buildDashboardRealtimeWsUrl(),
        token ? ["alpha-auth", token] : undefined
      );
      ws.onopen = () => setRealtimeConnected(true);
      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as DashboardRealtimePatch;
          if (message.type !== "dashboard_realtime") return;
          const messageSeq = message.seq || 0;
          if (message.full === false) {
            if (
              !hasFullSnapshot ||
              (messageSeq > 0 && lastSeq > 0 && messageSeq !== lastSeq + 1)
            ) {
              ws?.close();
              return;
            }
            setRealtime((previous) =>
              previous ? mergeDashboardRealtime(previous, message) : previous
            );
            if (messageSeq > 0) lastSeq = messageSeq;
            return;
          }
          hasFullSnapshot = true;
          lastSeq = messageSeq;
          setRealtime((previous) =>
            previous
              ? mergeDashboardRealtime(previous, message)
              : (message as API.DashboardRealtimeResponse)
          );
        } catch {
          // Malformed realtime messages are ignored and the stream stays open.
        }
      };
      ws.onclose = () => {
        setRealtimeConnected(false);
        if (!closed) reconnectTimer = setTimeout(connect, 3000);
      };
      ws.onerror = () => ws?.close();
    };

    connect();
    return () => {
      closed = true;
      setRealtimeConnected(false);
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, []);

  return (
    <div className="flex w-full flex-col gap-4">
      <HealthSummary connected={realtimeConnected} realtime={realtime} />

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <NetworkChartCard realtime={realtime} />
        <AlertQueueCard realtime={realtime} />
      </div>

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.65fr)]">
        <TrafficRankingCard realtime={realtime} />
        <div className="grid gap-4">
          <BusinessSnapshotCard realtime={realtime} />
          <ResourceLevelsCard realtime={realtime} />
        </div>
      </div>
    </div>
  );
}
