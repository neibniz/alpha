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
import { Badge } from "@workspace/ui/components/badge";
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
import { Switch } from "@workspace/ui/components/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import {
  ProTable,
  type ProTableActions,
} from "@workspace/ui/composed/pro-table/pro-table";
import { cn } from "@workspace/ui/lib/utils";
import {
  createNode,
  deleteNode,
  resetSortWithNode,
  toggleNodeStatus,
  updateNode,
} from "@workspace/ui/services/admin/server";
import {
  AlertTriangle,
  Braces,
  Clock3,
  Copy,
  Globe2,
  MoreHorizontal,
  Pencil,
  Plus,
  Power,
  PowerOff,
  RefreshCcw,
  Search,
  Server as ServerIcon,
  Trash2,
  Waypoints,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { useNode } from "@/stores/node";
import { useServer } from "@/stores/server";
import {
  analyseNodes,
  buildNodeSummary,
  getInboundAlias,
  getInboundProtocol,
  getNodeReadiness,
  getServerHealth,
  type NodeAnalysis,
  type NodeFilter,
  type ServerHealth,
  sortNodes,
} from "./node-directory-data";
import NodeForm, { type NodeFormValues } from "./node-form";

function formatTimestamp(timestamp?: number) {
  if (!timestamp) return null;
  const value = timestamp < 1_000_000_000_000 ? timestamp * 1000 : timestamp;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return {
    date: new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(date),
    time: new Intl.DateTimeFormat(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    }).format(date),
  };
}

function NodeIdentity({
  analysis,
  node,
}: {
  analysis: NodeAnalysis;
  node: API.NodeListItem;
}) {
  const readiness = getNodeReadiness(node, analysis);
  return (
    <div className="flex min-w-[184px] items-start gap-2">
      <span
        aria-hidden
        className={cn(
          "mt-1.5 size-2 shrink-0 rounded-full",
          readiness === "disabled" && "bg-muted-foreground/50",
          readiness === "ready" && "bg-emerald-500",
          readiness === "attention" && "bg-amber-500"
        )}
      />
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-baseline gap-2">
          <p className="truncate font-semibold text-sm">{node.name || "—"}</p>
          <span className="shrink-0 text-[10px] text-muted-foreground tabular-nums">
            #{node.id}
          </span>
        </div>
        <div className="mt-2">
          <NodeTags tags={node.tags} />
        </div>
      </div>
    </div>
  );
}

function NodeTags({ tags }: { tags?: string[] }) {
  const uniqueTags = Array.from(new Set(tags || []));
  const visible = uniqueTags.slice(0, 2);
  if (visible.length === 0) {
    return <span className="text-[10px] text-muted-foreground">—</span>;
  }
  return (
    <div className="flex min-w-0 gap-1">
      {visible.map((tag) => (
        <Badge
          className="h-5 max-w-[78px] truncate px-1.5 font-normal text-[10px]"
          key={tag}
          variant="outline"
        >
          {tag}
        </Badge>
      ))}
      {uniqueTags.length > visible.length ? (
        <Badge className="h-5 px-1.5 text-[10px]" variant="secondary">
          +{uniqueTags.length - visible.length}
        </Badge>
      ) : null}
    </div>
  );
}

function NodeAccessCell({ node }: { node: API.NodeListItem }) {
  const { t } = useTranslation("nodes");
  const inbound = getInboundAlias(node);
  const endpointComplete = Boolean(
    node.address?.trim() && Number(node.port) > 0
  );
  return (
    <div className="min-w-[218px]">
      <p
        className={cn(
          "flex min-w-0 items-center gap-2 font-medium text-xs",
          !endpointComplete && "text-amber-500"
        )}
      >
        <Globe2 className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate font-mono">
          {endpointComplete
            ? `${node.address}:${node.port}`
            : t("endpointIncomplete", "Endpoint incomplete")}
        </span>
      </p>
      <div className="mt-2 flex min-w-0 items-center gap-2 text-[11px]">
        <Braces className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{inbound || "—"}</span>
        <span className="shrink-0 text-muted-foreground">
          {getInboundProtocol(node)}
        </span>
      </div>
    </div>
  );
}

