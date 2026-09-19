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
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
} from "@workspace/ui/components/item";
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
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import { cn } from "@workspace/ui/lib/utils";
import {
  createXrayTemplate,
  deleteXrayTemplate,
  filterXrayTemplateList,
  getXrayTemplateDetail,
  updateXrayTemplate,
} from "@workspace/ui/services/admin/server";
import { fetchAllPages } from "@workspace/ui/utils/pagination";
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  Download,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Trash2,
  Upload,
} from "lucide-react";
import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import XrayTemplateForm from "./template-form";
import {
  buildXrayTemplateSummary,
  filterXrayTemplates,
  getXrayTemplateCapabilities,
  getXrayTemplateUseState,
  normalizeImportedXrayTemplates,
  type XrayTemplateBindingFilter,
  type XrayTemplateStateFilter,
  type XrayTemplateTypeFilter,
  xrayTemplateDetailToCreateRequest,
} from "./list-data";

const MAX_IMPORT_BYTES = 2 * 1024 * 1024;

const DEFAULT_DESCRIPTION_KEYS: Record<string, string> = {
  "Default VLESS Reality inbound. Replace the Reality key pair and camouflage target before production.":
    "vlessReality",
  "VLESS Reality 入站模板，生产环境请替换密钥和伪装目标。": "vlessReality",
  "Default VLESS Reality gRPC inbound. Replace the Reality key pair and camouflage target before production.":
    "vlessRealityGrpc",
  "VLESS Reality gRPC 入站模板，生产环境请替换密钥和伪装目标。":
    "vlessRealityGrpc",
  "Default VLESS Reality XHTTP inbound. XHTTP extra sets XMUX maxConnections to 1 so clients reuse one TCP connection for transport.":
    "vlessRealityXhttp",
  "VLESS Reality XHTTP 入站模板，XHTTP extra 已限制 XMUX maxConnections 为 1，客户端仅复用一条 TCP 连接。":
    "vlessRealityXhttp",
  "Default Hysteria2 inbound. xray-agent injects real users into settings.users with auth, level and email.":
    "hysteria2",
  "Default Hysteria2 inbound. xray-agent injects real users into settings.clients with auth, level and email.":
    "hysteria2",
  "Hysteria2 入站模板，xray-agent 会将真实用户注入 settings.users。":
    "hysteria2",
  "Hysteria2 入站模板，xray-agent 会将真实用户注入 settings.clients。":
    "hysteria2",
  "Default DNS template.": "dns",
  "DNS 模板。": "dns",
  "Default blackhole outbound.": "blackhole",
  "阻断出站模板。": "blackhole",
  "Default freedom outbound.": "freedom",
  "直连出站模板。": "freedom",
  "Default safety routing template.": "safetyRouting",
  "安全路由模板。": "safetyRouting",
  "Default routing template.": "routing",
  "路由模板。": "routing",
  "Default Xray geodata auto update template. Xray writes into its asset directory, so geoip.dat and geosite.dat must already exist in the image or host asset path.":
    "geodata",
  "Xray 地理数据模板，用于声明 geoip.dat 与 geosite.dat 的自动更新信息。":
    "geodata",
};

const TYPE_FILTERS: XrayTemplateTypeFilter[] = [
  "all",
  "inbound",
  "outbound",
  "dns",
  "routing",
  "geodata",
];

function formatTimestamp(timestamp?: number) {
  if (!timestamp) return;
  const value = timestamp < 1_000_000_000_000 ? timestamp * 1000 : timestamp;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return;
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

function safeFilename(value: string) {
  const normalized = value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, "-")
    .replace(/\s+/g, "-");
  return normalized || "xray-template";
}

