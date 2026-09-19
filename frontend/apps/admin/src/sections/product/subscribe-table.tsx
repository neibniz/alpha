"use client";

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
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Checkbox } from "@workspace/ui/components/checkbox";
import { Field, FieldLabel } from "@workspace/ui/components/field";
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
import { Tabs, TabsList, TabsTrigger } from "@workspace/ui/components/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@workspace/ui/components/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { cn } from "@workspace/ui/lib/utils";
import {
  batchDeleteSubscribe,
  createSubscribe,
  deleteSubscribe,
  getSubscribeDetails,
  subscribeSort,
  updateSubscribe,
} from "@workspace/ui/services/admin/subscribe";
import {
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  Check,
  Circle,
  CircleDollarSign,
  Copy,
  Download,
  Eye,
  MoreHorizontal,
  Package,
  Pencil,
  Plus,
  Search,
  Settings2,
  ShoppingCart,
  Trash2,
  UserRound,
  Users,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { Display } from "@/components/display";
import { useGlobalStore } from "@/stores/global";
import { useSubscribe } from "@/stores/subscribe";
import {
  buildProductPortfolioSummary,
  filterProducts,
  getProductLifecycle,
  getProductPositioning,
  getVisibleProductMatrixRows,
  isProductLowStock,
  moveProduct,
  type ProductLifecycle,
  type ProductMatrixRowKey,
  sortProducts,
} from "./product-data";
import SubscribeForm from "./subscribe-form";

const MATRIX_PAGE_SIZE = 4;

async function loadSubscribeDetail(id?: number) {
  if (!id) throw new Error("Missing subscribe id");
  const { data } = await getSubscribeDetails({ id });
  if (!data.data) throw new Error("Missing subscribe detail");
  return data.data;
}

function formatTimestamp(timestamp?: number) {
  if (!timestamp) return "—";
  const value = timestamp < 1_000_000_000_000 ? timestamp * 1000 : timestamp;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

function formatSpeedLimit(value?: number) {
  const bits = Math.max(0, Number(value) || 0);
  if (bits === 0) return "∞";
  const megabits = bits / 1024 ** 2;
  if (megabits >= 1024) {
    const gigabits = megabits / 1024;
    return `${gigabits.toFixed(Number.isInteger(gigabits) ? 0 : 1)} Gbps`;
  }
  return `${megabits.toFixed(Number.isInteger(megabits) ? 0 : 1)} Mbps`;
}

type ProductSaleState =
  | "available"
  | "comingSoon"
  | "ended"
  | "outOfStock"
  | "paused";

function getProductSaleState(product: API.SubscribeItem): ProductSaleState {
  if (!product.sell) return "paused";
  const now = Date.now();
  if (product.sale_starts_at && now < product.sale_starts_at)
    return "comingSoon";
  if (product.sale_ends_at && now >= product.sale_ends_at) return "ended";
  if (product.inventory === 0) return "outOfStock";
  return "available";
}

function ProductStatus({
  lifecycle,
  labels,
}: {
  lifecycle: ProductLifecycle;
  labels: Record<ProductLifecycle, string>;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 text-xs",
        lifecycle === "selling" && "text-emerald-500",
        lifecycle === "draft" && "text-sky-500",
        lifecycle === "hidden" && "text-muted-foreground"
      )}
    >
      <Circle className="size-2 fill-current" />
      {labels[lifecycle]}
    </span>
  );
}

