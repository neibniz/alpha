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
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@workspace/ui/components/hover-card";
import { Separator } from "@workspace/ui/components/separator";
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
  getOrderList,
  updateOrderStatus,
} from "@workspace/ui/services/admin/order";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { ChevronDown, Eye, LoaderCircle } from "lucide-react";
import { type MouseEvent, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Display } from "@/components/display";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { useSubscribe } from "@/stores/subscribe";
import { formatDate } from "@/utils/common";
import { UserDetail } from "../user/user-detail";
import { OrderDetailSheet } from "./order-detail-sheet";
import {
  type AdminOrderListFilter,
  adminOrderStatusTargets,
  toAdminOrderListRequest,
} from "./order-status-data";

type OrderFilter = AdminOrderListFilter;

type OrderRow = API.OrderListItem;

type OrderStatusOption = {
  value: number;
  label: string;
  dotClassName: string;
};

function OrderStatusControl({
  order,
  optionByValue,
  onUpdated,
}: {
  order: Pick<OrderRow, "id" | "status">;
  optionByValue: Map<number, OrderStatusOption>;
  onUpdated: (status: number) => void;
}) {
  const { t } = useTranslation("order");
  const [targetStatus, setTargetStatus] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);
  const current = optionByValue.get(order.status);
  const target = targetStatus ? optionByValue.get(targetStatus) : undefined;
  const targets = adminOrderStatusTargets(order.status);

  const description =
    targetStatus === 2
      ? t(
          "statusConfirmPaidDescription",
          "Marking this order as paid starts subscription activation. Confirm the payment before continuing."
        )
      : targetStatus === 1
        ? t(
            "statusConfirmReopenDescription",
            "The order will return to pending and can be paid again."
          )
        : t(
            "statusConfirmDescription",
            "Change the order from {{current}} to {{target}}?",
            {
              current: current?.label || String(order.status),
              target: target?.label || String(targetStatus || ""),
            }
          );

  async function confirmStatusChange(event: MouseEvent<HTMLButtonElement>) {
    event.preventDefault();
    if (!targetStatus) return;
    setSaving(true);
    try {
      await updateOrderStatus(
        { id: order.id, status: targetStatus },
        { skipErrorHandler: true }
      );
      toast.success(t("statusUpdateSuccess", "Order status updated"));
      setTargetStatus(null);
      onUpdated(targetStatus);
    } catch {
      toast.error(t("statusUpdateFailed", "Failed to update order status"));
    } finally {
      setSaving(false);
    }
  }

  if (!targets.length) {
    return (
      <Badge className="gap-1.5" variant="outline">
        <span
          aria-hidden="true"
          className={cn("size-2 rounded-full", current?.dotClassName)}
        />
        {current?.label || t(`status.${order.status}`)}
      </Badge>
    );
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={t("changeStatus", "Change order status")}
            className="h-8 gap-1.5"
            size="sm"
            variant="outline"
          >
            <span
              aria-hidden="true"
              className={cn("size-2 rounded-full", current?.dotClassName)}
            />
            {current?.label || t(`status.${order.status}`)}
            <ChevronDown className="size-3.5 text-muted-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>
            {t("changeStatusTo", "Change status to")}
          </DropdownMenuLabel>
          {targets.map((status) => {
            const option = optionByValue.get(status);
            return (
              <DropdownMenuItem
                key={status}
                onSelect={() => setTargetStatus(status)}
                variant={[3, 4].includes(status) ? "destructive" : "default"}
              >
                <span
                  aria-hidden="true"
                  className={cn("size-2 rounded-full", option?.dotClassName)}
                />
                {option?.label || t(`status.${status}`)}
              </DropdownMenuItem>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog
        onOpenChange={(open) => {
          if (!(open || saving)) setTargetStatus(null);
        }}
        open={targetStatus !== null}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("statusConfirmTitle", "Confirm status update")}
            </AlertDialogTitle>
            <AlertDialogDescription>{description}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={saving}>
              {t("cancel", "Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              className={cn({
                "bg-destructive text-white hover:bg-destructive/90":
                  targetStatus === 3 || targetStatus === 4,
              })}
              disabled={saving}
              onClick={confirmStatusChange}
            >
              {saving ? <LoaderCircle className="animate-spin" /> : null}
              {t("confirm", "Confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

export default function Order() {
  const { t } = useTranslation("order");
  const [detailOrderId, setDetailOrderId] = useState<number | null>(null);

  const statusOptions = useMemo(
    () => [
      {
        value: 1,
        label: t("status.1", "Pending"),
        dotClassName: "bg-amber-500",
      },
      {
        value: 2,
        label: t("status.2", "Paid"),
        dotClassName: "bg-emerald-500",
      },
      {
        value: 3,
        label: t("status.3", "Cancelled"),
        dotClassName: "bg-muted-foreground",
      },
      {
        value: 4,
        label: t("status.4", "Closed"),
        dotClassName: "bg-red-500",
      },
      {
        value: 5,
        label: t("status.5", "Completed"),
        dotClassName: "bg-primary",
      },
    ],
    [t]
  );

  const typeOptions = useMemo(
    () => [
      { value: 1, label: t("type.1", "New Purchase") },
      { value: 2, label: t("type.2", "Renewal") },
      { value: 3, label: t("type.3", "Reset Traffic") },
      { value: 4, label: t("type.4", "Recharge") },
      { value: 5, label: t("type.5", "Membership Card") },
    ],
    [t]
  );
  const statusOptionByValue = useMemo(
    () => new Map(statusOptions.map((option) => [option.value, option])),
    [statusOptions]
  );
  const typeLabelByValue = useMemo(
    () => new Map(typeOptions.map((option) => [option.value, option.label])),
    [typeOptions]
  );
  const statusFilterOptions = useMemo(
    () =>
      statusOptions.map((item) => ({
        label: item.label,
        value: String(item.value),
      })),
    [statusOptions]
  );

  const ref = useRef<ProTableActions>(null);

  const { subscribes, getSubscribeName } = useSubscribe();
  const subscribeFilterOptions = useMemo(
    () =>
      subscribes?.map((item) => ({
        label: item.name!,
        value: String(item.id),
      })),
    [subscribes]
  );
  const getTypeLabel = (type: number) =>
    typeLabelByValue.get(type) || t(`type.${type}`);
  const getOrderProductName = (order: OrderRow) => {
    if (order.product_name) {
      return order.type === 4 ? getTypeLabel(order.type) : order.product_name;
    }
    if (order.type === 4 || order.type === 5) {
      return getTypeLabel(order.type);
    }
    const name = getSubscribeName(order.subscribe_id);
    return name ? `${name} × ${order.quantity}` : "";
  };

  return (
    <>
      <AdminPage>
        <AdminPageHeader
          title={t("managementTitle", "Orders")}
          description={t(
            "pageDescription",
            "Payments, fulfillment and order history."
          )}
        />
        <ProTable<OrderRow, OrderFilter>
          action={ref}
          actions={{
            columnSize: 64,
            render: (order) => [
              <Tooltip key="detail">
                <TooltipTrigger asChild>
                  <Button
                    aria-label={t("viewDetail", "View order detail")}
                    onClick={() => setDetailOrderId(order.id)}
                    size="icon-sm"
                    variant="ghost"
                  >
                    <Eye />
                  </Button>
                </TooltipTrigger>
                <TooltipContent sideOffset={6}>
                  {t("viewDetail", "View order detail")}
                </TooltipContent>
              </Tooltip>,
            ],
          }}
          columns={[
            {
              accessorKey: "order_no",
              header: t("orderNumber", "Order Number"),
            },
            {
              accessorKey: "type",
              header: t("type.0", "Type"),
              cell: ({ row }) => {
                const type = row.getValue("type") as number;
                return getTypeLabel(type);
              },
            },
            {
              accessorKey: "subscribe_id",
              header: t("subscribe", "Subscribe"),
              cell: ({ row }) => getOrderProductName(row.original as OrderRow),
            },
            {
              accessorKey: "amount",
              header: t("amount", "Amount"),
              cell: ({ row }) => {
                const order = row.original as OrderRow;
                return (
                  <HoverCard>
                    <HoverCardTrigger asChild>
                      <Button className="p-0" variant="link">
                        <Display type="currency" value={order.amount} />
                      </Button>
                    </HoverCardTrigger>
                    <HoverCardContent>
                      <div className="grid gap-3">
                        {order.trade_no && (
                          <>
                            <div className="font-semibold">
                              {t("tradeNo", "Transaction Number")}
                            </div>
                            <span className="text-muted-foreground">
                              {order.trade_no}
                            </span>
                            <Separator className="my-2" />
                          </>
                        )}
                        <ul className="grid gap-3">
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {order.type === 5
                                ? t("membershipPrice", "Membership Price")
                                : t("subscribePrice", "Subscription Price")}
                            </span>
                            <span>
                              <Display type="currency" value={order.price} />
                            </span>
                          </li>
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("discount", "Discount Amount")}
                            </span>
                            <span>
                              <Display type="currency" value={order.discount} />
                            </span>
                          </li>
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("couponDiscount", "Coupon Discount")}
                            </span>
                            <span>
                              <Display
                                type="currency"
                                value={order.coupon_discount}
                              />
                            </span>
                          </li>
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t(
                                "commissionDeduction",
                                "Commission balance used"
                              )}
                            </span>
                            <span>
                              <Display
                                type="currency"
                                value={order.commission_deduction}
                              />
                            </span>
                          </li>
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("giftAmount", "Gift amount")}
                            </span>
                            <span>
                              <Display
                                type="currency"
                                value={order.gift_amount}
                              />
                            </span>
                          </li>
                          <li className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("feeAmount", "Fee Amount")}
                            </span>
                            <span>
                              <Display
                                type="currency"
                                value={order.fee_amount}
                              />
                            </span>
                          </li>
                          <li className="flex items-center justify-between font-semibold">
                            <span className="text-muted-foreground">
                              {t("total", "Total")}
                            </span>
                            <span>
                              <Display type="currency" value={order.amount} />
                            </span>
                          </li>
                        </ul>
                      </div>
                      <Separator className="my-4" />
                      <ul className="grid gap-3">
                        <li className="flex items-center justify-between">
                          <span className="text-muted-foreground">
                            {t("method", "Payment Method")}
                          </span>
                          <span>
                            {order.payment?.name || order.payment?.platform}
                          </span>
                        </li>
                      </ul>
                    </HoverCardContent>
                  </HoverCard>
                );
              },
            },
            {
              accessorKey: "user_id",
              header: t("user", "User"),
              cell: ({ row }) => {
                const order = row.original as OrderRow;
                return <UserDetail id={order.user_id} />;
              },
            },
            {
              accessorKey: "updated_at",
              header: t("updateTime", "Update Time"),
              cell: ({ row }) => {
                const order = row.original as OrderRow;
                return formatDate(order.updated_at);
              },
            },
            {
              accessorKey: "status",
              header: t("status.0", "Status"),
              cell: ({ row }) => (
                <OrderStatusControl
                  onUpdated={() => ref.current?.refresh()}
                  optionByValue={statusOptionByValue}
                  order={row.original as OrderRow}
                />
              ),
            },
          ]}
          params={[
            {
              key: "status",
              placeholder: t("status.0", "Status"),
              options: statusFilterOptions,
            },
            {
              key: "subscribe_id",
              placeholder: `${t("subscribe", "Subscribe")}`,
              options: subscribeFilterOptions,
            },
            {
              key: "search",
              placeholder: t(
                "searchPlaceholder",
                "Search order, transaction, or coupon"
              ),
            },
            {
              key: "user_id",
              placeholder: t("userIdPlaceholder", "User ID"),
              options: undefined,
            },
          ]}
          request={async (pagination, filter) => {
            const { data } = await getOrderList(
              toAdminOrderListRequest(pagination, filter)
            );
            return normalizePageResult(data.data);
          }}
        />
      </AdminPage>
      <OrderDetailSheet
        onOpenChange={(open) => {
          if (!open) setDetailOrderId(null);
        }}
        open={detailOrderId !== null}
        orderId={detailOrderId}
        renderStatus={(order, onUpdated) => (
          <OrderStatusControl
            onUpdated={(status) => {
              onUpdated(status);
              ref.current?.refresh();
            }}
            optionByValue={statusOptionByValue}
            order={order}
          />
        )}
      />
    </>
  );
}