function ServerCell({
  health,
  server,
  labels,
}: {
  health: ServerHealth;
  server?: API.Server;
  labels: Record<ServerHealth, string>;
}) {
  return (
    <div className="min-w-[190px]">
      <p className="flex min-w-0 items-center gap-2 truncate font-medium text-sm">
        <ServerIcon className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{server?.name || "—"}</span>
      </p>
      <p className="mt-1 truncate pl-5 text-[11px] text-muted-foreground tabular-nums">
        {server?.address || "—"}
      </p>
      <span
        className={cn(
          "mt-2 inline-flex items-center gap-1.5 pl-5 text-[10px]",
          health === "online" && "text-emerald-500",
          health === "offline" && "text-red-500",
          (health === "unknown" || health === "missing") &&
            "text-muted-foreground"
        )}
      >
        <span
          aria-hidden
          className={cn(
            "size-1.5 rounded-full bg-current",
            health === "unknown" && "opacity-60"
          )}
        />
        {labels[health]}
      </span>
    </div>
  );
}

function NodeReadinessCell({
  analysis,
  disabled,
  node,
  onToggle,
}: {
  analysis: NodeAnalysis;
  disabled: boolean;
  node: API.NodeListItem;
  onToggle: (enabled: boolean) => void;
}) {
  const { t } = useTranslation("nodes");
  const readiness = getNodeReadiness(node, analysis);
  const updatedAt = formatTimestamp(node.updated_at);
  const issues = [
    analysis.conflictingNodeIds.has(node.id)
      ? t("issues.endpointConflict", "Endpoint conflict")
      : "",
    analysis.offlineNodeIds.has(node.id)
      ? t("issues.serverOffline", "Linked server offline")
      : "",
    analysis.missingServerNodeIds.has(node.id)
      ? t("issues.serverMissing", "Linked server missing")
      : "",
    analysis.incompleteNodeIds.has(node.id)
      ? t("issues.incomplete", "Configuration incomplete")
      : "",
  ].filter(Boolean);
  const detail =
    issues.join(" · ") ||
    (readiness === "disabled"
      ? t("readiness.disabledDetail", "Node is currently disabled")
      : t("readiness.readyDetail", "Endpoint and binding are ready"));
  const label =
    readiness === "ready"
      ? t("readiness.ready", "Ready")
      : readiness === "disabled"
        ? t("readiness.disabled", "Disabled")
        : t("readiness.attention", "Needs attention");

  return (
    <div className="min-w-[232px]">
      <div className="flex items-center justify-between gap-3">
        <span
          className={cn(
            "inline-flex items-center gap-2 font-medium text-xs",
            readiness === "ready" && "text-emerald-500",
            readiness === "attention" && "text-amber-500",
            readiness === "disabled" && "text-muted-foreground"
          )}
        >
          <span className="size-1.5 rounded-full bg-current" />
          {label}
        </span>
        <Switch
          aria-label={t("toggleNode", "Toggle node status")}
          checked={node.enabled}
          className="data-[state=checked]:bg-emerald-500"
          disabled={disabled}
          onCheckedChange={onToggle}
        />
      </div>
      <p
        className="mt-2 truncate text-[10px] text-muted-foreground"
        title={detail}
      >
        {detail}
      </p>
      <p className="mt-2 flex items-center gap-1.5 text-[10px] text-muted-foreground tabular-nums">
        <Clock3 className="size-3" />
        {updatedAt
          ? `${updatedAt.date} · ${updatedAt.time}`
          : t("notAvailable", "Not available")}
      </p>
    </div>
  );
}

