"use client";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@workspace/ui/components/alert-dialog";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { Input } from "@workspace/ui/components/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import { ConfirmButton } from "@/components/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import { cn } from "@workspace/ui/lib/utils";
import {
  createServer,
  deleteServer,
  filterServerList,
  queryServerNetworkRateHistory,
  updateServer,
} from "@workspace/ui/services/admin/server";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import {
  Activity,
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  Braces,
  ChevronDown,
  CloudDownload,
  Copy,
  Gauge,
  MapPin,
  MoreHorizontal,
  Network,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Server as ServerIcon,
  Settings2,
  Trash2,
  UsersRound,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Line, LineChart } from "recharts";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { mergeRealtimeStatus } from "@/lib/server-realtime-data";
import { useServerRealtime } from "@/lib/use-server-realtime";
import { useNode } from "@/stores/node";
import { useServer } from "@/stores/server";
import ServerXrayTemplateBindForm from "../xray-templates/server-bind-form";
import DynamicMultiplier from "./dynamic-multiplier";
import {
  mapServerNetworkRateSeries,
  mergeServerNetworkRatePoint,
  serverStatusToNetworkRatePoint,
} from "./network-rate-data";
import ServerConfig from "./server-config";
import {
  buildAttentionBreakdown,
  buildFleetSummary,
  configHealth,
  type FleetFilter,
  getOverallHealth,
  type HealthState,
  nodeHealth,
  resourceHealth,
  serverNeedsAttention,
  statusIsOffline,
  xrayHealth,
} from "./server-fleet-data";
import ServerForm from "./server-form";
import ServerInstall from "./server-install";

type ToolPanel = "multiplier" | "config" | null;

function usageColor(value: number) {
  if (value >= 90) return "bg-red-500";
  if (value >= 75) return "bg-amber-500";
  return "bg-emerald-500";
}

function formatBitrate(value?: number) {
  const bits = Math.max(0, Number(value) || 0);
  if (bits < 1000) return `${bits.toFixed(0)} bps`;
  if (bits < 1_000_000) {
    const precision = bits >= 100_000 ? 0 : 1;
    return `${(bits / 1000).toFixed(precision)} Kbps`;
  }
  const precision = bits >= 10_000_000 ? 0 : 2;
  return `${(bits / 1_000_000).toFixed(precision)} Mbps`;
}

function formatRelativeTime(
  timestamp: number | undefined,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (!timestamp) return t("notAvailable", "Not available");
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (seconds < 60) {
    return t("relative.seconds", "{{count}} seconds ago", { count: seconds });
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return t("relative.minutes", "{{count}} minutes ago", { count: minutes });
  }
  const hours = Math.floor(minutes / 60);
  return t("relative.hours", "{{count}} hours ago", { count: hours });
}

function healthTone(state: HealthState) {
  if (state === "ok") return "bg-emerald-500 text-emerald-500";
  if (state === "warning") return "bg-amber-500 text-amber-500";
  if (state === "error") return "bg-red-500 text-red-500";
  return "bg-muted-foreground text-muted-foreground";
}

function MiniHealthStatus({
  state,
  label,
}: {
  state: HealthState;
  label: string;
}) {
  return (
    <span className="inline-flex items-center gap-1.5 text-[10px] text-muted-foreground">
      <span
        aria-hidden="true"
        className={cn("size-1.5 rounded-full", healthTone(state))}
      />
      {label}
    </span>
  );
}

function ResourceMetric({ label, value }: { label: string; value: number }) {
  const normalized = Math.max(0, Math.min(100, Number(value) || 0));
  return (
    <div className="grid grid-cols-[34px_1fr_34px] items-center gap-2">
      <span className="text-[10px] text-muted-foreground">{label}</span>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full", usageColor(normalized))}
          style={{ width: `${normalized}%` }}
        />
      </div>
      <span className="text-right font-medium text-[11px] tabular-nums">
        {normalized.toFixed(0)}%
      </span>
    </div>
  );
}

