"use client";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Input } from "@workspace/ui/components/input";
import { Label } from "@workspace/ui/components/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Separator } from "@workspace/ui/components/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@workspace/ui/components/table";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { getAuthorizationToken } from "@workspace/ui/lib/auth-token";
import { getApiBaseURL, getApiPrefix } from "@workspace/ui/lib/runtime-config";
import { cn } from "@workspace/ui/lib/utils";
import {
  getForwarderConfig,
  listForwarderConnections,
  listForwarders,
} from "@workspace/ui/services/admin/forwarder";
import {
  type Activity,
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  CheckCircle2,
  CircleOff,
  Clock3,
  Network,
  RefreshCw,
  Search,
  ServerCog,
  Unplug,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  AdminPage,
  AdminPageHeader,
  AdminSectionHeader,
} from "@/components/admin-page";
import { formatDate } from "@/utils/common";

type RatePoint = {
  at: number;
  upload: number;
  download: number;
  connections: number;
};

type ConnectionFilters = {
  eventType: string;
  listenPort: string;
  sourceIP: string;
  destination: string;
};

const defaultConnectionFilters: ConnectionFilters = {
  eventType: "all",
  listenPort: "",
  sourceIP: "",
  destination: "",
};

const emptyStatus: API.ForwarderStatus = {
  instance_id: "",
  boot_id: "",
  online: false,
  uptime: 0,
  active_connections: 0,
  accepted_connections: 0,
  failed_connections: 0,
  blocked_connections: 0,
  upload_bps: 0,
  download_bps: 0,
  upload_bytes: 0,
  download_bytes: 0,
  apply_status: "pending",
  last_seen: 0,
  rules: [],
};

