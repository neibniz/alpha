"use client";

import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import type { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import { Input } from "@workspace/ui/components/input";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { Switch } from "@workspace/ui/components/switch";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Textarea } from "@workspace/ui/components/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { ConfirmButton } from "@/components/confirm-button";
import { GoTemplateEditor } from "@workspace/ui/composed/editor/index";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import { UploadImage } from "@workspace/ui/composed/upload-image";
import {
  createSubscribeApplication,
  deleteSubscribeApplication,
  getSubscribeApplication,
  getSubscribeApplicationList,
  updateSubscribeApplication,
} from "@workspace/ui/services/admin/application";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import {
  AppWindow,
  ExternalLink,
  Github,
  Pencil,
  Plus,
  Star,
  Trash2,
} from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { type FieldErrors, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { DiscardChangesDialog } from "../../components/discard-changes-dialog";
import { createSubscribeClientFormValues } from "./client-form-data";
import { subscribeSchema } from "./schema";
import { TemplatePreview } from "./template-preview";

type SubscribeTranslator = (key: string, defaultValue: string) => string;

const OUTPUT_FORMATS = ["base64", "yaml", "json", "conf", "plain"];

function optionalHttpUrl(message: string) {
  return z
    .string()
    .trim()
    .refine((value) => {
      if (!value) return true;
      try {
        const url = new URL(value);
        return url.protocol === "http:" || url.protocol === "https:";
      } catch {
        return false;
      }
    }, message);
}

function optionalImageSource(message: string) {
  return z
    .string()
    .trim()
    .refine((value) => {
      if (!value) return true;
      if (value.startsWith("data:image/")) return true;
      if (value.startsWith("/") && !value.startsWith("//")) return true;
      try {
        const url = new URL(value);
        return url.protocol === "http:" || url.protocol === "https:";
      } catch {
        return false;
      }
    }, message);
}

const createClientFormSchema = (t: SubscribeTranslator) =>
  z.object({
    name: z
      .string()
      .trim()
      .min(1, t("form.validation.nameRequired", "Client name is required"))
      .max(255, t("form.validation.nameTooLong", "Use at most 255 characters")),
    description: z
      .string()
      .max(
        255,
        t("form.validation.descriptionTooLong", "Use at most 255 characters")
      )
      .optional(),
    icon: optionalImageSource(
      t(
        "form.validation.iconSource",
        "Enter an HTTP(S), relative, or image data URL"
      )
    ),
    user_agent: z
      .string()
      .trim()
      .min(
        1,
        `User-Agent ${t("form.validation.userAgentRequiredSuffix", "is required")}`
      )
      .max(
        255,
        t("form.validation.userAgentTooLong", "Use at most 255 characters")
      ),
    enabled: z.boolean(),
    scheme: z.string().max(255).optional(),
    template: z
      .string()
      .min(
        1,
        t(
          "form.validation.templateRequired",
          "Subscription template is required"
        )
      ),
    output_format: z
      .string()
      .min(
        1,
        t("form.validation.outputFormatRequired", "Output format is required")
      )
      .refine(
        (value) => OUTPUT_FORMATS.includes(value),
        t(
          "form.validation.outputFormatInvalid",
          "Select a supported output format"
        )
      ),
    download_link: z.object({
      windows: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
      mac: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
      linux: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
      ios: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
      android: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
      harmony: optionalHttpUrl(
        t("form.validation.downloadUrl", "Enter a valid HTTP or HTTPS URL")
      ),
    }),
  });

type ClientFormData = z.infer<ReturnType<typeof createClientFormSchema>>;
type SubscribeApplicationRow = API.SubscribeApplicationListItem;
type ClientFormTab = "basic" | "template" | "download";
type ClientListFilters = {
  search?: string;
  enabled?: string;
};

function tabForClientErrors(
  errors: FieldErrors<ClientFormData>
): ClientFormTab {
  if (
    errors.name ||
    errors.description ||
    errors.icon ||
    errors.user_agent ||
    errors.enabled
  ) {
    return "basic";
  }
  if (errors.output_format || errors.scheme || errors.template) {
    return "template";
  }
  if (errors.download_link) return "download";
  return "basic";
}

function ClientIcon({ icon }: { icon?: string }) {
  const [failed, setFailed] = useState(false);

  return (
    <div className="subscribe-client-icon" aria-hidden="true">
      {icon && !failed ? (
        <img
          alt=""
          className="size-full object-contain"
          height={32}
          onError={() => setFailed(true)}
          src={icon}
          width={32}
        />
      ) : (
        <AppWindow className="size-4" />
      )}
    </div>
  );
}

const DEFAULT_DESCRIPTION_KEYS: Record<string, string> = {
  "Default URI subscription template for generic clients.": "defaultUri",
  "通用 URI 订阅模板。": "defaultUri",
  "Shadowrocket URI subscription template.": "shadowrocket",
  "Shadowrocket URI 订阅模板。": "shadowrocket",
  "Clash and Mihomo YAML subscription template.": "clash",
  "Clash/Mihomo YAML 订阅模板。": "clash",
  "sing-box 1.12 JSON subscription template.": "singBox112",
  "sing-box 1.12 JSON 订阅模板。": "singBox112",
  "Surge managed configuration template.": "surge",
  "Surge 托管配置模板。": "surge",
  "Stash YAML subscription template.": "stash",
  "Stash YAML 订阅模板。": "stash",
  "Surfboard configuration template.": "surfboard",
  "Surfboard 配置模板。": "surfboard",
  "Loon configuration template.": "loon",
  "Loon 配置模板。": "loon",
  "Quantumult configuration template.": "quantumult",
  "Quantumult 配置模板。": "quantumult",
  "Quantumult X configuration template.": "quantumultX",
  "Quantumult X 配置模板。": "quantumultX",
  "Egern YAML subscription template.": "egern",
  "Egern YAML 订阅模板。": "egern",
  "Hiddify JSON subscription template.": "hiddify",
  "Hiddify JSON 订阅模板。": "hiddify",
  "sing-box 1.11 JSON subscription template.": "singBox111",
  "sing-box 1.11 JSON 订阅模板。": "singBox111",
  "Xray-core client config with one local SOCKS inbound per node and one-to-one routing.":
    "xrayCore",
  "Xray-core 客户端 JSON 模板，为每个节点生成本地 SOCKS 入站与一对一路由。":
    "xrayCore",
  "Surge config that connects to local xray-core SOCKS inbounds while reusing the built-in Surge rules and policy groups.":
    "surgeXrayCore",
  "Surge 配置模板，连接本地 xray-core SOCKS 入站并复用 Surge 规则与策略组。":
    "surgeXrayCore",
};

export function ProtocolForm() {
  const { t } = useTranslation("subscribe");
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<ClientFormTab>("basic");
  const [pendingClientId, setPendingClientId] = useState<number | null>(null);
  const [batchDeleting, setBatchDeleting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [editingClient, setEditingClient] =
    useState<API.SubscribeApplication | null>(null);
  const tableRef = useRef<ProTableActions>(null);

  const clientFormSchema = useMemo(() => createClientFormSchema(t), [t]);
  const getDescription = (description?: string) => {
    if (!description) return "—";
    const key = DEFAULT_DESCRIPTION_KEYS[description];
    return key ? t(`defaultDescriptions.${key}`, description) : description;
  };

  const form = useForm<ClientFormData>({
    resolver: zodResolver(clientFormSchema),
    mode: "onChange",
    defaultValues: createSubscribeClientFormValues(),
  });

  const request = async (
    pagination: { page: number; size: number },
    filter: ClientListFilters
  ) => {
    const { data } = await getSubscribeApplicationList({
      page: pagination.page,
      size: pagination.size,
      search: filter.search?.trim() || undefined,
      enabled:
        filter.enabled === "true"
          ? true
          : filter.enabled === "false"
            ? false
            : undefined,
    });

    return normalizePageResult(data.data);
  };

  const handleToggleClient = async (
    client: SubscribeApplicationRow,
    patch: Pick<API.UpdateSubscribeApplicationRequest, "enabled" | "is_default">
  ) => {
    if (client.is_default && patch.enabled === false) {
      toast.error(
        t(
          "actions.defaultMustStayEnabled",
          "Set another default client before disabling this one"
        )
      );
      return;
    }
    setPendingClientId(client.id);
    try {
      await updateSubscribeApplication({
        id: client.id,
        ...patch,
        ...(patch.is_default ? { enabled: true } : {}),
      });
      tableRef.current?.refresh();
      if (patch.is_default) {
        toast.success(t("actions.setDefaultSuccess", "Default client updated"));
      }
    } catch (error) {
      console.error("Failed to update client:", error);
      toast.error(t("actions.updateFailed", "Update failed"));
    } finally {
      setPendingClientId(null);
    }
  };

  const columns: ColumnDef<SubscribeApplicationRow, unknown>[] = [
    {
      accessorKey: "name",
      header: t("table.columns.client", "Client"),
      size: 360,
      cell: ({ row }) => {
        const description = getDescription(row.original.description);
        return (
          <div className="subscribe-client-cell">
            <ClientIcon icon={row.original.icon} />
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                <span className="truncate font-medium">
                  {row.original.name}
                </span>
                {row.original.is_default ? (
                  <Badge className="shrink-0 text-[10px]" variant="outline">
                    <Star aria-hidden="true" className="size-3 fill-current" />
                    {t("table.defaultBadge", "Default")}
                  </Badge>
                ) : null}
              </div>
              <div className="mt-0.5 truncate font-mono text-muted-foreground text-xs">
                {row.original.user_agent}
              </div>
              {description !== "—" ? (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <p className="subscribe-client-description">
                        {description}
                      </p>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-xs">
                      <p>{description}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : null}
              <div className="subscribe-client-mobile-meta">
                <Switch
                  aria-label={`${t("table.columns.enabled", "Enabled")} ${row.original.name}`}
                  checked={row.original.enabled !== false}
                  disabled={
                    row.original.is_default ||
                    pendingClientId === row.original.id
                  }
                  onCheckedChange={async (checked) => {
                    await handleToggleClient(row.original, {
                      enabled: checked,
                    });
                  }}
                />
                <span className="text-muted-foreground text-xs">
                  {row.original.enabled !== false
                    ? t("table.enabled", "Enabled")
                    : t("table.disabled", "Disabled")}
                </span>
                <Badge className="text-[11px]" variant="secondary">
                  {t(
                    `outputFormats.${row.original.output_format}`,
                    row.original.output_format
                  ) || row.original.output_format}
                </Badge>
              </div>
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "output_format",
      header: t("table.columns.delivery", "Delivery"),
      size: 280,
      cell: ({ row }) => {
        const platforms = Object.entries(row.original.download_link || {})
          .filter(([, value]) => Boolean(value))
          .map(([key]) => key);
        return (
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-1.5">
              <Badge className="text-xs" variant="secondary">
                {t(
                  `outputFormats.${row.original.output_format}`,
                  row.original.output_format
                ) || row.original.output_format}
              </Badge>
              {row.original.scheme ? (
                <Badge className="text-xs" variant="outline">
                  {t("table.deepLink", "Deep link")}
                </Badge>
              ) : null}
            </div>
            {platforms.length ? (
              <div className="flex flex-wrap gap-1">
                {platforms.slice(0, 3).map((key) => (
                  <Badge className="text-[11px]" key={key} variant="outline">
                    {t(`platforms.${key}`, key)}
                  </Badge>
                ))}
                {platforms.length > 3 ? (
                  <Badge className="text-[11px]" variant="outline">
                    +{platforms.length - 3}
                  </Badge>
                ) : null}
              </div>
            ) : (
              <span className="text-muted-foreground text-xs">
                {t("table.noDownloads", "No download links")}
              </span>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: "enabled",
      header: t("table.columns.status", "Status"),
      size: 130,
      cell: ({ row }) => (
        <div className="subscribe-client-status">
          <Switch
            aria-label={`${t("table.columns.enabled", "Enabled")} ${row.original.name}`}
            checked={row.original.enabled !== false}
            disabled={
              row.original.is_default || pendingClientId === row.original.id
            }
            onCheckedChange={async (checked) => {
              await handleToggleClient(row.original, { enabled: checked });
            }}
          />
          <div className="min-w-0">
            <div className="font-medium text-xs">
              {row.original.enabled !== false
                ? t("table.enabled", "Enabled")
                : t("table.disabled", "Disabled")}
            </div>
          </div>
        </div>
      ),
    },
  ];

  const handleAdd = () => {
    setEditingClient(null);
    setActiveTab("basic");
    form.reset(createSubscribeClientFormValues());
    setOpen(true);
  };

  const handleEdit = async (client: SubscribeApplicationRow) => {
    setPendingClientId(client.id);
    try {
      const { data } = await getSubscribeApplication({ id: client.id });
      const detail = data.data;
      if (!detail) {
        throw new Error("missing subscribe application detail");
      }
      setEditingClient(detail);
      setActiveTab("basic");
      form.reset(createSubscribeClientFormValues(detail));
      setOpen(true);
    } catch (error) {
      console.error("Failed to load client:", error);
      toast.error(t("actions.loadFailed", "Load failed"));
    } finally {
      setPendingClientId(null);
    }
  };

  const handleDelete = async (client: SubscribeApplicationRow) => {
    if (client.is_default) {
      toast.error(
        t(
          "actions.defaultCannotDelete",
          "Set another default client before deleting this one"
        )
      );
      return;
    }
    setPendingClientId(client.id);
    try {
      await deleteSubscribeApplication({ id: client.id });
      tableRef.current?.refresh();
      toast.success(t("actions.deleteSuccess", "Deleted successfully"));
    } catch (error) {
      console.error("Failed to delete client:", error);
      toast.error(t("actions.deleteFailed", "Delete failed"));
    } finally {
      setPendingClientId(null);
    }
  };

  const handleBatchDelete = async (clients: SubscribeApplicationRow[]) => {
    if (clients.some((client) => client.is_default)) {
      toast.error(
        t(
          "actions.defaultCannotDelete",
          "Set another default client before deleting this one"
        )
      );
      return;
    }
    setBatchDeleting(true);
    try {
      const results = await Promise.allSettled(
        clients.map((client) => deleteSubscribeApplication({ id: client.id }))
      );
      const successCount = results.filter(
        (result) => result.status === "fulfilled"
      ).length;
      const failedCount = clients.length - successCount;
      if (successCount > 0) tableRef.current?.refresh();
      if (failedCount === 0) {
        toast.success(
          t(
            "actions.batchDeleteSuccess",
            "Successfully deleted {{count}} clients",
            { count: clients.length }
          )
        );
      } else if (successCount > 0) {
        toast.warning(
          t(
            "actions.batchDeletePartial",
            "Deleted {{success}} clients; {{failed}} could not be deleted",
            { failed: failedCount, success: successCount }
          )
        );
      } else {
        toast.error(t("actions.deleteFailed", "Delete failed"));
      }
    } finally {
      setBatchDeleting(false);
    }
  };

  const onSubmit = async (data: ClientFormData) => {
    setSaving(true);
    try {
      if (editingClient) {
        await updateSubscribeApplication({
          ...data,
          enabled: data.enabled ?? editingClient.enabled ?? true,
          is_default: editingClient.is_default,
          id: editingClient.id,
        });
        toast.success(t("actions.updateSuccess", "Updated successfully"));
      } else {
        await createSubscribeApplication({
          ...data,
          enabled: data.enabled ?? true,
          is_default: false,
        });
        toast.success(t("actions.createSuccess", "Created successfully"));
      }

      setOpen(false);
      tableRef.current?.refresh();
    } catch (error) {
      console.error("Failed to save client:", error);
      toast.error(t("actions.saveFailed", "Save failed"));
    } finally {
      setSaving(false);
    }
  };

  const handleInvalidSubmit = (errors: FieldErrors<ClientFormData>) => {
    setActiveTab(tabForClientErrors(errors));
  };

  const handleSheetOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && saving) return;
    if (!nextOpen && form.formState.isDirty) {
      setDiscardOpen(true);
      return;
    }
    setOpen(nextOpen);
  };

  const discardChanges = () => {
    form.reset(form.getValues());
    setDiscardOpen(false);
    setOpen(false);
  };

  return (
    <>
      <section className="subscribe-client-panel">
        <header className="subscribe-client-panel-header">
          <div className="min-w-0">
            <h2 className="font-semibold text-base">
              {t("protocol.title", "Client templates")}
            </h2>
            <p className="mt-1 text-muted-foreground text-sm">
              {t(
                "protocol.description",
                "Control output format, client detection, and installation links"
              )}
            </p>
          </div>
          <div className="subscribe-client-panel-actions">
            <Button asChild size="sm" variant="outline">
              <a
                href="https://github.com/neibniz/alpha/tree/main/server/reference/subscription-templates"
                rel="noreferrer noopener"
                target="_blank"
              >
                <Github aria-hidden="true" className="size-4" />
                {t("protocol.templateRepo", "Template Repo")}
                <ExternalLink aria-hidden="true" className="size-3.5" />
              </a>
            </Button>
            <Button onClick={handleAdd} size="sm">
              <Plus aria-hidden="true" className="size-4" />
              {t("actions.add", "Add")}
            </Button>
          </div>
        </header>
        <div className="subscribe-client-table">
          <ProTable<SubscribeApplicationRow, ClientListFilters>
            action={tableRef}
            actions={{
              columnSize: 152,
              render: (row: SubscribeApplicationRow) => [
                <span
                  className="subscribe-template-preview-action"
                  key="preview"
                >
                  <TemplatePreview
                    applicationId={row.id}
                    compact
                    output_format={row.output_format}
                  />
                </span>,
                <Button
                  aria-label={`${t("actions.setDefault", "Set as default")} ${row.name}`}
                  disabled={
                    row.is_default ||
                    pendingClientId === row.id ||
                    batchDeleting
                  }
                  key="default"
                  onClick={() =>
                    handleToggleClient(row, {
                      is_default: true,
                    })
                  }
                  size="icon-sm"
                  title={
                    row.is_default
                      ? t("table.defaultBadge", "Default")
                      : t("actions.setDefault", "Set as default")
                  }
                  variant="ghost"
                >
                  <Star
                    aria-hidden="true"
                    className={
                      row.is_default ? "size-4 fill-current" : "size-4"
                    }
                  />
                </Button>,
                <Button
                  aria-label={t("actions.edit", "Edit")}
                  disabled={pendingClientId === row.id || batchDeleting}
                  key="edit"
                  onClick={() => handleEdit(row)}
                  size="icon-sm"
                  title={t("actions.edit", "Edit")}
                  variant="ghost"
                >
                  <Pencil aria-hidden="true" className="size-4" />
                </Button>,
                <ConfirmButton
                  cancelText={t("actions.cancel", "Cancel")}
                  confirmText={t("actions.confirm", "Confirm")}
                  description={t(
                    "actions.deleteWarning",
                    "This operation cannot be undone. Are you sure you want to delete this client?"
                  )}
                  key="delete"
                  onConfirm={() => handleDelete(row)}
                  title={t("actions.confirmDelete", "Confirm Delete")}
                  trigger={
                    <Button
                      aria-label={t("actions.delete", "Delete")}
                      disabled={
                        row.is_default ||
                        pendingClientId === row.id ||
                        batchDeleting
                      }
                      size="icon-sm"
                      title={
                        row.is_default
                          ? t(
                              "actions.defaultCannotDelete",
                              "Set another default client before deleting this one"
                            )
                          : t("actions.delete", "Delete")
                      }
                      variant="ghost"
                    >
                      <Trash2
                        aria-hidden="true"
                        className="size-4 text-destructive"
                      />
                    </Button>
                  }
                />,
              ],
              batchRender: (rows: SubscribeApplicationRow[]) => [
                <ConfirmButton
                  cancelText={t("actions.cancel", "Cancel")}
                  confirmText={t("actions.confirm", "Confirm")}
                  description={t(
                    "actions.batchDeleteWarning",
                    "Are you sure you want to delete the selected {count} clients?",
                    {
                      count: rows.length,
                    }
                  )}
                  key="batchDelete"
                  onConfirm={() => handleBatchDelete(rows)}
                  title={t("actions.confirmDelete", "Confirm Delete")}
                  trigger={
                    <Button disabled={batchDeleting} variant="destructive">
                      {batchDeleting ? (
                        <Icon className="animate-spin" icon="mdi:loading" />
                      ) : null}
                      {t("actions.batchDelete", "Batch Delete")}
                    </Button>
                  }
                />,
              ],
            }}
            columns={columns}
            header={{
              title: t("protocol.directory", "Client directory"),
            }}
            params={[
              {
                key: "search",
                placeholder: t(
                  "protocol.searchPlaceholder",
                  "Search name, User-Agent, or format"
                ),
                className: "sm:w-72",
              },
              {
                key: "enabled",
                placeholder: t("protocol.statusFilter", "All statuses"),
                type: "select",
                options: [
                  { label: t("table.enabled", "Enabled"), value: "true" },
                  { label: t("table.disabled", "Disabled"), value: "false" },
                ],
              },
            ]}
            request={request}
          />
        </div>
      </section>

      <Sheet onOpenChange={handleSheetOpenChange} open={open}>
        <SheetContent className="subscribe-client-sheet w-[760px] max-w-full md:max-w-3xl">
          <SheetHeader>
            <SheetTitle>
              {editingClient
                ? t("form.editTitle", "Edit Client")
                : t("form.addTitle", "Add Client")}
            </SheetTitle>
            <SheetDescription>
              {t(
                "form.sheetDescription",
                "Configure client detection, rendered subscription output, and platform links"
              )}
            </SheetDescription>
          </SheetHeader>
          <ScrollArea className="min-h-0 flex-1 px-6">
            <Form {...form}>
              <AdminForm
                className="space-y-6 py-5"
                id="subscribe-client-form"
                onSubmit={form.handleSubmit(onSubmit, handleInvalidSubmit)}
              >
                <Tabs
                  className="subscribe-client-tabs w-full"
                  onValueChange={(value) =>
                    setActiveTab(value as ClientFormTab)
                  }
                  value={activeTab}
                >
                  <TabsList className="grid w-full grid-cols-3 md:w-[420px]">
                    <TabsTrigger value="basic">
                      {t("form.tabs.basic", "Basic Info")}
                    </TabsTrigger>
                    <TabsTrigger value="template">
                      {t("form.tabs.template", "Templates")}
                    </TabsTrigger>
                    <TabsTrigger value="download">
                      {t("form.tabs.download", "Downloads")}
                    </TabsTrigger>
                  </TabsList>

                  <TabsContent
                    className="subscribe-client-basic grid gap-4 pt-4 md:grid-cols-2"
                    value="basic"
                  >
                    <div className="subscribe-client-tab-heading md:col-span-2">
                      <h3 className="font-semibold text-sm">
                        {t("form.sections.identity", "Client identity")}
                      </h3>
                      <p className="text-muted-foreground text-xs leading-5">
                        {t(
                          "form.sections.identityDescription",
                          "Name the client and define the User-Agent used to match requests"
                        )}
                      </p>
                    </div>
                    <FormField
                      control={form.control}
                      name="icon"
                      render={({ field }) => (
                        <FormItem className="md:col-span-2">
                          <FormLabel>{t("form.fields.icon", "Icon")}</FormLabel>
                          <FormControl>
                            <EnhancedInput
                              onValueChange={(value) => field.onChange(value)}
                              placeholder="https://example.com/icon.png"
                              suffix={
                                <UploadImage
                                  className="h-9 rounded-none border-none bg-muted px-2"
                                  onChange={(value) => field.onChange(value)}
                                />
                              }
                              type="text"
                              value={field.value}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              "form.descriptions.icon",
                              "Icon URL or base64 encoding"
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="name"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.fields.name", "Name")}</FormLabel>
                          <FormControl>
                            <Input placeholder="Clash for Windows" {...field} />
                          </FormControl>
                          <FormDescription>
                            {t("form.descriptions.name", "Client display name")}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="user_agent"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.fields.userAgent", "User-Agent")}
                          </FormLabel>
                          <FormControl>
                            <Input placeholder="Clash" {...field} />
                          </FormControl>
                          <FormDescription>
                            {t(
                              "form.descriptions.userAgentPrefix",
                              "Client identifier for distinguishing different clients"
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="enabled"
                      render={({ field }) => (
                        <FormItem className="md:col-span-2">
                          <div className="flex items-center justify-between rounded-md border px-3 py-2">
                            <div className="space-y-0.5">
                              <FormLabel>{t("form.enabled", "启用")}</FormLabel>
                              <FormDescription>
                                {editingClient?.is_default
                                  ? t(
                                      "form.defaultMustStayEnabled",
                                      "Set another default client before disabling this one"
                                    )
                                  : t(
                                      "form.enabledDescription",
                                      "关闭后用户端不显示，也不能通过该客户端模版生成订阅"
                                    )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                aria-label={t("form.enabled", "Enabled")}
                                checked={field.value !== false}
                                disabled={editingClient?.is_default}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="description"
                      render={({ field }) => (
                        <FormItem className="md:col-span-2">
                          <FormLabel>
                            {t("form.fields.description", "Description")}
                          </FormLabel>
                          <FormControl>
                            <Textarea
                              placeholder={t(
                                "form.descriptions.description",
                                "Detailed client description"
                              )}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              "form.descriptions.description",
                              "Detailed client description"
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </TabsContent>

                  <TabsContent
                    className="subscribe-client-template grid gap-4 pt-4 md:grid-cols-2"
                    value="template"
                  >
                    <div className="subscribe-client-tab-heading md:col-span-2">
                      <h3 className="font-semibold text-sm">
                        {t("form.sections.rendering", "Output rendering")}
                      </h3>
                      <p className="text-muted-foreground text-xs leading-5">
                        {t(
                          "form.sections.renderingDescription",
                          "Choose the response format, deep-link scheme, and Go template"
                        )}
                      </p>
                    </div>
                    <FormField
                      control={form.control}
                      name="output_format"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.fields.outputFormat", "Output Format")}
                          </FormLabel>
                          <FormControl>
                            <Select
                              onValueChange={field.onChange}
                              value={field.value}
                            >
                              <SelectTrigger className="w-full">
                                <SelectValue
                                  placeholder={t(
                                    "form.selectPlaceholder",
                                    "Select ..."
                                  )}
                                />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="base64">
                                  {t("outputFormats.base64", "Base64")}
                                </SelectItem>
                                <SelectItem value="yaml">
                                  {t("outputFormats.yaml", "YAML")}
                                </SelectItem>
                                <SelectItem value="json">
                                  {t("outputFormats.json", "JSON")}
                                </SelectItem>
                                <SelectItem value="conf">
                                  {t("outputFormats.conf", "CONF")}
                                </SelectItem>
                                <SelectItem value="plain">
                                  {t("outputFormats.plain", "Plain Text")}
                                </SelectItem>
                              </SelectContent>
                            </Select>
                          </FormControl>
                          <FormDescription>
                            {t(
                              "form.descriptions.outputFormat",
                              "Subscription configuration file format"
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="scheme"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel className="flex items-center gap-2">
                            {t("form.fields.scheme", "URL Scheme")}
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                  }}
                                >
                                  <Icon
                                    className="h-4 w-4 text-muted-foreground"
                                    icon="mdi:help-circle-outline"
                                  />
                                </TooltipTrigger>
                                <TooltipContent
                                  className="max-w-md bg-secondary text-secondary-foreground"
                                  side="right"
                                >
                                  <div className="space-y-2 text-sm">
                                    <div className="font-medium">
                                      {t(
                                        "form.descriptions.scheme.title",
                                        "URL Scheme template"
                                      )}
                                    </div>

                                    <div>
                                      <div className="font-medium">
                                        {t(
                                          "form.descriptions.scheme.variables",
                                          "Supports variables:"
                                        )}
                                      </div>
                                      <ul className="ml-2 list-disc space-y-1 text-xs">
                                        <li>
                                          <code className="rounded px-1">
                                            {"${url}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.scheme.urlVariable",
                                            "subscription URL"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            {"${name}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.scheme.nameVariable",
                                            "site name"
                                          )}
                                        </li>
                                      </ul>
                                    </div>

                                    <div>
                                      <div className="font-medium">
                                        {t(
                                          "form.descriptions.scheme.functions",
                                          "Supports functions:"
                                        )}
                                      </div>
                                      <ul className="ml-2 list-disc space-y-1 text-xs">
                                        <li>
                                          <code className="rounded px-1">
                                            {"${encodeURIComponent(...)}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.scheme.urlEncoding",
                                            "URL encoding"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            {"${window.btoa(...)}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.scheme.base64Encoding",
                                            "Base64 encoding"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            {"${JSON.stringify(...)}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.scheme.jsonStringify",
                                            "JSON object to string"
                                          )}
                                        </li>
                                      </ul>
                                    </div>
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </FormLabel>
                          <FormControl>
                            <Input
                              placeholder="clash://install-config?url=${url}&name=${name}"
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="template"
                      render={({ field }) => (
                        <FormItem className="md:col-span-2">
                          <FormLabel className="flex items-center gap-2">
                            {t(
                              "form.fields.template",
                              "Subscription File Template"
                            )}
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                  }}
                                >
                                  <Icon
                                    className="h-4 w-4 text-muted-foreground"
                                    icon="mdi:help-circle-outline"
                                  />
                                </TooltipTrigger>
                                <TooltipContent
                                  className="max-w-md bg-secondary text-secondary-foreground"
                                  side="right"
                                >
                                  <div className="space-y-2 text-sm">
                                    <div className="font-medium">
                                      {t(
                                        "form.descriptions.template.title",
                                        "Go Template Syntax"
                                      )}
                                    </div>

                                    <div>
                                      <div className="font-medium">
                                        {t(
                                          "form.descriptions.template.variables",
                                          "Available variables:"
                                        )}
                                      </div>
                                      <ul className="ml-2 list-disc space-y-1 text-xs">
                                        <li>
                                          <code className="rounded px-1">
                                            .SiteName
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.siteName",
                                            "site name"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            .SubscribeName
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.subscribeName",
                                            "subscription name"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            .Proxies
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.nodes",
                                            "proxy nodes list"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            .UserInfo
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.userInfo",
                                            "user info (traffic, expiry, etc.)"
                                          )}
                                        </li>
                                      </ul>
                                    </div>

                                    <div>
                                      <div className="font-medium">
                                        {t(
                                          "form.descriptions.template.functions",
                                          "Template functions:"
                                        )}
                                      </div>
                                      <ul className="ml-2 list-disc space-y-1 text-xs">
                                        <li>
                                          <code className="rounded px-1">
                                            {"{{range .Proxies}}...{{end}}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.range",
                                            "iterate arrays"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            {"{{if .condition}}...{{end}}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.if",
                                            "conditional statements"
                                          )}
                                        </li>
                                        <li>
                                          <code className="rounded px-1">
                                            {"{{sprig_func}}"}
                                          </code>{" "}
                                          -{" "}
                                          {t(
                                            "form.descriptions.template.sprig",
                                            "Sprig function library (string processing, dates, etc.)"
                                          )}
                                        </li>
                                      </ul>
                                    </div>
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </FormLabel>
                          <FormControl>
                            <GoTemplateEditor
                              enableSprig
                              onChange={(value: string | undefined) =>
                                field.onChange(value || "")
                              }
                              schema={subscribeSchema}
                              showLineNumbers
                              value={field.value || ""}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </TabsContent>

                  <TabsContent className="space-y-4 pt-4" value="download">
                    <div className="subscribe-client-tab-heading">
                      <h3 className="font-semibold text-sm">
                        {t("form.sections.downloads", "Platform downloads")}
                      </h3>
                      <p className="text-muted-foreground text-xs leading-5">
                        {t(
                          "form.sections.downloadsDescription",
                          "Optional official HTTP or HTTPS download links shown to users"
                        )}
                      </p>
                    </div>
                    <div>
                      <div className="grid gap-4 md:grid-cols-2">
                        {[
                          "windows",
                          "mac",
                          "linux",
                          "ios",
                          "android",
                          "harmony",
                        ].map((key) => (
                          <FormField
                            control={form.control}
                            key={key}
                            name={
                              `download_link.${key}` as `download_link.${keyof ClientFormData["download_link"]}`
                            }
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t(`platforms.${key}`, key)}
                                </FormLabel>
                                <FormControl>
                                  <Input {...field} />
                                </FormControl>
                                <FormDescription>
                                  {t(`platforms.${key}`, key)}{" "}
                                  {t(
                                    "form.descriptions.downloadLink",
                                    "platform download URL"
                                  )}
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        ))}
                      </div>
                    </div>
                  </TabsContent>
                </Tabs>
              </AdminForm>
            </Form>
          </ScrollArea>
          <SheetFooter className="subscribe-client-footer flex-row justify-end gap-2">
            <Button
              disabled={saving}
              onClick={() => handleSheetOpenChange(false)}
              type="button"
              variant="outline"
            >
              {t("actions.cancel", "Cancel")}
            </Button>
            <Button
              disabled={saving || !form.formState.isDirty}
              form="subscribe-client-form"
              type="submit"
            >
              {saving && (
                <Icon className="mr-2 animate-spin" icon="mdi:loading" />
              )}
              {editingClient
                ? t("actions.update", "Update")
                : t("actions.add", "Add")}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <DiscardChangesDialog
        onDiscard={discardChanges}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