function ResourceCell({ status }: { status: Partial<API.ServerStatus> }) {
  return (
    <div className="min-w-[142px] space-y-2">
      <ResourceMetric label="CPU" value={Number(status.cpu) || 0} />
      <ResourceMetric label="MEM" value={Number(status.mem) || 0} />
      <ResourceMetric label="DISK" value={Number(status.disk) || 0} />
    </div>
  );
}

function TrafficSparkline({
  points,
  uploadKey,
  downloadKey,
  offline,
}: {
  points: API.ServerNetworkRatePoint[];
  uploadKey: keyof API.ServerNetworkRatePoint;
  downloadKey: keyof API.ServerNetworkRatePoint;
  offline: boolean;
}) {
  if (points.length < 2) return <span className="block h-6 w-[72px]" />;
  return (
    <LineChart
      aria-hidden="true"
      data={points}
      height={24}
      margin={{ bottom: 2, left: 1, right: 1, top: 2 }}
      width={72}
    >
      <Line
        dataKey={downloadKey}
        dot={false}
        isAnimationActive={false}
        stroke={offline ? "#ef4444" : "#38bdf8"}
        strokeWidth={1.3}
        type="monotone"
      />
      <Line
        dataKey={uploadKey}
        dot={false}
        isAnimationActive={false}
        stroke={offline ? "#ef4444" : "#2dd4bf"}
        strokeWidth={1.3}
        type="monotone"
      />
    </LineChart>
  );
}

function TrafficLane({
  label,
  upload,
  download,
  points,
  uploadKey,
  downloadKey,
  offline,
}: {
  label: string;
  upload?: number;
  download?: number;
  points: API.ServerNetworkRatePoint[];
  uploadKey: keyof API.ServerNetworkRatePoint;
  downloadKey: keyof API.ServerNetworkRatePoint;
  offline: boolean;
}) {
  return (
    <div className="grid grid-cols-[38px_minmax(0,1fr)_72px] items-center gap-2">
      <span className="text-[10px] text-muted-foreground">{label}</span>
      <div className="grid min-w-0 grid-cols-2 gap-2 text-[10px] tabular-nums">
        <span className="flex min-w-0 items-center gap-1 font-medium">
          <ArrowUp className="size-3 shrink-0 text-teal-500" />
          <span className="truncate">{formatBitrate(upload)}</span>
        </span>
        <span className="flex min-w-0 items-center gap-1 font-medium">
          <ArrowDown className="size-3 shrink-0 text-sky-500" />
          <span className="truncate">{formatBitrate(download)}</span>
        </span>
      </div>
      <TrafficSparkline
        downloadKey={downloadKey}
        offline={offline}
        points={points}
        uploadKey={uploadKey}
      />
    </div>
  );
}

function TrafficCell({
  points,
  status,
}: {
  points: API.ServerNetworkRatePoint[];
  status: Partial<API.ServerStatus>;
}) {
  const { t } = useTranslation("servers");
  const offline = statusIsOffline(status);
  return (
    <div className="min-w-[272px] space-y-2">
      <TrafficLane
        download={status.net_rx_bps}
        downloadKey="net_rx_bps"
        label={t("systemShort", "System")}
        offline={offline}
        points={points}
        upload={status.net_tx_bps}
        uploadKey="net_tx_bps"
      />
      <TrafficLane
        download={status.xray_rx_bps}
        downloadKey="xray_rx_bps"
        label="Xray"
        offline={offline}
        points={points}
        upload={status.xray_tx_bps}
        uploadKey="xray_tx_bps"
      />
    </div>
  );
}