export default function Nodes() {
  const { t } = useTranslation("nodes");
  const tableRef = useRef<ProTableActions>(null);
  const [operationLoading, setOperationLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [nodeFilter, setNodeFilter] = useState<NodeFilter>("all");
  const [serverFilter, setServerFilter] = useState("all");
  const [inboundFilter, setInboundFilter] = useState("all");
  const [deleteTargets, setDeleteTargets] = useState<API.NodeListItem[]>([]);

  const { nodes, fetchNodes, fetchTags } = useNode();
  const { servers, fetchServers } = useServer();

  const serverById = useMemo(
    () => new Map(servers.map((server) => [server.id, server])),
    [servers]
  );
  const nodeList = nodes as API.NodeListItem[];
  const analysis = useMemo(
    () => analyseNodes(nodeList, serverById),
    [nodeList, serverById]
  );
  const summary = useMemo(
    () => buildNodeSummary(nodeList, analysis, serverById),
    [analysis, nodeList, serverById]
  );
  const inboundOptions = useMemo(
    () =>
      Array.from(new Set(nodeList.map(getInboundAlias).filter(Boolean))).sort(
        (a, b) => a.localeCompare(b)
      ),
    [nodeList]
  );
  const hasActiveFilters =
    Boolean(searchInput) ||
    nodeFilter !== "all" ||
    serverFilter !== "all" ||
    inboundFilter !== "all";

  useEffect(() => {
    const timer = window.setTimeout(() => setSearch(searchInput.trim()), 250);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  const filteredNodes = useMemo(() => {
    const query = search.toLocaleLowerCase();
    return sortNodes(
      nodeList.filter((node) => {
        const server = serverById.get(node.server_id);
        const matchesSearch =
          !query ||
          [
            node.name,
            node.address,
            String(node.port || ""),
            getInboundAlias(node),
            server?.name,
            server?.address,
            ...(node.tags || []),
          ].some((value) => value?.toLocaleLowerCase().includes(query));
        const matchesState =
          nodeFilter === "all" ||
          (nodeFilter === "enabled" && node.enabled) ||
          (nodeFilter === "disabled" && !node.enabled) ||
          (nodeFilter === "attention" &&
            analysis.attentionNodeIds.has(node.id));
        const matchesServer =
          serverFilter === "all" || String(node.server_id) === serverFilter;
        const matchesInbound =
          inboundFilter === "all" || getInboundAlias(node) === inboundFilter;
        return matchesSearch && matchesState && matchesServer && matchesInbound;
      })
    );
  }, [
    analysis.attentionNodeIds,
    inboundFilter,
    nodeFilter,
    nodeList,
    search,
    serverById,
    serverFilter,
  ]);

  useEffect(() => {
    const timer = window.setTimeout(() => tableRef.current?.refresh(), 0);
    return () => window.clearTimeout(timer);
  }, [filteredNodes]);

  const refreshData = useCallback(async () => {
    await Promise.all([fetchNodes(), fetchTags(), fetchServers()]);
    tableRef.current?.refresh();
  }, [fetchNodes, fetchServers, fetchTags]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await refreshData();
    } finally {
      setRefreshing(false);
    }
  }, [refreshData]);

  const setNodesEnabled = useCallback(
    async (targets: API.NodeListItem[], enabled: boolean) => {
      if (targets.length === 0) return;
      setOperationLoading(true);
      try {
        await Promise.all(
          targets.map((node) =>
            toggleNodeStatus({ id: node.id, enable: enabled })
          )
        );
        toast.success(
          enabled
            ? t("batchEnabled", "Selected nodes enabled")
            : t("batchDisabled", "Selected nodes disabled")
        );
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const copyExistingNode = useCallback(
    async (node: API.NodeListItem) => {
      const {
        id: _id,
        sort: _sort,
        enabled: _enabled,
        updated_at: _updatedAt,
        created_at: _createdAt,
        ...rest
      } = node;
      setOperationLoading(true);
      try {
        await createNode({ ...rest, enabled: false });
        toast.success(t("copied", "Copied"));
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const handleDelete = useCallback(async () => {
    if (deleteTargets.length === 0) return;
    setOperationLoading(true);
    try {
      await Promise.all(
        deleteTargets.map((node) => deleteNode({ id: node.id }))
      );
      toast.success(t("deleted", "Deleted"));
      setDeleteTargets([]);
      await refreshData();
    } finally {
      setOperationLoading(false);
    }
  }, [deleteTargets, refreshData, t]);

  const submitCreate = useCallback(
    async (values: NodeFormValues) => {
      setOperationLoading(true);
      try {
        await createNode({
          name: values.name,
          server_id: Number(values.server_id),
          inbound_alias: values.inbound_alias,
          protocol: values.inbound_alias,
          address: values.address,
          port: Number(values.port),
          tags: values.tags || [],
          enabled: false,
        });
        toast.success(t("created", "Created"));
        await refreshData();
        return true;
      } catch {
        return false;
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const submitUpdate = useCallback(
    async (node: API.NodeListItem, values: NodeFormValues) => {
      setOperationLoading(true);
      try {
        await updateNode({
          ...values,
          id: node.id,
          enabled: node.enabled,
          server_id: values.server_id ?? node.server_id,
        });
        toast.success(t("updated", "Updated"));
        await refreshData();
        return true;
      } catch {
        return false;
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const serverHealthLabels: Record<ServerHealth, string> = {
    online: t("serverOnline", "Online"),
    offline: t("serverOffline", "Offline"),
    unknown: t("serverUnknown", "Unknown"),
    missing: t("serverMissing", "Server missing"),
  };

  return (
    <AdminPage className="pb-6">
      <AdminPageHeader
        actions={
          <NodeForm
            loading={operationLoading}
            onSubmit={submitCreate}
            title={t("drawerCreateTitle", "Create node")}
            trigger={
              <Button>
                <Plus data-icon="inline-start" />
                {t("createNode", "Create node")}
              </Button>
            }
          />
        }
        description={t(
          "pageDescription",
          "Manage subscription delivery endpoints, inbound bindings, and server placement."
        )}
        eyebrow={t("workspace", "Infrastructure")}
        title={t("pageTitle", "Node management")}
      />

      <ToggleGroup
        className="admin-stat-strip grid w-full grid-cols-2 overflow-hidden rounded-lg border bg-card lg:grid-cols-4"
        onValueChange={(value) => {
          if (value) setNodeFilter(value as NodeFilter);
        }}
        type="single"
        value={nodeFilter}
      >
        {[
          {
            detail: t("summary.totalDetail", "Across {{count}} servers", {
              count: summary.servers,
            }),
            filter: "all" as const,
            icon: Waypoints,
            label: t("summary.total", "Nodes"),
            tone: "text-foreground",
            value: summary.total,
          },
          {
            detail: t("summary.enabledDetail", "Available for delivery"),
            filter: "enabled" as const,
            icon: Power,
            label: t("summary.enabled", "Enabled"),
            tone: "text-emerald-500",
            value: summary.enabled,
          },
          {
            detail: t("summary.disabledDetail", "Excluded from delivery"),
            filter: "disabled" as const,
            icon: PowerOff,
            label: t("summary.disabled", "Disabled"),
            tone: "text-muted-foreground",
            value: summary.disabled,
          },
          {
            detail: t("summary.attentionDetail", "Conflicts or invalid setup"),
            filter: "attention" as const,
            icon: AlertTriangle,
            label: t("summary.attention", "Needs attention"),
            tone: "text-amber-500",
            value: summary.attention,
          },
        ].map((item) => {
          const Icon = item.icon;
          const active = nodeFilter === item.filter;
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

      {analysis.attentionNodeIds.size > 0 ? (
        <Alert>
          <AlertTriangle />
          <AlertTitle>
            {t("attentionCount", "{{count}} nodes need attention", {
              count: analysis.attentionNodeIds.size,
            })}
          </AlertTitle>
          <AlertDescription className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs">
              {analysis.conflictingNodeIds.size > 0 ? (
                <span>
                  {t("endpointConflicts", "{{count}} conflict nodes", {
                    count: analysis.conflictingNodeIds.size,
                  })}
                </span>
              ) : null}
              {analysis.offlineServerIds.size > 0 ? (
                <span>
                  {t("serversOffline", "{{count}} linked servers offline", {
                    count: analysis.offlineServerIds.size,
                  })}
                </span>
              ) : null}
              {analysis.incompleteNodeIds.size > 0 ? (
                <span>
                  {t("incompleteNodes", "{{count}} incomplete", {
                    count: analysis.incompleteNodeIds.size,
                  })}
                </span>
              ) : null}
              {analysis.missingServerNodeIds.size > 0 ? (
                <span>
                  {t("serversMissing", "{{count}} linked servers missing", {
                    count: analysis.missingServerNodeIds.size,
                  })}
                </span>
              ) : null}
            </div>
            <Button
              className="w-full shrink-0 sm:w-auto"
              onClick={() => setNodeFilter("attention")}
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
              placeholder={t(
                "searchPlaceholder",
                "Search name, address, or tag"
              )}
              value={searchInput}
            />
          </div>
          <Select onValueChange={setServerFilter} value={serverFilter}>
            <SelectTrigger className="h-9 w-full sm:w-44">
              <SelectValue
                placeholder={t("filters.allServers", "All servers")}
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">
                {t("filters.allServers", "All servers")}
              </SelectItem>
              {servers.map((server) => (
                <SelectItem key={server.id} value={String(server.id)}>
                  {server.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select onValueChange={setInboundFilter} value={inboundFilter}>
            <SelectTrigger className="h-9 w-full sm:w-44">
              <SelectValue
                placeholder={t("filters.allInbounds", "All Inbounds")}
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">
                {t("filters.allInbounds", "All Inbounds")}
              </SelectItem>
              {inboundOptions.map((inbound) => (
                <SelectItem key={inbound} value={inbound}>
                  {inbound}
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
                setNodeFilter("all");
                setServerFilter("all");
                setInboundFilter("all");
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
                ["enabled", t("filters.enabled", "Enabled")],
                ["disabled", t("filters.disabled", "Disabled")],
                ["attention", t("filters.attention", "Issues")],
              ] as const
            ).map(([value, label]) => (
              <Button
                className={cn(
                  "h-8 min-w-0 rounded-md px-3",
                  nodeFilter === value && "bg-primary text-primary-foreground"
                )}
                key={value}
                onClick={() => setNodeFilter(value)}
                size="sm"
                variant={nodeFilter === value ? "default" : "ghost"}
              >
                {label}
              </Button>
            ))}
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                aria-label={t("refresh", "Refresh")}
                disabled={refreshing}
                onClick={handleRefresh}
                size="icon"
                variant="outline"
              >
                <RefreshCcw
                  className={cn("size-4", refreshing && "animate-spin")}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("refresh", "Refresh")}</TooltipContent>
          </Tooltip>
        </div>
      </section>

      <div className="min-w-0">
        <ProTable<API.NodeListItem, Record<string, never>>
          action={tableRef}
          actions={{
            columnSize: 82,
            render: (node) => [
              <NodeForm
                initialValues={node}
                key="edit"
                loading={operationLoading}
                onSubmit={(values) => submitUpdate(node, values)}
                title={t("drawerEditTitle", "Edit node")}
                trigger={
                  <Button
                    aria-label={t("edit", "Edit")}
                    size="icon-sm"
                    variant="ghost"
                  >
                    <Pencil className="size-4" />
                  </Button>
                }
              />,
              <DropdownMenu key="more">
                <DropdownMenuTrigger asChild>
                  <Button
                    aria-label={t("moreActions", "More actions")}
                    size="icon-sm"
                    variant="ghost"
                  >
                    <MoreHorizontal className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    disabled={operationLoading}
                    onSelect={() => copyExistingNode(node)}
                  >
                    <Copy className="size-4" />
                    {t("copy", "Copy")}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onClick={() => setDeleteTargets([node])}
                  >
                    <Trash2 className="size-4" />
                    {t("delete", "Delete")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>,
            ],
            batchRender: (rows) => [
              <Button
                disabled={operationLoading}
                key="enable"
                onClick={() => setNodesEnabled(rows, true)}
                size="sm"
                variant="outline"
              >
                <Power className="size-4" />
                {t("batchEnable", "Enable selected")}
              </Button>,
              <Button
                disabled={operationLoading}
                key="disable"
                onClick={() => setNodesEnabled(rows, false)}
                size="sm"
                variant="outline"
              >
                <PowerOff className="size-4" />
                {t("batchDisable", "Disable selected")}
              </Button>,
              <Button
                disabled={operationLoading}
                key="delete"
                onClick={() => setDeleteTargets(rows)}
                size="sm"
                variant="destructive"
              >
                <Trash2 className="size-4" />
                {t("delete", "Delete")}
              </Button>,
            ],
          }}
          columns={[
            {
              id: "identity",
              size: 210,
              header: t("table.node", "Node"),
              cell: ({ row }) => (
                <NodeIdentity analysis={analysis} node={row.original} />
              ),
            },
            {
              id: "access",
              size: 245,
              header: t("table.access", "Access method"),
              cell: ({ row }) => <NodeAccessCell node={row.original} />,
            },
            {
              id: "server",
              size: 220,
              header: t("table.placement", "Server placement"),
              cell: ({ row }) => {
                const server = serverById.get(row.original.server_id);
                return (
                  <ServerCell
                    health={getServerHealth(server)}
                    labels={serverHealthLabels}
                    server={server}
                  />
                );
              },
            },
            {
              id: "readiness",
              size: 260,
              header: t("table.readiness", "Readiness"),
              cell: ({ row }) => (
                <NodeReadinessCell
                  analysis={analysis}
                  disabled={operationLoading}
                  node={row.original}
                  onToggle={(enabled) =>
                    setNodesEnabled([row.original], enabled)
                  }
                />
              ),
            },
          ]}
          header={{ hidden: true }}
          onSort={async (source, target, items) => {
            const sourceIndex = items.findIndex(
              (item) => String(item.id) === source
            );
            const targetIndex = items.findIndex(
              (item) => String(item.id) === target
            );
            if (sourceIndex === -1 || targetIndex === -1) return items;

            const next = items.slice();
            const [movedItem] = next.splice(sourceIndex, 1);
            if (!movedItem) return items;
            next.splice(targetIndex, 0, movedItem);

            const currentSorts = items
              .map((item) => item.sort)
              .filter((value): value is number => Number.isFinite(value));
            const baseSort = currentSorts.length
              ? Math.min(...currentSorts)
              : 0;
            const previousSort = new Map(
              items.map((item) => [item.id, item.sort])
            );
            const updatedItems = next.map((item, index) => ({
              ...item,
              sort: baseSort + index,
            }));
            const changedItems = updatedItems.filter(
              (item) => item.sort !== previousSort.get(item.id)
            );

            if (changedItems.length > 0) {
              await resetSortWithNode({
                sort: changedItems.map((item) => ({
                  id: item.id,
                  sort: item.sort,
                })) as API.SortItem[],
              });
              toast.success(t("sorted_success", "Sorted successfully"));
              await refreshData();
            }
            return updatedItems;
          }}
          request={async (pagination) => {
            const start = (pagination.page - 1) * pagination.size;
            return {
              list: filteredNodes.slice(start, start + pagination.size),
              total: filteredNodes.length,
            };
          }}
          texts={{
            actions: t("table.actions", "Actions"),
            selectedRowsText: (total) =>
              t("selectedRows", "{{count}} nodes selected", { count: total }),
          }}
        />
      </div>

      <AlertDialog
        onOpenChange={(open) => {
          if (!open) setDeleteTargets([]);
        }}
        open={deleteTargets.length > 0}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {deleteTargets.length > 1
                ? t("confirmBatchDeleteTitle", "Delete selected nodes?")
                : t("confirmDeleteTitle", "Delete this node?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("confirmDeleteDesc", "This action cannot be undone.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel", "Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              disabled={operationLoading}
              onClick={handleDelete}
            >
              {t("confirm", "Confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AdminPage>
  );
}
