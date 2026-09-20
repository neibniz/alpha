import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@workspace/ui/components/avatar";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Switch } from "@workspace/ui/components/switch";
import { ConfirmButton } from "@/components/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import {
  createPaymentMethod,
  deletePaymentMethod,
  getPaymentMethod,
  getPaymentMethodList,
  updatePaymentMethod,
} from "@workspace/ui/services/admin/payment";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { Copy, Pencil, Plus, Trash2 } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Display } from "@/components/display";
import PaymentForm from "./payment-form";

export default function PaymentTable() {
  const { t } = useTranslation("payment");
  const [formPending, setFormPending] = useState<number | "create" | null>(
    null
  );
  const [rowPendingIds, setRowPendingIds] = useState<Set<number>>(new Set());
  const ref = useRef<ProTableActions>(null);

  const setRowPending = (id: number, pending: boolean) => {
    setRowPendingIds((current) => {
      const next = new Set(current);
      if (pending) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  return (
    <ProTable<
      API.PaymentMethodListItem,
      { search?: string; platform?: string; enable?: string }
    >
      action={ref}
      actions={{
        render: (row) => [
          <PaymentForm<API.UpdatePaymentMethodRequest>
            initialValues={row}
            isEdit
            key="edit"
            loadInitialValues={async () => {
              try {
                const { data } = await getPaymentMethod({ id: row.id });
                return {
                  ...row,
                  ...(data.data || {}),
                };
              } catch {
                toast.error(
                  t("detailLoadFailed", "Failed to load payment method detail")
                );
                return;
              }
            }}
            loading={formPending === row.id}
            onSubmit={async (values) => {
              setFormPending(row.id);
              try {
                await updatePaymentMethod({
                  ...row,
                  ...values,
                });
                toast.success(t("updateSuccess", "Updated successfully"));
                ref.current?.refresh();
                setFormPending(null);
                return true;
              } catch {
                setFormPending(null);
                return false;
              }
            }}
            title={t("editPayment", "Edit Payment Method")}
            trigger={
              <Button
                aria-label={t("edit", "Edit")}
                size="icon"
                title={t("edit", "Edit")}
                variant="ghost"
              >
                <Pencil />
              </Button>
            }
          />,
          <ConfirmButton
            cancelText={t("cancel", "Cancel")}
            confirmText={t("confirm", "Confirm")}
            description={t(
              "deleteWarning",
              "Are you sure you want to delete this payment method? This action cannot be undone."
            )}
            key="delete"
            onConfirm={async () => {
              setRowPending(row.id, true);
              try {
                await deletePaymentMethod({ id: row.id });
                toast.success(t("deleteSuccess", "Deleted successfully"));
                ref.current?.refresh();
              } finally {
                setRowPending(row.id, false);
              }
            }}
            title={t("confirmDelete", "Confirm Delete")}
            trigger={
              <Button
                aria-label={t("delete", "Delete")}
                disabled={rowPendingIds.has(row.id) || row.id === -1}
                size="icon"
                title={t("delete", "Delete")}
                variant="ghost"
              >
                <Trash2 className="text-destructive" />
              </Button>
            }
          />,
          <Button
            aria-label={t("copy", "Copy")}
            disabled={
              rowPendingIds.has(row.id) ||
              row.platform.toLocaleLowerCase() === "balance"
            }
            key="copy"
            onClick={async () => {
              setRowPending(row.id, true);
              try {
                const { data } = await getPaymentMethod({ id: row.id });
                if (!data.data) {
                  throw new Error("payment detail is empty");
                }
                const { id: _id, ...params } = data.data;
                await createPaymentMethod({
                  ...params,
                  config: params.config || {},
                  enable: false,
                });
                toast.success(t("copySuccess", "Copied successfully"));
                ref.current?.refresh();
                return true;
              } catch {
                toast.error(
                  t("copyFailed", "Payment method could not be copied")
                );
                return false;
              } finally {
                setRowPending(row.id, false);
              }
            }}
            size="icon"
            title={
              row.platform.toLocaleLowerCase() === "balance"
                ? t(
                    "balanceCopyDisabled",
                    "The built-in balance payment method cannot be copied"
                  )
                : t("copy", "Copy")
            }
            variant="ghost"
          >
            <Copy />
          </Button>,
        ],
        batchRender(rows) {
          return [
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteWarning",
                "Are you sure you want to delete this payment method? This action cannot be undone."
              )}
              key="delete"
              onConfirm={async () => {
                const deletable = rows.filter((row) => row.id !== -1);
                const results = await Promise.allSettled(
                  deletable.map((row) => deletePaymentMethod({ id: row.id }))
                );
                const failed = results.filter(
                  (result) => result.status === "rejected"
                ).length;
                if (failed > 0) {
                  toast.error(
                    t("batchDeletePartial", {
                      count: failed,
                      defaultValue:
                        "{{count}} payment methods could not be deleted",
                    })
                  );
                } else {
                  toast.success(t("deleteSuccess", "Deleted successfully"));
                }
                ref.current?.refresh();
              }}
              title={t("confirmDelete", "Confirm Delete")}
              trigger={
                <Button variant="destructive">
                  {t("batchDelete", "Batch Delete")}
                </Button>
              }
            />,
          ];
        },
      }}
      columns={[
        {
          accessorKey: "enable",
          header: t("enable", "Enable"),
          cell: ({ row }) => (
            <Switch
              aria-label={t("togglePayment", {
                defaultValue: "Toggle {{name}}",
                name: row.original.name,
              })}
              checked={Boolean(row.getValue("enable"))}
              disabled={rowPendingIds.has(row.original.id)}
              onCheckedChange={async (checked) => {
                const id = row.original.id;
                setRowPending(id, true);
                ref.current?.updateRow({ ...row.original, enable: checked });
                try {
                  await updatePaymentMethod({ id, enable: checked });
                  toast.success(
                    checked
                      ? t("enableSuccess", "Payment method enabled")
                      : t("disableSuccess", "Payment method disabled")
                  );
                } catch {
                  ref.current?.updateRow({ ...row.original, enable: !checked });
                } finally {
                  setRowPending(id, false);
                }
              }}
            />
          ),
        },
        {
          accessorKey: "icon",
          header: t("icon", "Icon"),
          cell: ({ row }) => {
            const icon = row.getValue("icon") as string;
            return (
              <Avatar className="h-8 w-8">
                {icon ? (
                  <AvatarImage alt={row.getValue("name")} src={icon} />
                ) : null}
                <AvatarFallback>
                  {(row.getValue("name") as string)?.charAt(0) || "?"}
                </AvatarFallback>
              </Avatar>
            );
          },
        },
        {
          accessorKey: "name",
          header: t("name", "Name"),
        },
        {
          accessorKey: "platform",
          header: t("platform", "Platform"),
          cell: ({ row }) => (
            <Badge variant="outline">{row.original.platform}</Badge>
          ),
        },
        {
          accessorKey: "notify_url",
          header: t("notify_url", "Notify URL"),
          cell: ({ row }) => (
            <code
              className="block max-w-72 truncate text-xs"
              title={row.original.notify_url}
            >
              {row.original.notify_url || "--"}
            </code>
          ),
        },
        {
          accessorKey: "fee",
          header: t("handlingFee", "Handling Fee"),
          cell: ({ row }) => {
            const feeMode = row.original.fee_mode;
            if (feeMode === 1) {
              return <Badge>{row.original.fee_percent}%</Badge>;
            }
            if (feeMode === 2) {
              return (
                <Badge>
                  <Display type="currency" value={row.original.fee_amount} />
                </Badge>
              );
            }
            if (feeMode === 3) {
              return (
                <Badge variant="outline">
                  {row.original.fee_percent}% +{" "}
                  <Display type="currency" value={row.original.fee_amount} />
                </Badge>
              );
            }
            return "--";
          },
        },
      ]}
      header={{
        title: t("paymentManagement", "Payment Management"),
        toolbar: (
          <PaymentForm<API.CreatePaymentMethodRequest>
            loading={formPending === "create"}
            onSubmit={async (values) => {
              setFormPending("create");
              try {
                await createPaymentMethod({
                  ...values,
                  enable: false,
                });
                toast.success(t("createSuccess", "Created successfully"));
                ref.current?.refresh();
                setFormPending(null);
                return true;
              } catch {
                setFormPending(null);
                return false;
              }
            }}
            title={t("createPayment", "Add Payment Method")}
            trigger={
              <Button>
                <Plus />
                {t("create", "Add Payment Method")}
              </Button>
            }
          />
        ),
      }}
      params={[
        {
          key: "platform",
          placeholder: t("platform", "Platform"),
          options: ["Stripe", "AlipayF2F", "EPay", "CryptoSaaS"].map(
            (value) => ({
              label: value,
              value,
            })
          ),
        },
        {
          key: "enable",
          placeholder: t("status", "Status"),
          options: [
            { label: t("enabled", "Enabled"), value: "true" },
            { label: t("disabled", "Disabled"), value: "false" },
          ],
        },
        {
          key: "search",
          placeholder: t("searchPlaceholder", "Enter search terms"),
        },
      ]}
      request={async (pagination, filter) => {
        const { data } = await getPaymentMethodList({
          ...pagination,
          search: filter.search,
          platform: filter.platform,
          enable:
            filter.enable === "true"
              ? true
              : filter.enable === "false"
                ? false
                : undefined,
        });
        return normalizePageResult(data?.data);
      }}
    />
  );
}