export default function XrayTemplates() {
  const { t } = useTranslation("xray-templates");
  const tableRef = useRef<ProTableActions>(null);
  const importInputRef = useRef<HTMLInputElement>(null);
  const [templates, setTemplates] = useState<API.XrayTemplateListItem[]>([]);
  const [operationLoading, setOperationLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<XrayTemplateTypeFilter>("all");
  const [stateFilter, setStateFilter] =
    useState<XrayTemplateStateFilter>("all");
  const [bindingFilter, setBindingFilter] =
    useState<XrayTemplateBindingFilter>("all");
  const [deleteTarget, setDeleteTarget] = useState<API.XrayTemplateListItem>();

  const getDescription = useCallback(
    (description?: string) => {
      if (!description) return t("empty.description", "No description");
      const key = DEFAULT_DESCRIPTION_KEYS[description];
      return key ? t(`defaultDescriptions.${key}`, description) : description;
    },
    [t]
  );

  const loadTemplates = useCallback(async () => {
    const list = await fetchAllPages<API.XrayTemplateListItem>(
      ({ page, size }) => filterXrayTemplateList({ page, size })
    );
    setTemplates(list);
  }, []);

  useEffect(() => {
    loadTemplates().catch(() => undefined);
  }, [loadTemplates]);

  useEffect(() => {
    const timer = window.setTimeout(() => setSearch(searchInput.trim()), 250);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  const summary = useMemo(
    () => buildXrayTemplateSummary(templates),
    [templates]
  );
  const filteredTemplates = useMemo(
    () =>
      filterXrayTemplates(templates, {
        search,
        type: typeFilter,
        state: stateFilter,
        binding: bindingFilter,
      }),
    [bindingFilter, search, stateFilter, templates, typeFilter]
  );

  useEffect(() => {
    const timer = window.setTimeout(() => tableRef.current?.refresh(), 0);
    return () => window.clearTimeout(timer);
  }, [filteredTemplates]);

  const refreshData = useCallback(async () => {
    await loadTemplates();
    tableRef.current?.refresh();
  }, [loadTemplates]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await refreshData();
    } finally {
      setRefreshing(false);
    }
  }, [refreshData]);

  const loadTemplateDetail = useCallback(
    async (template: API.XrayTemplateListItem) => {
      try {
        const { data } = await getXrayTemplateDetail({ id: template.id });
        return data.data;
      } catch {
        toast.error(
          t("message.detailLoadFailed", "Failed to load template details")
        );
      }
    },
    [t]
  );

  const createTemplate = useCallback(
    async (values: API.CreateXrayTemplateRequest) => {
      setOperationLoading(true);
      try {
        await createXrayTemplate(values);
        toast.success(t("message.created", "Template created"));
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

  const updateTemplate = useCallback(
    async (
      template: API.XrayTemplateListItem,
      values: Omit<API.UpdateXrayTemplateRequest, "id">
    ) => {
      setOperationLoading(true);
      try {
        await updateXrayTemplate({ id: template.id, ...values });
        toast.success(t("message.updated", "Template updated"));
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

  const toggleTemplate = useCallback(
    async (template: API.XrayTemplateListItem, enabled: boolean) => {
      setOperationLoading(true);
      try {
        await updateXrayTemplate({ id: template.id, enabled });
        toast.success(
          enabled
            ? t("message.enabled", "Template enabled")
            : t("message.disabled", "Template disabled")
        );
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const duplicateTemplate = useCallback(
    async (template: API.XrayTemplateListItem) => {
      setOperationLoading(true);
      try {
        const detail = await loadTemplateDetail(template);
        if (!detail) return;
        await createXrayTemplate(
          xrayTemplateDetailToCreateRequest(detail, {
            name: t("copyName", "{{name}} copy", { name: detail.name }),
            enabled: false,
          })
        );
        toast.success(t("message.copied", "Template copied"));
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [loadTemplateDetail, refreshData, t]
  );

  const exportTemplate = useCallback(
    async (template: API.XrayTemplateListItem) => {
      const detail = await loadTemplateDetail(template);
      if (!detail) return;
      const payload = xrayTemplateDetailToCreateRequest(detail);
      const blob = new Blob([JSON.stringify(payload, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `${safeFilename(detail.name)}.json`;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success(t("message.exported", "Template exported"));
    },
    [loadTemplateDetail, t]
  );

  const importTemplates = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      event.target.value = "";
      if (!file) return;
      if (file.size > MAX_IMPORT_BYTES) {
        toast.error(
          t("message.importTooLarge", "Import file must be under 2 MB")
        );
        return;
      }

      setOperationLoading(true);
      try {
        const payloads = normalizeImportedXrayTemplates(
          JSON.parse(await file.text())
        );
        for (const payload of payloads) await createXrayTemplate(payload);
        toast.success(
          t("message.imported", "{{count}} templates imported", {
            count: payloads.length,
          })
        );
        await refreshData();
      } catch (error) {
        toast.error(
          error instanceof Error
            ? error.message
            : t("message.importFailed", "Template import failed")
        );
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setOperationLoading(true);
    try {
      await deleteXrayTemplate({ id: deleteTarget.id });
      toast.success(t("message.deleted", "Template deleted"));
      setDeleteTarget(undefined);
      await refreshData();
    } finally {
      setOperationLoading(false);
    }
  }, [deleteTarget, refreshData, t]);

  const summaryItems = [
    {
      label: t("summary.total", "Templates"),
      value: summary.total,
      tone: "text-foreground",
    },
    {
      label: t("summary.enabled", "Enabled"),
      value: summary.enabled,
      tone: "text-emerald-500",
    },
    {
      label: t("summary.attention", "Needs attention"),
      value: summary.attention,
      tone: summary.attention > 0 ? "text-amber-500" : "text-foreground",
    },
    {
      label: t("summary.bindings", "Server bindings"),
      value: summary.bindings,
      tone: "text-foreground",
    },
    {
      label: t("summary.subscriptions", "Subscription bindings"),
      value: summary.subscriptions,
      tone: "text-foreground",
    },
  ];

  return (
    <AdminPage>
      <AdminPageHeader
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Input
              accept="application/json,.json"
              className="hidden"
              onChange={importTemplates}
              ref={importInputRef}
              type="file"
            />
            <Button
              className="gap-2"
              disabled={operationLoading}
              onClick={() => importInputRef.current?.click()}
              variant="outline"
            >
              <Upload className="size-4" />
              {t("action.import", "Import")}
            </Button>
            <XrayTemplateForm
              loading={operationLoading}
              onSubmit={createTemplate}
              title={t("form.createTitle", "Create Xray template")}
              trigger={
                <Button className="gap-2">
                  <Plus className="size-4" />
                  {t("action.create", "Create template")}
                </Button>
              }
            />
          </div>
        }
        description={t(
          "pageDescription",
          "Manage reusable configurations, activation state, and server usage."
        )}
        eyebrow={t("workspace", "Infrastructure")}
        title={t("pageTitle", "Xray templates")}
      />

      <ItemGroup className="admin-stat-strip xray-template-stats grid grid-cols-2 overflow-hidden rounded-lg border bg-card md:grid-cols-5">
        {summaryItems.map((item) => (
          <Item
            className="xray-template-stat rounded-none border-0 border-border/70 px-5 py-4 last:col-span-2 md:last:col-span-1"
            key={item.label}
            size="sm"
          >
            <ItemContent>
              <ItemDescription>{item.label}</ItemDescription>
              <ItemTitle
                className={cn("font-semibold text-2xl tabular-nums", item.tone)}
              >
                {item.value}
              </ItemTitle>
            </ItemContent>
          </Item>
        ))}
      </ItemGroup>

      {summary.attention > 0 ? (
        <Alert>
          <AlertTriangle />
          <AlertTitle>
            {t(
              "attentionTitle",
              "{{count}} enabled templates are not bound to a server",
              { count: summary.attention }
            )}
          </AlertTitle>
          <AlertDescription className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <span>
              {t(
                "attentionDescription",
                "They will not take effect until assigned in server management."
              )}
            </span>
            <Button
              className="shrink-0"
              onClick={() => {
                setStateFilter("enabled");
                setBindingFilter("unbound");
              }}
              size="sm"
              variant="outline"
            >
              {t("action.viewAttention", "View unbound")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      <section className="space-y-3">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div className="relative w-full xl:w-72">
            <Search className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-3 size-4 text-muted-foreground" />
            <Input
              className="h-9 pl-9"
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t(
                "filter.searchPlaceholder",
                "Search name, protocol, or ID"
              )}
              value={searchInput}
            />
          </div>
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
            <div className="inline-flex h-9 max-w-full items-center overflow-x-auto rounded-lg border bg-card p-0.5">
              {TYPE_FILTERS.map((value) => (
                <Button
                  className={cn(
                    "h-8 shrink-0 rounded-md px-3",
                    typeFilter === value && "bg-primary text-primary-foreground"
                  )}
                  key={value}
                  onClick={() => setTypeFilter(value)}
                  size="sm"
                  variant="ghost"
                >
                  {value === "all"
                    ? t("filter.allTypes", "All")
                    : t(`type.${value}`, value)}
                </Button>
              ))}
            </div>
            <Select
              onValueChange={(value) =>
                setStateFilter(value as XrayTemplateStateFilter)
              }
              value={stateFilter}
            >
              <SelectTrigger className="h-9 w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t("filter.allStates", "All states")}
                </SelectItem>
                <SelectItem value="enabled">
                  {t("state.enabled", "Enabled")}
                </SelectItem>
                <SelectItem value="disabled">
                  {t("state.disabled", "Disabled")}
                </SelectItem>
              </SelectContent>
            </Select>
            <Select
              onValueChange={(value) =>
                setBindingFilter(value as XrayTemplateBindingFilter)
              }
              value={bindingFilter}
            >
              <SelectTrigger className="h-9 w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t("filter.allBindings", "All usage")}
                </SelectItem>
                <SelectItem value="bound">
                  {t("filter.bound", "Bound")}
                </SelectItem>
                <SelectItem value="unbound">
                  {t("filter.unbound", "Unbound")}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                aria-label={t("action.refresh", "Refresh")}
                className="shrink-0"
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
            <TooltipContent>{t("action.refresh", "Refresh")}</TooltipContent>
          </Tooltip>
        </div>
      </section>

      <ProTable<API.XrayTemplateListItem, Record<string, never>>
        action={tableRef}
        actions={{
          columnSize: 116,
          render: (template) => [
            <XrayTemplateForm
              initialValues={template}
              key="edit"
              loadInitialValues={async () => loadTemplateDetail(template)}
              loading={operationLoading}
              onSubmit={(values) => updateTemplate(template, values)}
              title={t("form.editTitle", "Edit Xray template")}
              trigger={
                <Button
                  aria-label={t("action.edit", "Edit")}
                  size="icon-sm"
                  variant="ghost"
                >
                  <Pencil className="size-4" />
                </Button>
              }
            />,
            <Button
              aria-label={t("action.copy", "Copy")}
              disabled={operationLoading}
              key="copy"
              onClick={() => duplicateTemplate(template)}
              size="icon-sm"
              variant="ghost"
            >
              <Copy className="size-4" />
            </Button>,
            <DropdownMenu key="more">
              <DropdownMenuTrigger asChild>
                <Button
                  aria-label={t("action.more", "More actions")}
                  size="icon-sm"
                  variant="ghost"
                >
                  <MoreHorizontal className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => exportTemplate(template)}>
                  <Download className="size-4" />
                  {t("action.export", "Export JSON")}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() => setDeleteTarget(template)}
                >
                  <Trash2 className="size-4" />
                  {t("action.delete", "Delete")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>,
          ],
        }}
        columns={[
          {
            id: "identity",
            size: 235,
            header: t("column.template", "Template"),
            cell: ({ row }) => (
              <div className="min-w-0">
                <p className="truncate font-medium text-sm">
                  {row.original.name || "—"}
                </p>
                <p className="truncate text-muted-foreground text-xs">
                  tpl_{String(row.original.id).padStart(4, "0")} ·{" "}
                  {getDescription(row.original.description)}
                </p>
              </div>
            ),
          },
          {
            accessorKey: "type",
            size: 82,
            header: t("column.type", "Type"),
            cell: ({ row }) => (
              <Badge className="font-normal" variant="outline">
                {t(`type.${row.original.type}`, row.original.type)}
              </Badge>
            ),
          },
          {
            id: "capabilities",
            size: 180,
            header: t("column.capabilities", "Protocol / capability"),
            cell: ({ row }) => (
              <div className="flex min-w-0 flex-wrap gap-1">
                {getXrayTemplateCapabilities(row.original).map((capability) => (
                  <Badge
                    className="border-primary/20 bg-primary/[0.06] font-normal text-primary"
                    key={capability}
                    variant="outline"
                  >
                    {capability}
                  </Badge>
                ))}
              </div>
            ),
          },
          {
            id: "enabled",
            size: 70,
            header: t("column.enabled", "Enabled"),
            cell: ({ row }) => (
              <Switch
                aria-label={t("action.toggle", "Toggle template")}
                checked={row.original.enabled}
                className="data-[state=checked]:bg-emerald-500"
                disabled={operationLoading}
                onCheckedChange={(enabled) =>
                  toggleTemplate(row.original, enabled)
                }
              />
            ),
          },
          {
            id: "useState",
            size: 112,
            header: t("column.useState", "Usage state"),
            cell: ({ row }) => {
              const state = getXrayTemplateUseState(row.original);
              return (
                <span
                  className={cn(
                    "inline-flex items-center gap-1.5 text-xs",
                    state === "ready" && "text-emerald-500",
                    state === "unused" && "text-amber-500",
                    state === "disabled" && "text-muted-foreground"
                  )}
                >
                  {state === "ready" ? (
                    <CheckCircle2 className="size-3.5" />
                  ) : state === "unused" ? (
                    <AlertTriangle className="size-3.5" />
                  ) : (
                    <span className="size-1.5 rounded-full bg-current" />
                  )}
                  {t(`useState.${state}`, state)}
                </span>
              );
            },
          },
          {
            accessorKey: "binding_count",
            size: 92,
            header: t("column.bindings", "Servers"),
            cell: ({ row }) => (
              <div className="text-xs tabular-nums">
                <p className="font-medium text-sm">
                  {row.original.binding_count || 0}
                </p>
                <p className="text-muted-foreground">
                  {t("unit.servers", "servers")}
                </p>
              </div>
            ),
          },
          {
            accessorKey: "subscription_binding_count",
            size: 90,
            header: t("column.subscriptions", "Subscription"),
            cell: ({ row }) => (
              <div className="text-xs tabular-nums">
                <p className="font-medium text-sm">
                  {row.original.subscription_binding_count || 0}
                </p>
                <p className="text-muted-foreground">
                  {t("unit.entries", "entries")}
                </p>
              </div>
            ),
          },
          {
            accessorKey: "updated_at",
            size: 105,
            header: t("column.updatedAt", "Updated"),
            cell: ({ row }) => {
              const updatedAt = formatTimestamp(row.original.updated_at);
              return updatedAt ? (
                <div className="text-xs tabular-nums">
                  <p>{updatedAt.date}</p>
                  <p className="text-muted-foreground">{updatedAt.time}</p>
                </div>
              ) : (
                <span className="text-muted-foreground">—</span>
              );
            },
          },
        ]}
        header={{ hidden: true }}
        request={async (pagination) => {
          const start = (pagination.page - 1) * pagination.size;
          return {
            list: filteredTemplates.slice(start, start + pagination.size),
            total: filteredTemplates.length,
          };
        }}
        texts={{ actions: t("column.actions", "Actions") }}
      />

      <AlertDialog
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(undefined);
        }}
        open={Boolean(deleteTarget)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("message.deleteTitle", "Delete this template?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "message.deleteDescription",
                "This cannot be undone. Remove server bindings before deleting a template in use."
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {t("action.cancel", "Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={operationLoading}
              onClick={handleDelete}
            >
              {t("action.confirm", "Confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AdminPage>
  );
}