function realtimeURL() {
  const base = getApiBaseURL() || window.location.origin;
  const prefix = getApiPrefix();
  const url = new URL(`${prefix}/v1/admin/forwarder/realtime/ws`, base);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

function formatRate(bytes = 0) {
  if (bytes < 1024) return `${bytes.toFixed(0)} B/s`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB/s`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MiB/s`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GiB/s`;
}

function formatBytes(bytes = 0) {
  if (bytes < 1024) return `${bytes.toFixed(0)} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MiB`;
  if (bytes < 1024 ** 4) return `${(bytes / 1024 ** 3).toFixed(2)} GiB`;
  return `${(bytes / 1024 ** 4).toFixed(2)} TiB`;
}

function formatDuration(seconds = 0) {
  const days = Math.floor(seconds / 86_400);
  const hours = Math.floor((seconds % 86_400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function shortID(value = "") {
  if (!value) return "-";
  if (value.length <= 12) return value;
  return `${value.slice(0, 6)}…${value.slice(-6)}`;
}

function syncOK(item: API.ForwarderInstance) {
  return Boolean(
    item.desired_hash &&
      item.desired_hash === item.applied_hash &&
      item.status.apply_status === "ok"
  );
}

function SummaryMetric({
  className,
  icon: Icon,
  label,
  value,
  detail,
}: {
  className?: string;
  icon: typeof Activity;
  label: string;
  value: string | number;
  detail: string;
}) {
  return (
    <div className={cn("min-w-0 px-3 py-4 sm:px-4", className)}>
      <div className="flex items-center gap-2 text-muted-foreground text-xs">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className="mt-2 font-semibold text-xl tabular-nums">{value}</div>
      <div className="mt-1 truncate text-muted-foreground text-xs">
        {detail}
      </div>
    </div>
  );
}

export default function Forwarders() {
  const { t } = useTranslation("forwarders");
  const [rows, setRows] = useState<API.ForwarderInstance[]>([]);
  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [selectedID, setSelectedID] = useState("");
  const [config, setConfig] = useState<API.ForwarderConfig>();
  const [configLoading, setConfigLoading] = useState(false);
  const [events, setEvents] = useState<API.ForwarderConnectionEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [nextCursor, setNextCursor] = useState("");
  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const [eventType, setEventType] = useState("all");
  const [listenPort, setListenPort] = useState("");
  const [sourceIP, setSourceIP] = useState("");
  const [destination, setDestination] = useState("");
  const [appliedFilters, setAppliedFilters] = useState<ConnectionFilters>(
    defaultConnectionFilters
  );
  const [history, setHistory] = useState<Record<string, RatePoint[]>>({});
  const knownIDs = useRef(new Set<string>());

  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const response = await listForwarders({ search: appliedSearch });
      const list = response.data?.data?.list || [];
      knownIDs.current = new Set(list.map((item) => item.instance_id));
      setRows(list);
      setSelectedID((current) => current || list[0]?.instance_id || "");
    } finally {
      setLoading(false);
    }
  }, [appliedSearch]);

  useEffect(() => {
    loadList();
  }, [loadList]);

  const applyRealtime = useCallback(
    (status: API.ForwarderStatus) => {
      if (!status.instance_id) return;
      if (!knownIDs.current.has(status.instance_id)) {
        loadList();
        return;
      }
      setRows((current) =>
        current.map((item) =>
          item.instance_id === status.instance_id
            ? {
                ...item,
                version: status.version || item.version,
                last_seen_at: status.last_seen || item.last_seen_at,
                applied_version: status.applied_version || item.applied_version,
                applied_hash: status.applied_hash || item.applied_hash,
                apply_status: status.apply_status || item.apply_status,
                apply_error: status.apply_error || "",
                status: { ...item.status, ...status },
              }
            : item
        )
      );
      setHistory((current) => {
        const next = [
          ...(current[status.instance_id] || []),
          {
            at: status.last_seen || Date.now(),
            upload: status.upload_bps || 0,
            download: status.download_bps || 0,
            connections: status.active_connections || 0,
          },
        ].slice(-60);
        return { ...current, [status.instance_id]: next };
      });
    },
    [loadList]
  );

  useEffect(() => {
    let socket: WebSocket | undefined;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let closed = false;
    const connect = () => {
      const token = getAuthorizationToken();
      socket = new WebSocket(
        realtimeURL(),
        token ? ["alpha-auth", token] : undefined
      );
      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as
            | API.ForwarderStatus
            | { type: string; data: API.ForwarderStatus[] };
          if ("data" in message && Array.isArray(message.data)) {
            for (const status of message.data) applyRealtime(status);
          } else {
            applyRealtime(message as API.ForwarderStatus);
          }
        } catch {
          // Ignore malformed telemetry.
        }
      };
      socket.onerror = () => socket?.close();
      socket.onclose = () => {
        if (!closed) reconnectTimer = setTimeout(connect, 3000);
      };
    };
    connect();
    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [applyRealtime]);

  const selected = useMemo(
    () => rows.find((item) => item.instance_id === selectedID),
    [rows, selectedID]
  );

  const loadConfig = useCallback(async () => {
    if (!selectedID) return;
    setConfigLoading(true);
    try {
      const response = await getForwarderConfig({ instance_id: selectedID });
      setConfig(response.data?.data);
    } finally {
      setConfigLoading(false);
    }
  }, [selectedID]);

  const loadEvents = useCallback(async () => {
    if (!selectedID) return;
    setEventsLoading(true);
    try {
      const end = Date.now();
      const response = await listForwarderConnections({
        instance_id: selectedID,
        start_time: end - 24 * 60 * 60 * 1000,
        end_time: end,
        event_type:
          appliedFilters.eventType === "all"
            ? undefined
            : appliedFilters.eventType,
        listen_port: appliedFilters.listenPort
          ? Number(appliedFilters.listenPort)
          : undefined,
        source_ip: appliedFilters.sourceIP || undefined,
        destination: appliedFilters.destination || undefined,
        size: 50,
        cursor: cursor || undefined,
      });
      setEvents(response.data?.data?.list || []);
      setNextCursor(response.data?.data?.next_cursor || "");
    } finally {
      setEventsLoading(false);
    }
  }, [appliedFilters, cursor, selectedID]);

  useEffect(() => {
    setCursor("");
    loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  const summary = useMemo(() => {
    const online = rows.filter((item) => item.status.online).length;
    const active = rows.reduce(
      (sum, item) => sum + (item.status.active_connections || 0),
      0
    );
    const upload = rows.reduce(
      (sum, item) => sum + (item.status.upload_bps || 0),
      0
    );
    const download = rows.reduce(
      (sum, item) => sum + (item.status.download_bps || 0),
      0
    );
    return { online, active, upload, download };
  }, [rows]);

  const selectedStatus = selected?.status || emptyStatus;
  const rateHistory = history[selectedID] || [];
  const runtimeRules = useMemo(
    () =>
      new Map((selectedStatus.rules || []).map((rule) => [rule.rule_id, rule])),
    [selectedStatus.rules]
  );

  return (
    <AdminPage className="pb-8">
      <AdminPageHeader
        actions={
          <div className="flex items-center gap-2">
            <Input
              className="w-64"
              onChange={(event) => setSearch(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") setAppliedSearch(search.trim());
              }}
              placeholder={t("search")}
              value={search}
            />
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  aria-label="Search"
                  onClick={() => setAppliedSearch(search.trim())}
                  size="icon"
                  variant="outline"
                >
                  <Search />
                </Button>
              </TooltipTrigger>
              <TooltipContent>搜索</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  aria-label="Refresh"
                  disabled={loading}
                  onClick={loadList}
                  size="icon"
                  variant="outline"
                >
                  <RefreshCw className={cn(loading && "animate-spin")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>刷新</TooltipContent>
            </Tooltip>
          </div>
        }
        description={t("description")}
        eyebrow="Infrastructure"
        title={t("title")}
      />

      <section className="grid grid-cols-2 border-y lg:grid-cols-4">
        <SummaryMetric
          detail={`共 ${rows.length} 个已注册实例`}
          icon={ServerCog}
          label="在线进程"
          value={`${summary.online}/${rows.length}`}
        />
        <SummaryMetric
          className="border-l"
          detail="当前已建立的转发连接"
          icon={Network}
          label="活动 TCP"
          value={summary.active}
        />
        <SummaryMetric
          className="border-t lg:border-l lg:border-t-0"
          detail="客户端到后端"
          icon={ArrowUp}
          label="上行速率"
          value={formatRate(summary.upload)}
        />
        <SummaryMetric
          className="border-t border-l lg:border-t-0"
          detail="后端到客户端"
          icon={ArrowDown}
          label="下行速率"
          value={formatRate(summary.download)}
        />
      </section>

      <section className="space-y-3">
        <AdminSectionHeader
          description="实例首次通过合法凭据注册后自动出现在这里。"
          title="进程目录"
        />
        <div className="overflow-x-auto rounded-md border">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow>
                <TableHead>进程</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>活动 TCP</TableHead>
                <TableHead>实时速率</TableHead>
                <TableHead>转发表</TableHead>
                <TableHead>版本 / 运行时间</TableHead>
                <TableHead className="text-right">最后心跳</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((item) => (
                <TableRow
                  className={cn(
                    "cursor-pointer",
                    item.instance_id === selectedID && "bg-muted/60"
                  )}
                  key={item.instance_id}
                  onClick={() => {
                    setCursor("");
                    setCursorHistory([]);
                    setSelectedID(item.instance_id);
                  }}
                >
                  <TableCell>
                    <div className="font-medium">
                      {item.name || item.hostname}
                    </div>
                    <div className="mt-1 text-muted-foreground text-xs">
                      {item.public_ip} · {shortID(item.instance_id)}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={item.status.online ? "default" : "secondary"}
                    >
                      {item.status.online ? "在线" : "离线"}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-medium tabular-nums">
                    {item.status.active_connections || 0}
                  </TableCell>
                  <TableCell>
                    <div className="text-xs tabular-nums">
                      ↑ {formatRate(item.status.upload_bps)}
                    </div>
                    <div className="mt-1 text-muted-foreground text-xs tabular-nums">
                      ↓ {formatRate(item.status.download_bps)}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {syncOK(item) ? (
                        <CheckCircle2 className="size-4 text-emerald-600" />
                      ) : (
                        <AlertTriangle className="size-4 text-amber-600" />
                      )}
                      <span>{item.status.rules?.length || 0} 条</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>{item.version || "-"}</div>
                    <div className="mt-1 text-muted-foreground text-xs">
                      {formatDuration(item.status.uptime)}
                    </div>
                  </TableCell>
                  <TableCell className="text-right text-muted-foreground text-xs">
                    {item.last_seen_at ? formatDate(item.last_seen_at) : "-"}
                  </TableCell>
                </TableRow>
              ))}
              {!loading && rows.length === 0 ? (
                <TableRow>
                  <TableCell className="h-28 text-center" colSpan={7}>
                    <CircleOff className="mx-auto mb-2 size-5 text-muted-foreground" />
                    <span className="text-muted-foreground text-sm">
                      暂无已注册的转发进程
                    </span>
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </section>

      {selected ? (
        <section className="space-y-4 border-t pt-5">
          <AdminSectionHeader
            description={`${selected.hostname} · ${selected.public_ip} · ${selected.instance_id}`}
            title={selected.name || selected.hostname}
          />
          {!selected.status.online ? (
            <Alert>
              <Unplug />
              <AlertTitle>进程当前离线</AlertTitle>
              <AlertDescription>
                页面保留最后一次状态；实时指标在进程恢复心跳后继续更新。
              </AlertDescription>
            </Alert>
          ) : null}
          {selected.apply_error ? (
            <Alert variant="destructive">
              <AlertTriangle />
              <AlertTitle>转发表应用异常</AlertTitle>
              <AlertDescription>{selected.apply_error}</AlertDescription>
            </Alert>
          ) : null}

          <Tabs defaultValue="overview">
            <TabsList>
              <TabsTrigger value="overview">{t("overview")}</TabsTrigger>
              <TabsTrigger value="routes">{t("routes")}</TabsTrigger>
              <TabsTrigger value="connections">{t("connections")}</TabsTrigger>
            </TabsList>

            <TabsContent className="mt-5 space-y-5" value="overview">
              <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_280px]">
                <div className="h-64 min-w-0 border-y py-4">
                  {rateHistory.length > 1 ? (
                    <ResponsiveContainer height="100%" width="100%">
                      <AreaChart data={rateHistory}>
                        <defs>
                          <linearGradient
                            id="forward-upload"
                            x1="0"
                            x2="0"
                            y1="0"
                            y2="1"
                          >
                            <stop
                              offset="5%"
                              stopColor="#2563eb"
                              stopOpacity={0.24}
                            />
                            <stop
                              offset="95%"
                              stopColor="#2563eb"
                              stopOpacity={0}
                            />
                          </linearGradient>
                          <linearGradient
                            id="forward-download"
                            x1="0"
                            x2="0"
                            y1="0"
                            y2="1"
                          >
                            <stop
                              offset="5%"
                              stopColor="#16a34a"
                              stopOpacity={0.2}
                            />
                            <stop
                              offset="95%"
                              stopColor="#16a34a"
                              stopOpacity={0}
                            />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} />
                        <XAxis
                          dataKey="at"
                          tickFormatter={(value) =>
                            new Date(value).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                              second: "2-digit",
                            })
                          }
                          tickLine={false}
                        />
                        <YAxis
                          tickFormatter={(value) => formatRate(value)}
                          tickLine={false}
                          width={82}
                        />
                        <ChartTooltip
                          formatter={(value) => formatRate(Number(value))}
                          labelFormatter={(value) => formatDate(Number(value))}
                        />
                        <Area
                          dataKey="upload"
                          fill="url(#forward-upload)"
                          name="上行"
                          stroke="#2563eb"
                          type="monotone"
                        />
                        <Area
                          dataKey="download"
                          fill="url(#forward-download)"
                          name="下行"
                          stroke="#16a34a"
                          type="monotone"
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  ) : (
                    <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                      等待至少两个实时采样点
                    </div>
                  )}
                </div>
                <dl className="divide-y border-y text-sm">
                  {[
                    ["活动 TCP", selectedStatus.active_connections],
                    ["累计接受", selectedStatus.accepted_connections],
                    ["连接失败", selectedStatus.failed_connections],
                    ["策略拦截", selectedStatus.blocked_connections],
                    ["累计上行", formatBytes(selectedStatus.upload_bytes)],
                    ["累计下行", formatBytes(selectedStatus.download_bytes)],
                  ].map(([label, value]) => (
                    <div
                      className="flex justify-between gap-4 py-3"
                      key={label}
                    >
                      <dt className="text-muted-foreground">{label}</dt>
                      <dd className="font-medium tabular-nums">{value}</dd>
                    </div>
                  ))}
                </dl>
              </div>
              <div className="grid gap-4 border-y py-4 sm:grid-cols-3">
                <div>
                  <div className="text-muted-foreground text-xs">期望配置</div>
                  <div className="mt-1 font-mono text-sm">
                    {selected.desired_version || "-"}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">实际配置</div>
                  <div className="mt-1 font-mono text-sm">
                    {selected.applied_version || "-"}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">同步状态</div>
                  <div className="mt-1 flex items-center gap-2 font-medium text-sm">
                    {syncOK(selected) ? (
                      <CheckCircle2 className="size-4 text-emerald-600" />
                    ) : (
                      <Clock3 className="size-4 text-amber-600" />
                    )}
                    {syncOK(selected) ? "已同步" : "等待同步"}
                  </div>
                </div>
              </div>
            </TabsContent>

            <TabsContent className="mt-5 space-y-4" value="routes">
              <AdminSectionHeader
                actions={
                  <Button
                    disabled={configLoading}
                    onClick={loadConfig}
                    size="sm"
                    variant="outline"
                  >
                    <RefreshCw
                      className={cn(configLoading && "animate-spin")}
                    />
                    刷新转发表
                  </Button>
                }
                description={`期望版本 ${config?.version || "-"}；运行状态来自进程实时回报。`}
                title="该进程的转发表"
              />
              {config?.warnings?.length ? (
                <Alert>
                  <AlertTriangle />
                  <AlertTitle>配置生成警告</AlertTitle>
                  <AlertDescription>
                    {config.warnings.join("；")}
                  </AlertDescription>
                </Alert>
              ) : null}
              <div className="overflow-x-auto rounded-md border">
                <Table className="min-w-[900px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead>监听</TableHead>
                      <TableHead>目标</TableHead>
                      <TableHead>节点 / 入站</TableHead>
                      <TableHead>运行状态</TableHead>
                      <TableHead>活动 TCP</TableHead>
                      <TableHead>实时速率</TableHead>
                      <TableHead>累计流量</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(config?.rules || []).map((rule) => {
                      const runtime = runtimeRules.get(rule.rule_id);
                      return (
                        <TableRow key={rule.rule_id}>
                          <TableCell className="font-mono text-xs">
                            {rule.listen_ip || "*"}:{rule.listen_port}
                          </TableCell>
                          <TableCell className="font-mono text-xs">
                            {rule.target_host}:{rule.target_port}
                          </TableCell>
                          <TableCell>
                            <div>
                              {rule.node_name || `#${rule.node_id || "-"}`}
                            </div>
                            <div className="mt-1 text-muted-foreground text-xs">
                              {rule.server_name || `#${rule.server_id || "-"}`}{" "}
                              · {rule.inbound || "-"}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant={
                                runtime?.status === "running"
                                  ? "default"
                                  : "secondary"
                              }
                            >
                              {runtime?.status === "running"
                                ? "运行中"
                                : "未回报"}
                            </Badge>
                          </TableCell>
                          <TableCell className="tabular-nums">
                            {runtime?.active_connections || 0}
                          </TableCell>
                          <TableCell className="text-xs tabular-nums">
                            <div>↑ {formatRate(runtime?.upload_bps)}</div>
                            <div className="mt-1 text-muted-foreground">
                              ↓ {formatRate(runtime?.download_bps)}
                            </div>
                          </TableCell>
                          <TableCell className="text-xs tabular-nums">
                            <div>↑ {formatBytes(runtime?.upload_bytes)}</div>
                            <div className="mt-1 text-muted-foreground">
                              ↓ {formatBytes(runtime?.download_bytes)}
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                    {!configLoading && (config?.rules.length || 0) === 0 ? (
                      <TableRow>
                        <TableCell
                          className="h-24 text-center text-muted-foreground"
                          colSpan={7}
                        >
                          当前进程没有匹配到转发规则
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </div>
            </TabsContent>

            <TabsContent className="mt-5 space-y-4" value="connections">
              <div className="grid gap-3 border-y py-4 md:grid-cols-2 xl:grid-cols-5">
                <div className="space-y-1.5">
                  <Label>事件</Label>
                  <Select onValueChange={setEventType} value={eventType}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部事件</SelectItem>
                      <SelectItem value="connected">已建立</SelectItem>
                      <SelectItem value="closed">已断开</SelectItem>
                      <SelectItem value="dial_failed">连接失败</SelectItem>
                      <SelectItem value="blocked">策略拦截</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label>监听端口</Label>
                  <Input
                    max={65_535}
                    min={1}
                    onChange={(event) => setListenPort(event.target.value)}
                    step={1}
                    type="number"
                    value={listenPort}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>来源 IP</Label>
                  <Input
                    onChange={(event) => setSourceIP(event.target.value)}
                    value={sourceIP}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>目标</Label>
                  <Input
                    onChange={(event) => setDestination(event.target.value)}
                    value={destination}
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    className="w-full"
                    disabled={eventsLoading}
                    onClick={() => {
                      setCursor("");
                      setCursorHistory([]);
                      setAppliedFilters({
                        eventType,
                        listenPort: listenPort.trim(),
                        sourceIP: sourceIP.trim(),
                        destination: destination.trim(),
                      });
                    }}
                  >
                    <Search />
                    查询
                  </Button>
                </div>
              </div>
              <div className="overflow-x-auto rounded-md border">
                <Table className="min-w-[1100px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead>时间 / 事件</TableHead>
                      <TableHead>连接 ID</TableHead>
                      <TableHead>来源</TableHead>
                      <TableHead>监听</TableHead>
                      <TableHead>配置目标 / 实际目标</TableHead>
                      <TableHead>转发出口</TableHead>
                      <TableHead>流量 / 持续时间</TableHead>
                      <TableHead>关闭信息</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {events.map((event) => (
                      <TableRow key={event.event_id}>
                        <TableCell>
                          <Badge variant="outline">{event.event_type}</Badge>
                          <div className="mt-2 text-muted-foreground text-xs">
                            {formatDate(event.event_time)}
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {shortID(event.connection_id)}
                          <div className="mt-1 text-muted-foreground">
                            {shortID(event.rule_id)}
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {event.source_ip}:{event.source_port}
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {event.listen_ip || "*"}:{event.listen_port}
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          <div>
                            {event.configured_target_host}:
                            {event.configured_target_port}
                          </div>
                          <div className="mt-1 text-muted-foreground">
                            {event.resolved_target_ip
                              ? `${event.resolved_target_ip}:${event.resolved_target_port}`
                              : "-"}
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {event.forwarder_outbound_ip
                            ? `${event.forwarder_outbound_ip}:${event.forwarder_outbound_port}`
                            : "-"}
                        </TableCell>
                        <TableCell className="text-xs tabular-nums">
                          <div>↑ {formatBytes(event.upload_bytes)}</div>
                          <div className="mt-1">
                            ↓ {formatBytes(event.download_bytes)}
                          </div>
                          <div className="mt-1 text-muted-foreground">
                            {event.duration_ms
                              ? `${event.duration_ms} ms`
                              : "-"}
                          </div>
                        </TableCell>
                        <TableCell className="text-xs">
                          <div>{event.close_initiator || "-"}</div>
                          <div className="mt-1 text-muted-foreground">
                            {event.close_reason || "-"}
                          </div>
                          {event.error ? (
                            <div className="mt-1 max-w-48 truncate text-destructive">
                              {event.error}
                            </div>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    ))}
                    {!eventsLoading && events.length === 0 ? (
                      <TableRow>
                        <TableCell
                          className="h-24 text-center text-muted-foreground"
                          colSpan={8}
                        >
                          当前筛选范围内没有连接事件
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </div>
              <div className="flex items-center justify-end gap-2">
                <span className="mr-2 text-muted-foreground text-xs">
                  第 {cursorHistory.length + 1} 页
                </span>
                <Button
                  disabled={cursorHistory.length === 0 || eventsLoading}
                  onClick={() => {
                    const previous = cursorHistory.at(-1) || "";
                    setCursorHistory((current) => current.slice(0, -1));
                    setCursor(previous);
                  }}
                  variant="outline"
                >
                  上一页
                </Button>
                <Button
                  disabled={!nextCursor || eventsLoading}
                  onClick={() => {
                    setCursorHistory((current) => [...current, cursor]);
                    setCursor(nextCursor);
                  }}
                  variant="outline"
                >
                  下一页
                </Button>
              </div>
            </TabsContent>
          </Tabs>
        </section>
      ) : null}
      <Separator />
    </AdminPage>
  );
}