function IdentityCell({
  server,
  status,
}: {
  server: API.ServerListItem;
  status: Partial<API.ServerStatus>;
}) {
  const { t } = useTranslation("servers");
  const health = getOverallHealth(status);
  const location =
    [server.country, server.city].filter(Boolean).join(" / ") ||
    t("notAvailable", "Not available");
  return (
    <div className="flex min-w-[196px] items-start gap-3">
      <span
        className={cn(
          "mt-1.5 size-2 shrink-0 rounded-full",
          health === "offline" && "bg-red-500",
          health === "attention" && "bg-amber-500",
          health === "healthy" && "bg-emerald-500"
        )}
      />
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-baseline gap-2">
          <p className="truncate font-semibold text-sm">{server.name}</p>
          <span className="shrink-0 text-[10px] text-muted-foreground tabular-nums">
            #{server.id}
          </span>
        </div>
        <p className="mt-1 truncate font-medium text-[11px] tabular-nums">
          {server.address}
        </p>
        <p className="mt-1 flex min-w-0 items-center gap-1 text-[10px] text-muted-foreground">
          <MapPin className="size-3 shrink-0" />
          <span className="truncate">{location}</span>
        </p>
      </div>
    </div>
  );
}

function HealthCell({ status }: { status: Partial<API.ServerStatus> }) {
  const { t } = useTranslation("servers");
  const overall = getOverallHealth(status);
  const node = nodeHealth(status);
  const xray = xrayHealth(status);
  const config = configHealth(status);
  const resources = resourceHealth(status);
  const issue = (() => {
    if (overall === "offline")
      return t("health.offlineDetail", "Heartbeat lost");
    if (xray === "error") {
      return status.xray_stats_error || t("health.xrayStopped", "Xray stopped");
    }
    if (config === "error") {
      return (
        status.last_apply_error ||
        status.last_config_error ||
        t("health.configFailed", "Configuration failed")
      );
    }
    if (config === "warning") {
      return t("health.configPending", "Configuration waiting to apply");
    }
    if (resources !== "ok") {
      return t("health.resourcePressure", "Resource pressure detected");
    }
    return t("health.healthyDetail", "Agent, Xray, and config are healthy");
  })();
  const label =
    overall === "healthy"
      ? t("health.healthy", "Healthy")
      : overall === "offline"
        ? t("offline", "Offline")
        : t("health.attention", "Needs attention");

  return (
    <div className="min-w-[174px]">
      <p
        className={cn(
          "flex items-center gap-2 font-medium text-xs",
          overall === "healthy" && "text-emerald-500",
          overall === "attention" && "text-amber-500",
          overall === "offline" && "text-red-500"
        )}
      >
        <span className="size-1.5 rounded-full bg-current" />
        {label}
      </p>
      <p
        className="mt-1.5 truncate text-[10px] text-muted-foreground"
        title={issue}
      >
        {issue}
      </p>
      <div className="mt-2.5 flex items-center gap-3">
        <MiniHealthStatus label={t("node", "Agent")} state={node} />
        <MiniHealthStatus label="Xray" state={xray} />
        <MiniHealthStatus label={t("config", "Config")} state={config} />
      </div>
    </div>
  );
}

function ActivityCell({
  server,
  status,
}: {
  server: API.ServerListItem;
  status: Partial<API.ServerStatus>;
}) {
  const { t } = useTranslation("servers");
  return (
    <div className="min-w-[118px]">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
            <Activity className="size-3" />
            {t("connectionsShort", "Sessions")}
          </span>
          <p className="mt-1 font-semibold text-sm tabular-nums">
            {status.connections || 0}
          </p>
        </div>
        <div>
          <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
            <UsersRound className="size-3" />
            {t("usersShort", "Users")}
          </span>
          <p className="mt-1 font-semibold text-sm tabular-nums">
            {status.online_users || 0}
          </p>
        </div>
      </div>
      <p
        className={cn(
          "mt-2.5 flex items-center gap-1.5 text-[10px] tabular-nums",
          statusIsOffline(status) ? "text-red-500" : "text-muted-foreground"
        )}
      >
        <span className="size-1.5 rounded-full bg-current" />
        {formatRelativeTime(status.last_seen || server.last_reported_at, t)}
      </p>
    </div>
  );
}

