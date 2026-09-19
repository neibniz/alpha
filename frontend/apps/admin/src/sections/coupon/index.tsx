import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Switch } from "@workspace/ui/components/switch";
import { ConfirmButton } from "@workspace/ui/composed/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import {
  batchDeleteCoupon,
  createCoupon,
  deleteCoupon,
  getCoupon,
  getCouponList,
  updateCoupon,
} from "@workspace/ui/services/admin/coupon";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { Display } from "@/components/display";
import { useSubscribe } from "@/stores/subscribe";
import { formatDate } from "@/utils/common";
import CouponForm from "./coupon-form";
import {
  buildCreateCouponPayload,
  buildUpdateCouponPayload,
  couponLifecycle,
} from "./coupon-form-data";

export default function Coupon() {
  const { t } = useTranslation("coupon");
  const [createLoading, setCreateLoading] = useState(false);
  const [batchLoading, setBatchLoading] = useState(false);
  const [pendingIds, setPendingIds] = useState<Set<number>>(() => new Set());
  const { subscribes } = useSubscribe();
  const tableRef = useRef<ProTableActions>(null);

  const setPending = (id: number, pending: boolean) => {
    setPendingIds((current) => {
      const next = new Set(current);
      if (pending) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const loadCouponDetail = async (id: number) => {
    const { data } = await getCoupon({ id });
    if (!data.data) throw new Error("missing coupon detail");
    return data.data;
  };

  const loadCouponDetailWithToast = async (id: number) => {
    try {
      return await loadCouponDetail(id);
    } catch (error) {
      toast.error(t("loadFailed", "Failed to load coupon details"));
      throw error;
    }
  };

  const toggleCoupon = async (row: API.CouponListItem, enabled: boolean) => {
    setPending(row.id, true);
    try {
      const detail = await loadCouponDetail(row.id);
      await updateCoupon({ ...detail, enable: enabled });
      tableRef.current?.refresh();
      toast.success(
        enabled
          ? t("enabledSuccess", "Coupon enabled")
          : t("disabledSuccess", "Coupon disabled")
      );
    } catch {
      toast.error(t("updateFailed", "Update failed"));
    } finally {
      setPending(row.id, false);
    }
  };

  const lifecycleLabel = (row: API.CouponListItem) => {
    const lifecycle = couponLifecycle(row);
    const variants = {
      active: "default",
      scheduled: "secondary",
      disabled: "outline",
      expired: "destructive",
      exhausted: "destructive",
    } as const;
    return (
      <Badge variant={variants[lifecycle]}>
        {t(`status.${lifecycle}`, lifecycle)}
      </Badge>
    );
  };

  return (
    <AdminPage>
      <AdminPageHeader
        description={t(
          "description",
          "Create discount rules, control product scope and redemption limits, and monitor lifecycle state."
        )}
        eyebrow={t("workspace", "Commerce")}
        title={t("title", "Coupon management")}
      />
      <ProTable<API.CouponListItem, { subscribe?: number; search?: string }>
        action={tableRef}
        actions={{
          columnSize: 96,
          render: (row) => [
            <CouponForm
              initialValues={row}
              key="edit"
              loadInitialValues={() => loadCouponDetailWithToast(row.id)}
              loading={pendingIds.has(row.id)}
              onSubmit={async (values) => {
                setPending(row.id, true);
                try {
                  await updateCoupon(buildUpdateCouponPayload(values));
                  toast.success(t("updateSuccess", "Coupon updated"));
                  tableRef.current?.refresh();
                  return true;
                } catch {
                  toast.error(t("updateFailed", "Update failed"));
                  return false;
                } finally {
                  setPending(row.id, false);
                }
              }}
              title={t("editCoupon", "Edit Coupon")}
              trigger={
                <Button
                  aria-label={t("edit", "Edit")}
                  disabled={pendingIds.has(row.id) || batchLoading}
                  size="icon-sm"
                  title={t("edit", "Edit")}
                  variant="ghost"
                >
                  <Pencil className="size-4" />
                </Button>
              }
            />,
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteWarning",
                "Once deleted, data cannot be recovered."
              )}
              key="delete"
              onConfirm={async () => {
                setPending(row.id, true);
                try {
                  await deleteCoupon({ id: row.id });
                  toast.success(t("deleteSuccess", "Coupon deleted"));
                  tableRef.current?.refresh();
                } catch {
                  toast.error(t("deleteFailed", "Delete failed"));
                } finally {
                  setPending(row.id, false);
                }
              }}
              title={t("confirmDelete", "Delete coupon?")}
              trigger={
                <Button
                  aria-label={t("delete", "Delete")}
                  disabled={pendingIds.has(row.id) || batchLoading}
                  size="icon-sm"
                  title={t("delete", "Delete")}
                  variant="ghost"
                >
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              }
            />,
          ],
          batchRender: (rows) => [
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "batchDeleteWarning",
                "Delete the selected {{count}} coupons? This cannot be undone.",
                { count: rows.length }
              )}
              key="batch-delete"
              onConfirm={async () => {
                setBatchLoading(true);
                try {
                  await batchDeleteCoupon({ ids: rows.map((item) => item.id) });
                  toast.success(t("deleteSuccess", "Coupons deleted"));
                  tableRef.current?.reset();
                } catch {
                  toast.error(t("deleteFailed", "Delete failed"));
                } finally {
                  setBatchLoading(false);
                }
              }}
              title={t("confirmDelete", "Delete coupons?")}
              trigger={
                <Button disabled={batchLoading} variant="destructive">
                  <Trash2 className="size-4" />
                  {t("batchDelete", "Delete selected")}
                </Button>
              }
            />,
          ],
        }}
        columns={[
          {
            accessorKey: "enable",
            header: t("status.title", "Status"),
            size: 170,
            cell: ({ row }) => (
              <div className="flex items-center gap-3">
                <Switch
                  aria-label={`${t("enable", "Enable")} ${row.original.name}`}
                  checked={row.original.enable}
                  disabled={pendingIds.has(row.original.id) || batchLoading}
                  onCheckedChange={(checked) =>
                    toggleCoupon(row.original, checked)
                  }
                />
                {lifecycleLabel(row.original)}
              </div>
            ),
          },
          {
            accessorKey: "name",
            header: t("name", "Coupon"),
            size: 260,
            cell: ({ row }) => (
              <div className="min-w-0">
                <div className="truncate font-medium">{row.original.name}</div>
                <code className="mt-1 block truncate text-muted-foreground text-xs">
                  {row.original.code}
                </code>
              </div>
            ),
          },
          {
            accessorKey: "discount",
            header: t("discount", "Discount"),
            size: 150,
            cell: ({ row }) =>
              row.original.type === 1 ? (
                <span className="font-semibold tabular-nums">
                  {row.original.discount}%
                </span>
              ) : (
                <span className="font-semibold tabular-nums">
                  <Display type="currency" value={row.original.discount} />
                </span>
              ),
          },
          {
            accessorKey: "count",
            header: t("usage", "Usage"),
            size: 180,
            cell: ({ row }) => {
              const total = row.original.count;
              const remaining =
                total === 0
                  ? t("unlimited", "Unlimited")
                  : Math.max(total - row.original.used_count, 0);
              return (
                <div className="text-sm">
                  <div className="font-medium tabular-nums">
                    {row.original.used_count} / {total || "∞"}
                  </div>
                  <div className="text-muted-foreground text-xs">
                    {t("remainingTimes", "Remaining")}: {remaining}
                  </div>
                </div>
              );
            },
          },
          {
            accessorKey: "expire_time",
            header: t("validityPeriod", "Validity Period"),
            size: 250,
            cell: ({ row }) => (
              <div className="text-xs leading-5">
                <div>{formatDate(row.original.start_time)}</div>
                <div className="text-muted-foreground">
                  {t("until", "until")} {formatDate(row.original.expire_time)}
                </div>
              </div>
            ),
          },
        ]}
        header={{
          title: t("directory", "Coupon directory"),
          toolbar: (
            <CouponForm
              loading={createLoading}
              onSubmit={async (values) => {
                setCreateLoading(true);
                try {
                  await createCoupon(buildCreateCouponPayload(values));
                  toast.success(t("createSuccess", "Coupon created"));
                  tableRef.current?.refresh();
                  return true;
                } catch {
                  toast.error(t("createFailed", "Create failed"));
                  return false;
                } finally {
                  setCreateLoading(false);
                }
              }}
              title={t("createCoupon", "Create Coupon")}
              trigger={
                <Button size="sm">
                  <Plus className="size-4" />
                  {t("create", "Create")}
                </Button>
              }
            />
          ),
        }}
        params={[
          {
            key: "subscribe",
            placeholder: t("subscribe", "All subscriptions"),
            options: subscribes?.map((item) => ({
              label: item.name!,
              value: String(item.id),
            })),
          },
          {
            key: "search",
            placeholder: t("searchPlaceholder", "Search name or code"),
          },
        ]}
        request={async (pagination, filters) => {
          const { data } = await getCouponList({
            ...pagination,
            ...filters,
          });
          return normalizePageResult(data.data);
        }}
      />
    </AdminPage>
  );
}