function VisibilityIndicators({
  product,
  labels,
}: {
  product: API.SubscribeItem;
  labels: { guest: string; user: string; purchase: string };
}) {
  const items = [
    { key: "show", label: labels.guest, enabled: product.show, icon: Eye },
    {
      key: "user",
      label: labels.user,
      enabled: product.user_visible,
      icon: UserRound,
    },
    {
      key: "sell",
      label: labels.purchase,
      enabled: product.sell,
      icon: ShoppingCart,
    },
  ];
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((item) => {
        const IconComponent = item.icon;
        return (
          <Tooltip key={item.key}>
            <TooltipTrigger asChild>
              <span
                className={cn(
                  "inline-flex h-6 items-center gap-1 rounded-md border px-1.5 text-[11px]",
                  item.enabled
                    ? "border-primary/30 bg-primary/[0.06] text-primary"
                    : "border-border text-muted-foreground"
                )}
              >
                <IconComponent className="size-3" />
                {item.label}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {item.label}: {item.enabled ? "ON" : "OFF"}
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

function InventoryValue({
  product,
  unlimitedLabel,
  remainingLabel,
}: {
  product: API.SubscribeItem;
  unlimitedLabel: string;
  remainingLabel: (count: number) => string;
}) {
  if (product.inventory < 0) {
    return <span className="text-sm">{unlimitedLabel}</span>;
  }
  const low = isProductLowStock(product);
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 text-sm tabular-nums",
        low && "text-amber-500"
      )}
    >
      {low ? <AlertTriangle className="size-3.5" /> : null}
      {remainingLabel(product.inventory)}
    </span>
  );
}

export default function SubscribeTable() {
  const { t, i18n } = useTranslation("product");
  const currencySymbol = useGlobalStore(
    (state) => state.common.currency?.currency_symbol || ""
  );
  const { subscribes, fetchSubscribes, loading: storeLoading } = useSubscribe();
  const [operationLoading, setOperationLoading] = useState(false);
  const [view, setView] = useState<"matrix" | "queue">("matrix");
  const [search, setSearch] = useState("");
  const [language, setLanguage] = useState("all");
  const [onlyDifferences, setOnlyDifferences] = useState(false);
  const [matrixPage, setMatrixPage] = useState(0);
  const [sortMode, setSortMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [deleteTargets, setDeleteTargets] = useState<API.SubscribeItem[]>([]);

  const lifecycleLabels: Record<ProductLifecycle, string> = {
    selling: t("lifecycle.selling", "On sale"),
    draft: t("lifecycle.draft", "Draft"),
    hidden: t("lifecycle.hidden", "Hidden"),
  };
  const visibilityLabels = {
    guest: t("visibility.guest", "Guest"),
    user: t("visibility.user", "User"),
    purchase: t("visibility.purchase", "Purchase"),
  };

  const products = useMemo(() => sortProducts(subscribes), [subscribes]);
  const summary = useMemo(
    () => buildProductPortfolioSummary(products),
    [products]
  );
  const languageOptions = useMemo(
    () =>
      Array.from(
        new Set(products.map((product) => product.language).filter(Boolean))
      ).sort((left, right) => left.localeCompare(right)),
    [products]
  );
  const filteredProducts = useMemo(
    () => filterProducts(products, search, language),
    [language, products, search]
  );
  const sellingProducts = useMemo(
    () =>
      filteredProducts.filter(
        (product) => getProductLifecycle(product) === "selling"
      ),
    [filteredProducts]
  );
  const matrixPageCount = Math.max(
    1,
    Math.ceil(sellingProducts.length / MATRIX_PAGE_SIZE)
  );
  const matrixProducts = sellingProducts.slice(
    matrixPage * MATRIX_PAGE_SIZE,
    (matrixPage + 1) * MATRIX_PAGE_SIZE
  );
  const matrixRows = getVisibleProductMatrixRows(
    matrixProducts,
    onlyDifferences
  );
  const pendingProducts = filteredProducts.filter(
    (product) => getProductLifecycle(product) !== "selling"
  );
  const lowStockProducts = products.filter(isProductLowStock);

  useEffect(() => {
    if (matrixPage >= matrixPageCount) setMatrixPage(matrixPageCount - 1);
  }, [matrixPage, matrixPageCount]);

  const refreshData = useCallback(async () => {
    await fetchSubscribes();
  }, [fetchSubscribes]);

  const createProduct = useCallback(
    async (values: API.CreateSubscribeRequest) => {
      setOperationLoading(true);
      try {
        await createSubscribe({
          ...values,
          allow_renewal: values.allow_renewal ?? true,
          allow_reset_traffic: values.allow_reset_traffic ?? true,
        });
        toast.success(t("createSuccess", "Product created"));
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

  const updateProduct = useCallback(
    async (
      product: API.SubscribeItem,
      values: Omit<API.UpdateSubscribeRequest, "id">
    ) => {
      setOperationLoading(true);
      try {
        await updateSubscribe({ ...values, id: product.id });
        toast.success(t("updateSuccess", "Product updated"));
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

  const duplicateProduct = useCallback(
    async (product: API.SubscribeItem) => {
      setOperationLoading(true);
      try {
        const detail = await loadSubscribeDetail(product.id);
        const {
          id: _id,
          sort: _sort,
          sell: _sell,
          show: _show,
          user_visible: _userVisible,
          updated_at: _updatedAt,
          created_at: _createdAt,
          ...payload
        } = detail;
        await createSubscribe({
          ...payload,
          name: t("copyName", "{{name}} copy", { name: detail.name }),
          show: false,
          user_visible: false,
          sell: false,
        } as API.CreateSubscribeRequest);
        toast.success(t("copySuccess", "Product copied"));
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [refreshData, t]
  );

  const setLifecycle = useCallback(
    async (product: API.SubscribeItem, lifecycle: ProductLifecycle) => {
      setOperationLoading(true);
      try {
        const detail = await loadSubscribeDetail(product.id);
        const lifecycleValues = {
          selling: { show: true, user_visible: true, sell: true },
          draft: { show: false, user_visible: true, sell: false },
          hidden: { show: false, user_visible: false, sell: false },
        }[lifecycle];
        await updateSubscribe({ ...detail, ...lifecycleValues });
        toast.success(t("lifecycleUpdated", "Sales status updated"));
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
      if (deleteTargets.length === 1) {
        await deleteSubscribe({ id: deleteTargets[0]!.id });
      } else {
        await batchDeleteSubscribe({
          ids: deleteTargets.map((product) => product.id),
        });
      }
      toast.success(t("deleteSuccess", "Product deleted"));
      setDeleteTargets([]);
      setSelectedIds(new Set());
      await refreshData();
    } finally {
      setOperationLoading(false);
    }
  }, [deleteTargets, refreshData, t]);

  const reorderProduct = useCallback(
    async (product: API.SubscribeItem, direction: -1 | 1) => {
      const reordered = moveProduct(products, product.id, direction);
      const previousSort = new Map(
        products.map((item) => [item.id, item.sort])
      );
      const changed = reordered.filter(
        (item) => item.sort !== previousSort.get(item.id)
      );
      if (changed.length === 0) return;
      setOperationLoading(true);
      try {
        await subscribeSort({
          sort: changed.map((item) => ({ id: item.id, sort: item.sort })),
        });
        toast.success(t("sortSuccess", "Order updated"));
        await refreshData();
      } finally {
        setOperationLoading(false);
      }
    },
    [products, refreshData, t]
  );

  const exportComparison = useCallback(() => {
    const header = [
      "name",
      "price",
      "period",
      "traffic_bytes",
      "speed_bps",
      "device_limit",
      "inventory",
      "active_subscriptions",
    ];
    const rows = matrixProducts.map((product) => [
      product.name,
      product.unit_price,
      product.unit_time,
      product.traffic,
      product.speed_limit,
      product.device_limit,
      product.inventory,
      product.sold,
    ]);
    const csv = [header, ...rows]
      .map((row) =>
        row.map((value) => `"${String(value).replaceAll('"', '""')}"`).join(",")
      )
      .join("\n");
    const url = URL.createObjectURL(
      new Blob([csv], { type: "text/csv;charset=utf-8" })
    );
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "alpha-product-comparison.csv";
    anchor.click();
    URL.revokeObjectURL(url);
    toast.success(t("exportSuccess", "Comparison exported"));
  }, [matrixProducts, t]);

  const matrixCell = (product: API.SubscribeItem, row: ProductMatrixRowKey) => {
    switch (row) {
      case "positioning":
        return t(
          `positioning.${getProductPositioning(product)}`,
          getProductPositioning(product)
        );
      case "traffic":
        return (
          <Display type="traffic" unlimited value={product.traffic || 0} />
        );
      case "speed":
        return formatSpeedLimit(product.speed_limit);
      case "devices":
        return (
          <Display type="number" unlimited value={product.device_limit || 0} />
        );
      case "reset":
        return t(`resetCycle.${product.reset_cycle || 0}`, "No reset");
      case "renewal":
        return product.allow_renewal ? (
          <span className="inline-flex items-center gap-1.5 text-emerald-500">
            <Check className="size-3.5" />
            {t("allowed", "Allowed")}
          </span>
        ) : (
          t("notAllowed", "Not allowed")
        );
      case "visibility":
        return (
          <VisibilityIndicators labels={visibilityLabels} product={product} />
        );
      case "inventory":
        return (
          <InventoryValue
            product={product}
            remainingLabel={(count) =>
              t("remainingInventory", "{{count}} remaining", { count })
            }
            unlimitedLabel={t("unlimited", "Unlimited")}
          />
        );
      case "subscriptions":
        return (product.sold || 0).toLocaleString();
    }
  };

  const cellTone = (product: API.SubscribeItem, row: ProductMatrixRowKey) => {
    if (row === "inventory" && isProductLowStock(product)) {
      return "bg-amber-500/[0.08]";
    }
    if (matrixProducts.length < 2) return;
    if (row === "traffic") {
      const max = Math.max(...matrixProducts.map((item) => item.traffic || 0));
      if (
        product.traffic === max &&
        new Set(matrixProducts.map((item) => item.traffic)).size > 1
      ) {
        return "bg-primary/[0.06]";
      }
    }
    if (row === "speed") {
      const max = Math.max(
        ...matrixProducts.map((item) => item.speed_limit || 0)
      );
      if (
        product.speed_limit === max &&
        new Set(matrixProducts.map((item) => item.speed_limit)).size > 1
      ) {
        return "bg-primary/[0.06]";
      }
    }
    if (row === "devices") {
      const max = Math.max(
        ...matrixProducts.map((item) => item.device_limit || 0)
      );
      if (
        product.device_limit === max &&
        new Set(matrixProducts.map((item) => item.device_limit)).size > 1
      ) {
        return "bg-primary/[0.06]";
      }
    }
  };

  const productActions = (product: API.SubscribeItem) => (
    <div className="flex items-center justify-end gap-1">
      <SubscribeForm<API.UpdateSubscribeRequest>
        initialValues={product}
        loadInitialValues={() => loadSubscribeDetail(product.id)}
        loading={operationLoading}
        onSubmit={(values) => updateProduct(product, values)}
        title={t("editProduct", "Edit product")}
        trigger={
          <Button aria-label={t("edit", "Edit")} size="icon-sm" variant="ghost">
            <Pencil className="size-4" />
          </Button>
        }
      />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={t("moreActions", "More actions")}
            size="icon-sm"
            variant="ghost"
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => duplicateProduct(product)}>
            <Copy className="size-4" />
            {t("copy", "Copy")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setLifecycle(product, "selling")}>
            {t("setSelling", "Set on sale")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setLifecycle(product, "draft")}>
            {t("setDraft", "Move to draft")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setLifecycle(product, "hidden")}>
            {t("setHidden", "Hide product")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-destructive focus:text-destructive"
            onClick={() => setDeleteTargets([product])}
          >
            <Trash2 className="size-4" />
            {t("delete", "Delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );

  const renderQueue = (items: API.SubscribeItem[], selectable: boolean) => (
    <div className="overflow-x-auto rounded-lg border">
      <Table className="min-w-[860px]">
        <TableHeader>
          <TableRow>
            {selectable ? (
              <TableHead className="w-10">
                <Checkbox
                  aria-label={t("selectAll", "Select all")}
                  checked={
                    items.length > 0 &&
                    items.every((product) => selectedIds.has(product.id))
                  }
                  onCheckedChange={(checked) =>
                    setSelectedIds(
                      checked
                        ? new Set(items.map((product) => product.id))
                        : new Set()
                    )
                  }
                />
              </TableHead>
            ) : null}
            <TableHead>{t("table.product", "Product")}</TableHead>
            <TableHead>{t("table.price", "Price")}</TableHead>
            <TableHead>{t("table.status", "Status")}</TableHead>
            <TableHead>{t("table.reason", "Notes")}</TableHead>
            <TableHead>{t("table.inventory", "Inventory")}</TableHead>
            <TableHead>{t("table.updated", "Updated")}</TableHead>
            <TableHead className="text-right">
              {t("table.actions", "Actions")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((product, index) => {
            const lifecycle = getProductLifecycle(product);
            const displayTemplate = product.display_template || "standard";
            return (
              <TableRow key={product.id}>
                {selectable ? (
                  <TableCell>
                    <Checkbox
                      aria-label={t("selectProduct", "Select {{name}}", {
                        name: product.name,
                      })}
                      checked={selectedIds.has(product.id)}
                      onCheckedChange={(checked) => {
                        const next = new Set(selectedIds);
                        if (checked) next.add(product.id);
                        else next.delete(product.id);
                        setSelectedIds(next);
                      }}
                    />
                  </TableCell>
                ) : null}
                <TableCell>
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <p className="truncate font-medium">{product.name}</p>
                      {(product.badge || displayTemplate !== "standard") && (
                        <Badge className="shrink-0" variant="secondary">
                          {product.badge ||
                            t(
                              `form.template${displayTemplate[0]?.toUpperCase()}${displayTemplate.slice(1)}`,
                              displayTemplate
                            )}
                        </Badge>
                      )}
                    </div>
                    <p className="text-muted-foreground text-xs tabular-nums">
                      prod_{String(product.id).padStart(3, "0")} ·{" "}
                      {t("form.quantityRange", "{{min}}–{{max}} periods", {
                        min: product.min_quantity || 1,
                        max:
                          product.max_quantity || t("unlimited", "Unlimited"),
                      })}
                    </p>
                  </div>
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  <Display type="currency" value={product.unit_price} />/
                  {t(`form.${product.unit_time}`, product.unit_time)}
                </TableCell>
                <TableCell>
                  <ProductStatus
                    labels={lifecycleLabels}
                    lifecycle={lifecycle}
                  />
                </TableCell>
                <TableCell className="max-w-80 text-muted-foreground text-sm">
                  {t(
                    `reason.${getProductSaleState(product)}`,
                    getProductSaleState(product)
                  )}
                </TableCell>
                <TableCell>
                  <InventoryValue
                    product={product}
                    remainingLabel={(count) =>
                      t("remainingInventory", "{{count}} remaining", { count })
                    }
                    unlimitedLabel={t("unlimited", "Unlimited")}
                  />
                </TableCell>
                <TableCell className="text-muted-foreground text-xs tabular-nums">
                  {formatTimestamp(product.updated_at)}
                </TableCell>
                <TableCell>
                  <div className="flex items-center justify-end gap-1">
                    {sortMode ? (
                      <>
                        <Button
                          aria-label={t("moveUp", "Move up")}
                          disabled={index === 0 || operationLoading}
                          onClick={() => reorderProduct(product, -1)}
                          size="icon-sm"
                          variant="ghost"
                        >
                          <ArrowUp className="size-4" />
                        </Button>
                        <Button
                          aria-label={t("moveDown", "Move down")}
                          disabled={
                            index === items.length - 1 || operationLoading
                          }
                          onClick={() => reorderProduct(product, 1)}
                          size="icon-sm"
                          variant="ghost"
                        >
                          <ArrowDown className="size-4" />
                        </Button>
                      </>
                    ) : null}
                    {productActions(product)}
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );

  return (
    <AdminPage>
      <AdminPageHeader
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              className="gap-2"
              onClick={() => {
                setView("queue");
                setSortMode(true);
              }}
              variant="outline"
            >
              <Settings2 className="size-4" />
              {t("manageOrder", "Manage order")}
            </Button>
            <SubscribeForm<API.CreateSubscribeRequest>
              loading={operationLoading}
              onSubmit={createProduct}
              title={t("createProduct", "Create product")}
              trigger={
                <Button className="gap-2">
                  <Plus className="size-4" />
                  {t("createProduct", "Create product")}
                </Button>
              }
            />
          </div>
        }
        description={t(
          "portfolioDescription",
          "Compare pricing, benefits, and customer availability."
        )}
        eyebrow={t("workspace", "Commerce")}
        title={t("portfolioTitle", "Product portfolio")}
      />

      <section className="flex flex-wrap items-center gap-x-5 gap-y-2 border-b pb-4 text-sm">
        <span className="inline-flex items-center gap-2">
          <Package className="size-4 text-muted-foreground" />
          {t("summary.products", "{{count}} products", {
            count: summary.total,
          })}
        </span>
        <span className="inline-flex items-center gap-2">
          <Circle className="size-2 fill-emerald-500 text-emerald-500" />
          {t("summary.selling", "{{count}} on sale", {
            count: summary.selling,
          })}
        </span>
        <span className="inline-flex items-center gap-2">
          <Users className="size-4 text-muted-foreground" />
          {t("summary.activeSubscriptions", "{{count}} active subscriptions", {
            count: summary.subscriptions,
          })}
        </span>
        <span className="inline-flex items-center gap-2">
          <CircleDollarSign className="size-4 text-muted-foreground" />
          {t("summary.monthlyRevenue", "Monthly list revenue")}
          <strong className="font-semibold">
            {currencySymbol}
            {(summary.monthlyListRevenue / 100).toLocaleString(
              i18n.resolvedLanguage,
              { maximumFractionDigits: 0 }
            )}
          </strong>
        </span>
        {summary.lowStock > 0 ? (
          <span className="inline-flex items-center gap-2 text-amber-500">
            <AlertTriangle className="size-4" />
            {t("summary.lowStock", "{{count}} low-stock products", {
              count: summary.lowStock,
            })}
          </span>
        ) : null}
      </section>

      <section className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <Tabs
          onValueChange={(value) => setView(value as "matrix" | "queue")}
          value={view}
        >
          <TabsList>
            <TabsTrigger value="matrix">
              {t("view.matrix", "Plan matrix")}
            </TabsTrigger>
            <TabsTrigger value="queue">
              {t("view.queue", "Publishing queue")}
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:items-center xl:justify-end">
          <div className="relative w-full sm:max-w-sm">
            <Search className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-3 size-4 text-muted-foreground" />
            <Input
              aria-label={t("searchPlaceholder", "Search products")}
              className="h-10 pl-9"
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("searchPlaceholder", "Search products")}
              value={search}
            />
          </div>
          <Select onValueChange={setLanguage} value={language}>
            <SelectTrigger className="h-10 w-full sm:w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">
                {t("allLanguages", "All languages")}
              </SelectItem>
              <SelectItem value="default">
                {t("defaultLanguage", "Default language")}
              </SelectItem>
              {languageOptions.map((item) => (
                <SelectItem key={item} value={item}>
                  {item}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {view === "matrix" ? (
            <Field
              className="h-10 w-auto shrink-0 rounded-lg border px-3"
              orientation="horizontal"
            >
              <FieldLabel htmlFor="product-differences-only">
                {t("onlyDifferences", "Differences only")}
              </FieldLabel>
              <Switch
                checked={onlyDifferences}
                id="product-differences-only"
                onCheckedChange={setOnlyDifferences}
              />
            </Field>
          ) : null}
        </div>
      </section>

      {view === "matrix" ? (
        <>
          {matrixProducts.length > 0 ? (
            <section className="overflow-x-auto rounded-lg border bg-card">
              <div
                className="grid"
                style={{
                  gridTemplateColumns: `164px repeat(${matrixProducts.length}, minmax(220px, 1fr))`,
                  minWidth: 164 + matrixProducts.length * 220,
                }}
              >
                <div className="flex min-h-20 items-end border-r px-4 py-3 font-medium text-muted-foreground text-sm">
                  {t("matrix.attribute", "Attribute")}
                </div>
                {matrixProducts.map((product) => (
                  <div
                    className="min-w-0 border-r px-4 py-3 last:border-r-0"
                    key={product.id}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate font-semibold text-base">
                          {product.name}
                        </p>
                        <div className="mt-0.5 flex flex-wrap items-center gap-2">
                          <span className="text-muted-foreground text-xs tabular-nums">
                            prod_{String(product.id).padStart(3, "0")}
                          </span>
                          <ProductStatus
                            labels={lifecycleLabels}
                            lifecycle={getProductLifecycle(product)}
                          />
                          {(product.badge ||
                            (product.display_template &&
                              product.display_template !== "standard")) && (
                            <Badge className="h-5" variant="secondary">
                              {product.badge || product.display_template}
                            </Badge>
                          )}
                        </div>
                      </div>
                      {productActions(product)}
                    </div>
                    <p className="mt-2 font-semibold text-lg tabular-nums">
                      <Display type="currency" value={product.unit_price} />
                      <span className="ml-1 font-normal text-muted-foreground text-xs">
                        /{t(`form.${product.unit_time}`, product.unit_time)}
                      </span>
                    </p>
                  </div>
                ))}

                {matrixRows.map((row) => (
                  <div className="contents" key={row}>
                    <div className="flex min-h-10 items-center border-t border-r px-4 font-medium text-sm">
                      {t(`matrix.${row}`, row)}
                    </div>
                    {matrixProducts.map((product) => (
                      <div
                        className={cn(
                          "flex min-h-10 min-w-0 items-center border-t border-r px-4 text-sm last:border-r-0",
                          cellTone(product, row)
                        )}
                        key={`${row}-${product.id}`}
                      >
                        {matrixCell(product, row)}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </section>
          ) : (
            <section className="rounded-lg border px-6 py-16 text-center text-muted-foreground">
              <Package className="mx-auto mb-3 size-7" />
              {t("emptyMatrix", "No on-sale products match the filters")}
            </section>
          )}

          {matrixPageCount > 1 ? (
            <div className="flex justify-end gap-2">
              <Button
                aria-label={t("previousProducts", "Previous products")}
                disabled={matrixPage === 0}
                onClick={() => setMatrixPage((page) => Math.max(0, page - 1))}
                size="icon-sm"
                variant="outline"
              >
                <ArrowLeft className="size-4" />
              </Button>
              <span className="inline-flex h-8 items-center px-2 text-muted-foreground text-xs tabular-nums">
                {matrixPage + 1} / {matrixPageCount}
              </span>
              <Button
                aria-label={t("nextProducts", "Next products")}
                disabled={matrixPage >= matrixPageCount - 1}
                onClick={() =>
                  setMatrixPage((page) =>
                    Math.min(matrixPageCount - 1, page + 1)
                  )
                }
                size="icon-sm"
                variant="outline"
              >
                <ArrowRight className="size-4" />
              </Button>
            </div>
          ) : null}

          {lowStockProducts.length > 0 ? (
            <Alert className="ml-auto w-auto max-w-full">
              <AlertTriangle />
              <AlertTitle>
                {t("lowStockNotice", "Low inventory requires attention")}
              </AlertTitle>
              <AlertDescription className="truncate">
                {lowStockProducts.map((product) => product.name).join(", ")}
              </AlertDescription>
              <Button
                className="col-start-2 mt-1 w-fit"
                onClick={() => {
                  setView("queue");
                  setSearch(lowStockProducts[0]?.name || "");
                }}
                size="sm"
                variant="ghost"
              >
                {t("reviewInventory", "Review inventory")}
              </Button>
            </Alert>
          ) : null}

          <section className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="font-semibold text-base">
                  {t("pendingTitle", "Draft and hidden products")}
                </h2>
                <p className="mt-0.5 text-muted-foreground text-xs">
                  {t(
                    "pendingDescription",
                    "Finish configuration or restore products when they are ready."
                  )}
                </p>
              </div>
              <Badge variant="outline">{pendingProducts.length}</Badge>
            </div>
            {pendingProducts.length > 0 ? (
              renderQueue(pendingProducts, false)
            ) : (
              <div className="rounded-lg border px-4 py-8 text-center text-muted-foreground text-sm">
                {t("pendingEmpty", "No draft or hidden products")}
              </div>
            )}
          </section>

          <div className="flex flex-wrap justify-end gap-2">
            <Button
              className="gap-2"
              disabled={matrixProducts.length === 0}
              onClick={exportComparison}
              variant="outline"
            >
              <Download className="size-4" />
              {t("exportComparison", "Export comparison")}
            </Button>
            <Button
              className="gap-2"
              onClick={() => {
                setView("queue");
                setSortMode(true);
              }}
            >
              <Settings2 className="size-4" />
              {t("adjustPortfolio", "Adjust portfolio")}
            </Button>
          </div>
        </>
      ) : (
        <section className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="font-semibold text-base">
                {t("queueTitle", "Publishing queue")}
              </h2>
              <p className="mt-0.5 text-muted-foreground text-xs">
                {t(
                  "queueDescription",
                  "Manage sales status, customer visibility, and storefront order."
                )}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {selectedIds.size > 0 ? (
                <Button
                  onClick={() =>
                    setDeleteTargets(
                      products.filter((product) => selectedIds.has(product.id))
                    )
                  }
                  size="sm"
                  variant="destructive"
                >
                  <Trash2 className="size-4" />
                  {t("deleteSelected", "Delete selected")}
                  <span className="tabular-nums">{selectedIds.size}</span>
                </Button>
              ) : null}
              <Button
                onClick={() => setSortMode((enabled) => !enabled)}
                size="sm"
                variant={sortMode ? "secondary" : "outline"}
              >
                <ArrowUp className="size-4" />
                {t("sortMode", "Reorder")}
              </Button>
            </div>
          </div>
          {filteredProducts.length > 0 ? (
            renderQueue(filteredProducts, true)
          ) : (
            <div className="rounded-lg border px-4 py-16 text-center text-muted-foreground">
              {storeLoading
                ? t("loading", "Loading products...")
                : t("emptyProducts", "No products match the filters")}
            </div>
          )}
        </section>
      )}

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
                ? t("confirmBatchDelete", "Delete selected products?")
                : t("confirmDelete", "Delete this product?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "deleteWarning",
                "This action cannot be undone. Products with active subscriptions may be protected by the server."
              )}
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