function ServerRowActions({
  row,
  referenced,
  onEdit,
  onXray,
  onDeleted,
}: {
  row: API.ServerListItem;
  referenced: boolean;
  onEdit: () => void;
  onXray: () => void;
  onDeleted: () => void;
}) {
  const { t } = useTranslation("servers");
  const [deleteOpen, setDeleteOpen] = useState(false);

  async function copyServer() {
    await createServer({
      name: row.name,
      country: row.country,
      city: row.city,
      address: row.address,
    });
    toast.success(t("copied", "Copied"));
    onDeleted();
  }

  async function removeServer() {
    await deleteServer({ id: row.id });
    toast.success(t("deleted", "Deleted"));
    onDeleted();
  }

  return (
    <>
      <div className="flex items-center justify-end gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              aria-label={t("edit", "Edit")}
              className="size-7"
              onClick={onEdit}
              size="icon-sm"
              variant="ghost"
            >
              <Pencil />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("edit", "Edit")}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <ServerInstall
                server={row}
                trigger={
                  <Button
                    aria-label={t("connect", "Connect and deploy")}
                    className="size-7"
                    size="icon-sm"
                    variant="ghost"
                  >
                    <CloudDownload />
                  </Button>
                }
              />
            </span>
          </TooltipTrigger>
          <TooltipContent>{t("connect", "Connect and deploy")}</TooltipContent>
        </Tooltip>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              aria-label={t("moreActions", "More actions")}
              className="size-7"
              size="icon-sm"
              variant="ghost"
            >
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuItem onSelect={onXray}>
              <Braces />
              {t("xrayTemplates", "Xray templates")}
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={copyServer}>
              <Copy />
              {t("copy", "Copy")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={referenced}
              onSelect={() => setDeleteOpen(true)}
              title={
                referenced
                  ? t(
                      "deleteReferencedHint",
                      "Remove linked nodes and Xray templates before deleting this server."
                    )
                  : undefined
              }
              variant="destructive"
            >
              <Trash2 />
              {t("delete", "Delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <AlertDialog onOpenChange={setDeleteOpen} open={deleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("confirmDeleteTitle", "Delete this server?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("confirmDeleteDesc", "This action cannot be undone.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel", "Cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={removeServer}>
              {t("confirm", "Confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

export default function Servers() {
  const { t } = useTranslation("servers");
  const { isServerReferencedByNodes } = useNode();
  const { servers, fetchServers } = useServer();
  const [loading, setLoading] = useState(false);
  const [editingServer, setEditingServer] = useState<API.ServerListItem | null>(
    null
  );
  const [xrayTemplateServer, setXrayTemplateServer] =
    useState<API.ServerListItem | null>(null);
  const [toolPanel, setToolPanel] = useState<ToolPanel>(null);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [fleetFilter, setFleetFilter] = useState<FleetFilter>("all");
  const [regionFilter, setRegionFilter] = useState("all");
  const [pageServers, setPageServers] = useState<API.ServerListItem[]>([]);
  const [fleetTotal, setFleetTotal] = useState(0);
  const [realtimeStatus, setRealtimeStatus] = useState<
    Record<number, Partial<API.ServerStatus>>
  >({});
  const [networkRateHistory, setNetworkRateHistory] = useState<
    Record<number, API.ServerNetworkRatePoint[]>
  >({});
  const [historyServerIds, setHistoryServerIds] = useState<number[]>([]);
  const tableRef = useRef<ProTableActions>(null);

  const applyRealtimeStatus = useCallback(
    (serverId: number, status: Partial<API.ServerStatus>) => {
      setNetworkRateHistory((previous) => ({
        ...previous,
        [serverId]: mergeServerNetworkRatePoint(
          previous[serverId] || [],
          serverStatusToNetworkRatePoint(status)
        ),
      }));
      setRealtimeStatus((previous) => {
        const previousStatus = previous[serverId];
        const nextStatus = mergeRealtimeStatus(previousStatus, status);
        if (nextStatus === previousStatus) return previous;
        return { ...previous, [serverId]: nextStatus };
      });
    },
    []
  );

  useServerRealtime(
    historyServerIds,
    applyRealtimeStatus,
    historyServerIds.length > 0
  );

  const getStatus = (server: API.ServerListItem) => {
    const realtime = realtimeStatus[server.id];
    const status = realtime
      ? mergeRealtimeStatus(server.status, realtime)
      : server.status || {};
    return statusIsOffline(status) ? { ...status, status: "offline" } : status;
  };

  const fleetServers = useMemo(
    () =>
      (search
        ? pageServers
        : servers.length > 0
          ? (servers as API.ServerListItem[])
          : pageServers
      ).map((server) => ({ server, status: getStatus(server) })),
    [pageServers, realtimeStatus, search, servers]
  );

  const summary = useMemo(
    () =>
      buildFleetSummary(
        fleetServers.map(({ status }) => status),
        fleetTotal || fleetServers.length
      ),
    [fleetServers, fleetTotal]
  );

  const attentionBreakdown = useMemo(
    () => buildAttentionBreakdown(fleetServers.map(({ status }) => status)),
    [fleetServers]
  );

  const regionOptions = useMemo(() => {
    const values = new Set<string>();
    for (const { server } of fleetServers) {
      const value = [server.country, server.city].filter(Boolean).join(" / ");
      if (value) values.add(value);
    }
    return Array.from(values).sort((a, b) => a.localeCompare(b));
  }, [fleetServers]);

  const hasActiveFilters =
    Boolean(searchInput) || fleetFilter !== "all" || regionFilter !== "all";

  useEffect(() => {
    const timer = window.setTimeout(() => setSearch(searchInput.trim()), 250);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    const timer = window.setTimeout(() => tableRef.current?.refresh(), 0);
    return () => window.clearTimeout(timer);
  }, [fleetFilter, regionFilter, search]);

  useEffect(() => {
    if (historyServerIds.length === 0) return;
    let cancelled = false;
    const endTime = Date.now();
    queryServerNetworkRateHistory({
      server_ids: historyServerIds.join(","),
      start_time: endTime - 5 * 60 * 1000,
      end_time: endTime,
      interval: "5s",
    })
      .then(({ data }) => {
        if (cancelled || !data.data) return;
        const historical = mapServerNetworkRateSeries(data.data.series);
        setNetworkRateHistory((previous) => {
          const next = { ...previous };
          for (const serverID of historyServerIds) {
            let points = historical[serverID] || [];
            for (const point of previous[serverID] || []) {
              points = mergeServerNetworkRatePoint(points, point);
            }
            next[serverID] = points;
          }
          return next;
        });
      })
      .catch(() => {
        // Realtime values remain available when history cannot be loaded.
      });
    return () => {
      cancelled = true;
    };
  }, [historyServerIds]);

  async function refreshAll() {
    tableRef.current?.refresh();
    await fetchServers();
  }

  return (
    <AdminPage className="pb-6">
      <AdminPageHeader
        actions={
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  {t("managementTools", "Management tools")}
                  <ChevronDown />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-72">
                <DropdownMenuItem
                  className="items-start gap-3 py-2.5"
                  onSelect={() => setToolPanel("multiplier")}
                >
                  <Gauge className="mt-0.5" />
                  <span className="min-w-0">
                    <span className="block font-medium">
                      {t(
                        "server_config.dynamic_multiplier",
                        "Dynamic multiplier"
                      )}
                    </span>
                    <span className="mt-0.5 block text-muted-foreground text-xs">
                      {t(
                        "server_config.dynamic_multiplier_menu_desc",
                        "Adjust accounting by time period."
                      )}
                    </span>
                  </span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="items-start gap-3 py-2.5"
                  onSelect={() => setToolPanel("config")}
                >
                  <Settings2 className="mt-0.5" />
                  <span className="min-w-0">
                    <span className="block font-medium">
                      {t("server_config.title", "Node configuration")}
                    </span>
                    <span className="mt-0.5 block text-muted-foreground text-xs">
                      {t(
                        "server_config.node_config_menu_desc",
                        "Manage sync, reporting, and agent security."
                      )}
                    </span>
                  </span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <ServerForm
              loading={loading}
              onSubmit={async (values) => {
                setLoading(true);
                try {
                  await createServer(
                    values as unknown as API.CreateServerRequest
                  );
                  toast.success(t("created", "Created"));
                  await refreshAll();
                  return true;
                } catch {
                  return false;
                } finally {
                  setLoading(false);
                }
              }}
              title={t("drawerCreateTitle", "Create server")}
              trigger={
                <span className="inline-flex items-center gap-2">
                  <Plus className="size-4" />
                  {t("createServer", "Create server")}
                </span>
              }
            />
          </div>
        }
        description={t(
          "pageDescription",
          "Operate the server fleet, resolve health issues, and inspect live traffic from one view."
        )}
        eyebrow={t("workspace", "Infrastructure")}
        title={t("managementPageTitle", "Server management")}
      />

      <ToggleGroup
        className="admin-stat-strip grid w-full grid-cols-2 overflow-hidden rounded-lg border bg-card lg:grid-cols-4"
        onValueChange={(value) => {
          if (value) setFleetFilter(value as FleetFilter);
        }}
        type="single"
        value={fleetFilter}
      >
        {[
          {
            filter: "all" as const,
            icon: ServerIcon,
            label: t("summary.total", "Servers"),
            value: summary.total,
            detail: t("summary.totalDetail", "Registered fleet"),
            tone: "text-foreground",
          },
          {
            filter: "online" as const,
            icon: Network,
            label: t("summary.online", "Online"),
            value: summary.online,
            detail: t(
              "summary.onlineDetail",
              "{{online}} of {{total}} connected",
              {
                online: summary.online,
                total: summary.total,
              }
            ),
            tone: "text-emerald-500",
          },
          {
            filter: "attention" as const,
            icon: AlertTriangle,
            label: t("summary.attention", "Needs attention"),
            value: summary.attention,
            detail: t("summary.attentionDetail", "Offline or degraded"),
            tone: "text-amber-500",
          },
          {
            filter: "pending" as const,
            icon: Settings2,
            label: t("summary.pending", "Config pending"),
            value: summary.pending,
            detail: t("summary.pendingDetail", "Waiting to apply"),
            tone: "text-amber-500",
          },
        ].map((item) => {
          const Icon = item.icon;
          const active = fleetFilter === item.filter;
          return (
            <ToggleGroupItem
              aria-pressed={active}
              className={cn(
                "h-auto min-w-0 flex-col items-stretch rounded-none border-0 border-border/70 border-r border-b px-4 py-3 text-left even:border-r-0 [&:nth-last-child(-n+2)]:border-b-0 lg:border-b-0 lg:border-r lg:px-5 lg:py-4 lg:even:border-r lg:last:border-r-0",
                active && "bg-primary/5"
              )}
              key={item.label}
              value={item.filter}
            >
              <div className="flex items-center justify-between gap-3">
                <p className="text-muted-foreground text-xs">{item.label}</p>
                <Icon className={cn(item.tone)} />
              </div>
              <p
                className={cn(
                  "mt-1 font-semibold text-2xl tabular-nums",
                  item.tone
                )}
              >
                {item.value}
              </p>
              <p className="mt-1 truncate text-[10px] text-muted-foreground">
                {item.detail}
              </p>
            </ToggleGroupItem>
          );
        })}
      </ToggleGroup>

      {summary.attention > 0 ? (
        <Alert>
          <AlertTriangle />
          <AlertTitle>
            {t("attentionCount", "{{count}} servers need attention", {
              count: summary.attention,
            })}
          </AlertTitle>
          <AlertDescription className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs">
              {attentionBreakdown.offline > 0 ? (
                <span>
                  {t("attention.offline", "{{count}} offline", {
                    count: attentionBreakdown.offline,
                  })}
                </span>
              ) : null}
              {attentionBreakdown.xray > 0 ? (
                <span>
                  {t("attention.xray", "{{count}} Xray issue", {
                    count: attentionBreakdown.xray,
                  })}
                </span>
              ) : null}
              {attentionBreakdown.config > 0 ? (
                <span>
                  {t("attention.config", "{{count}} config issue", {
                    count: attentionBreakdown.config,
                  })}
                </span>
              ) : null}
              {attentionBreakdown.resources > 0 ? (
                <span>
                  {t("attention.resources", "{{count}} under pressure", {
                    count: attentionBreakdown.resources,
                  })}
                </span>
              ) : null}
            </div>
            <Button
              className="w-full shrink-0 sm:w-auto"
              onClick={() => setFleetFilter("attention")}
              size="sm"
              variant="outline"
            >
              {t("viewAttention", "View issues")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      <section className="flex flex-col gap-3 rounded-lg border bg-card p-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
          <div className="relative w-full sm:w-72">
            <Search className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-3 size-4 text-muted-foreground" />
            <Input
              className="h-9 pl-9"
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t("searchPlaceholder", "Search name or IP")}
              value={searchInput}
            />
          </div>
          <Select onValueChange={setRegionFilter} value={regionFilter}>
            <SelectTrigger className="h-9 w-full sm:w-44">
              <SelectValue placeholder={t("filters.region", "All regions")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">
                {t("filters.allRegions", "All regions")}
              </SelectItem>
              {regionOptions.map((region) => (
                <SelectItem key={region} value={region}>
                  {region}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {hasActiveFilters ? (
            <Button
              className="w-fit text-muted-foreground"
              onClick={() => {
                setSearchInput("");
                setSearch("");
                setFleetFilter("all");
                setRegionFilter("all");
              }}
              size="sm"
              variant="ghost"
            >
              <X className="size-4" />
              {t("filters.clear", "Clear")}
            </Button>
          ) : null}
        </div>
        <div className="flex items-center justify-between gap-2 sm:justify-start lg:justify-end">
          <div className="grid h-9 flex-1 grid-cols-4 items-center rounded-lg border bg-background p-0.5 sm:flex-none">
            {(
              [
                ["all", t("filters.all", "All")],
                ["online", t("filters.online", "Online")],
                ["attention", t("filters.attention", "Issues")],
                ["pending", t("filters.pending", "Pending")],
              ] as const
            ).map(([value, label]) => (
              <Button
                className={cn(
                  "h-8 min-w-0 rounded-md px-3",
                  fleetFilter === value && "bg-primary text-primary-foreground"
                )}
                key={value}
                onClick={() => setFleetFilter(value)}
                size="sm"
                variant={fleetFilter === value ? "default" : "ghost"}
              >
                {label}
              </Button>
            ))}
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                aria-label={t("refresh", "Refresh")}
                onClick={refreshAll}
                size="icon"
                variant="outline"
              >
                <RefreshCcw />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("refresh", "Refresh")}</TooltipContent>
          </Tooltip>
        </div>
      </section>

      <div className="min-w-0">
        <ProTable<API.ServerListItem, Record<string, never>>
          action={tableRef}
          actions={{
            columnSize: 100,
            render: (row) => [
              <ServerRowActions
                key="actions"
                onDeleted={refreshAll}
                onEdit={() => setEditingServer(row)}
                onXray={() => setXrayTemplateServer(row)}
                referenced={
                  row.node_count > 0 ||
                  row.template_count > 0 ||
                  isServerReferencedByNodes(row.id)
                }
                row={row}
              />,
            ],
            batchRender(rows) {
              const hasReferencedServers = rows.some(
                (row) =>
                  row.node_count > 0 ||
                  row.template_count > 0 ||
                  isServerReferencedByNodes(row.id)
              );
              return [
                <ConfirmButton
                  cancelText={t("cancel", "Cancel")}
                  confirmText={t("confirm", "Confirm")}
                  description={t(
                    "confirmDeleteDesc",
                    "This action cannot be undone."
                  )}
                  key="delete"
                  onConfirm={async () => {
                    await Promise.all(
                      rows.map((row) => deleteServer({ id: row.id }))
                    );
                    toast.success(t("deleted", "Deleted"));
                    await refreshAll();
                  }}
                  title={t("confirmDeleteTitle", "Delete these servers?")}
                  trigger={
                    <Button
                      disabled={hasReferencedServers}
                      size="sm"
                      variant="destructive"
                    >
                      <Trash2 className="size-4" />
                      {t("delete", "Delete")}
                    </Button>
                  }
                />,
              ];
            },
          }}
          columns={[
            {
              id: "identity",
              size: 220,
              header: t("table.server", "Server"),
              cell: ({ row }) => (
                <IdentityCell
                  server={row.original}
                  status={getStatus(row.original)}
                />
              ),
            },
            {
              id: "health",
              size: 194,
              header: t("table.health", "Health"),
              cell: ({ row }) => (
                <HealthCell status={getStatus(row.original)} />
              ),
            },
            {
              id: "resources",
              size: 166,
              header: t("table.resources", "Resources"),
              cell: ({ row }) => (
                <ResourceCell status={getStatus(row.original)} />
              ),
            },
            {
              id: "traffic",
              size: 304,
              header: t("table.traffic", "Live traffic · 5 minute trend"),
              cell: ({ row }) => (
                <TrafficCell
                  points={networkRateHistory[row.original.id] || []}
                  status={getStatus(row.original)}
                />
              ),
            },
            {
              id: "activity",
              size: 142,
              header: t("table.activity", "Activity / Report"),
              cell: ({ row }) => (
                <ActivityCell
                  server={row.original}
                  status={getStatus(row.original)}
                />
              ),
            },
          ]}
          header={{ hidden: true }}
          request={async (pagination) => {
            const { data } = await filterServerList({
              page: pagination.page,
              size: pagination.size,
              search: search || undefined,
            });
            const result = normalizePageResult<API.ServerListItem>(data?.data);
            setPageServers(result.list);
            setFleetTotal(result.total);
            const serverIDs = result.list.map((server) => server.id);
            setHistoryServerIds((previous) =>
              previous.length === serverIDs.length &&
              previous.every((serverID, index) => serverID === serverIDs[index])
                ? previous
                : serverIDs
            );

            const list = result.list.filter((server) => {
              const status = getStatus(server);
              const location = [server.country, server.city]
                .filter(Boolean)
                .join(" / ");
              if (regionFilter !== "all" && location !== regionFilter) {
                return false;
              }
              if (fleetFilter === "online" && statusIsOffline(status)) {
                return false;
              }
              if (
                fleetFilter === "attention" &&
                !serverNeedsAttention(status)
              ) {
                return false;
              }
              if (
                fleetFilter === "pending" &&
                configHealth(status) !== "warning"
              ) {
                return false;
              }
              return true;
            });
            return {
              list,
              total:
                fleetFilter === "all" && regionFilter === "all"
                  ? result.total
                  : list.length,
            };
          }}
          texts={{ actions: t("actionsColumn", "Actions") }}
        />
      </div>

      <DynamicMultiplier
        onOpenChange={(open) => !open && setToolPanel(null)}
        open={toolPanel === "multiplier"}
        trigger={null}
      />
      <ServerConfig
        onOpenChange={(open) => !open && setToolPanel(null)}
        open={toolPanel === "config"}
        trigger={null}
      />
      {xrayTemplateServer ? (
        <ServerXrayTemplateBindForm
          onOpenChange={(open) => !open && setXrayTemplateServer(null)}
          open={Boolean(xrayTemplateServer)}
          server={xrayTemplateServer}
        />
      ) : null}
      {editingServer ? (
        <ServerForm
          initialValues={editingServer}
          loading={loading}
          onOpenChange={(open) => !open && setEditingServer(null)}
          onSubmit={async (values) => {
            setLoading(true);
            try {
              await updateServer({
                id: editingServer.id,
                ...(values as unknown as Omit<API.UpdateServerRequest, "id">),
              });
              toast.success(t("updated", "Updated"));
              await refreshAll();
              return true;
            } catch {
              return false;
            } finally {
              setLoading(false);
            }
          }}
          open={Boolean(editingServer)}
          title={t("drawerEditTitle", "Edit server")}
        />
      ) : null}
    </AdminPage>
  );
}
