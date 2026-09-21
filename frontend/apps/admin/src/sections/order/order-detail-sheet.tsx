import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { getOrderDetail } from "@workspace/ui/services/admin/order";
import { AlertCircle, History, LoaderCircle, RefreshCw } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Display } from "@/components/display";
import { formatDate } from "@/utils/common";
import { UserDetail } from "../user/user-detail";

function DetailValue({
  label,
  children,
  mono = false,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className={mono ? "break-all font-mono text-sm" : "text-sm"}>
        {children ?? "—"}
      </dd>
    </div>
  );
}

function DetailSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="border-b py-5 last:border-b-0">
      <h3 className="mb-4 font-semibold text-sm">{title}</h3>
      {children}
    </section>
  );
}

export function OrderDetailSheet({
  onOpenChange,
  open,
  orderId,
  renderStatus,
}: {
  onOpenChange: (open: boolean) => void;
  open: boolean;
  orderId: number | null;
  renderStatus: (
    order: Pick<API.OrderDetail, "id" | "status">,
    onUpdated: (status: number) => void
  ) => React.ReactNode;
}) {
  const { t } = useTranslation("order");
  const [detail, setDetail] = useState<API.OrderDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const requestSequence = useRef(0);

  const loadDetail = useCallback(async () => {
    if (!orderId) return;
    const sequence = ++requestSequence.current;
    setDetail(null);
    setLoading(true);
    setFailed(false);
    try {
      const response = await getOrderDetail(
        { id: orderId },
        { skipErrorHandler: true }
      );
      if (sequence !== requestSequence.current) return;
      const order = response.data.data;
      if (order?.id !== orderId || !order.order_no) {
        throw new Error("Missing or mismatched order detail");
      }
      setDetail(order);
    } catch {
      if (sequence === requestSequence.current) {
        setDetail(null);
        setFailed(true);
      }
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [orderId]);

  useEffect(() => {
    if (open && orderId) loadDetail();
    if (!open) {
      requestSequence.current += 1;
      setDetail(null);
      setFailed(false);
    }
    return () => {
      requestSequence.current += 1;
    };
  }, [loadDetail, open, orderId]);

  const typeLabel = detail ? t(`type.${detail.type}`, String(detail.type)) : "";
  const productName = detail
    ? detail.product_name ||
      detail.membership_plan?.name ||
      detail.subscribe?.name ||
      typeLabel
    : "";

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent className="flex w-[640px] max-w-full flex-col gap-0 p-0 md:max-w-2xl">
        <SheetHeader className="border-b px-6 py-5 text-left">
          <div className="flex flex-wrap items-start justify-between gap-3 pr-8">
            <div className="min-w-0">
              <SheetTitle>{t("detailTitle", "Order detail")}</SheetTitle>
              <SheetDescription className="mt-1 break-all font-mono">
                {detail?.order_no ||
                  (failed
                    ? t("detailLoadFailed", "Unable to load order detail")
                    : t("detailLoading", "Loading order data"))}
              </SheetDescription>
            </div>
            {detail ? <Badge variant="secondary">{typeLabel}</Badge> : null}
          </div>
        </SheetHeader>

        <ScrollArea className="min-h-0 flex-1">
          <div className="px-6">
            {loading ? (
              <div aria-busy="true" className="flex flex-col gap-4 py-6">
                {Array.from({ length: 7 }, (_, index) => (
                  <Skeleton className="h-12 w-full" key={index} />
                ))}
              </div>
            ) : null}

            {failed && !loading ? (
              <Alert className="my-6" variant="destructive">
                <AlertCircle />
                <AlertTitle>
                  {t("detailLoadFailed", "Unable to load order detail")}
                </AlertTitle>
                <AlertDescription className="mt-2 flex flex-wrap items-center justify-between gap-3">
                  <span>
                    {t(
                      "detailLoadFailedDescription",
                      "The order may have changed or the connection is unavailable."
                    )}
                  </span>
                  <Button onClick={loadDetail} size="sm" variant="outline">
                    <RefreshCw />
                    {t("retry", "Retry")}
                  </Button>
                </AlertDescription>
              </Alert>
            ) : null}

            {detail && !loading ? (
              <>
                <DetailSection title={t("detailOverview", "Overview")}>
                  <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                    <DetailValue label={t("product", "Product")}>
                      {productName}
                    </DetailValue>
                    <DetailValue label={t("user", "User")}>
                      <UserDetail id={detail.user_id} />
                    </DetailValue>
                    <DetailValue label={t("status.0", "Status")}>
                      {renderStatus(detail, loadDetail)}
                    </DetailValue>
                    <DetailValue label={t("quantity", "Quantity")}>
                      <span className="tabular-nums">{detail.quantity}</span>
                    </DetailValue>
                    <DetailValue label={t("createdAt", "Created at")}>
                      {formatDate(detail.created_at)}
                    </DetailValue>
                    <DetailValue label={t("updateTime", "Updated at")}>
                      {formatDate(detail.updated_at)}
                    </DetailValue>
                  </dl>
                </DetailSection>

                <DetailSection
                  title={t("financialBreakdown", "Financial breakdown")}
                >
                  <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                    {detail.subscribe?.unit_price !== undefined &&
                    (detail.type === 1 || detail.type === 2) ? (
                      <DetailValue
                        label={t(
                          "transactionUnitPrice",
                          "Unit price at purchase"
                        )}
                      >
                        <Display
                          type="currency"
                          value={detail.subscribe.unit_price}
                        />
                      </DetailValue>
                    ) : null}
                    <DetailValue label={t("basePrice", "List price total")}>
                      <Display type="currency" value={detail.price} />
                    </DetailValue>
                    <DetailValue label={t("discount", "Discount")}>
                      <Display type="currency" value={detail.discount} />
                    </DetailValue>
                    <DetailValue label={t("couponDiscount", "Coupon discount")}>
                      <Display type="currency" value={detail.coupon_discount} />
                    </DetailValue>
                    <DetailValue label={t("feeAmount", "Fee")}>
                      <Display type="currency" value={detail.fee_amount} />
                    </DetailValue>
                    <DetailValue label={t("giftAmount", "Gift amount")}>
                      <Display type="currency" value={detail.gift_amount} />
                    </DetailValue>
                    <DetailValue
                      label={t(
                        "commissionDeduction",
                        "Commission balance used"
                      )}
                    >
                      <Display
                        type="currency"
                        value={detail.commission_deduction}
                      />
                    </DetailValue>
                    <DetailValue label={t("commission", "Commission")}>
                      <Display type="currency" value={detail.commission} />
                    </DetailValue>
                    <DetailValue label={t("total", "Total")}>
                      <strong>
                        <Display type="currency" value={detail.amount} />
                      </strong>
                    </DetailValue>
                  </dl>
                  {detail.type === 1 || detail.type === 2 ? (
                    <Alert className="mt-5">
                      <History />
                      <AlertTitle>
                        {t("priceSnapshotTitle", "Immutable price snapshot")}
                      </AlertTitle>
                      <AlertDescription>
                        {detail.type === 2
                          ? t(
                              "renewalPriceSnapshotDescription",
                              "This renewal used the product price and rules current when the order was created. Later product changes do not alter this order."
                            )
                          : t(
                              "purchasePriceSnapshotDescription",
                              "This order retains the product price and rules used at checkout. Later product changes do not alter this order."
                            )}
                      </AlertDescription>
                    </Alert>
                  ) : null}
                </DetailSection>

                <DetailSection title={t("paymentDetail", "Payment")}>
                  <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <DetailValue label={t("method", "Payment method")}>
                      {detail.payment?.name ||
                        detail.payment?.platform ||
                        detail.method}
                    </DetailValue>
                    <DetailValue label={t("paymentPlatform", "Platform")}>
                      {detail.payment?.platform || detail.method}
                    </DetailValue>
                    <DetailValue
                      label={t("tradeNo", "Transaction number")}
                      mono
                    >
                      {detail.trade_no}
                    </DetailValue>
                    <DetailValue label={t("coupon", "Coupon")} mono>
                      {detail.coupon}
                    </DetailValue>
                  </dl>
                </DetailSection>
              </>
            ) : null}
          </div>
        </ScrollArea>

        <div className="flex items-center justify-end gap-2 border-t px-6 py-4">
          <Button
            disabled={loading || !orderId}
            onClick={loadDetail}
            variant="outline"
          >
            {loading ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <RefreshCw />
            )}
            {t("refreshDetail", "Refresh detail")}
          </Button>
          <Button onClick={() => onOpenChange(false)} variant="secondary">
            {t("closeDetail", "Close")}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
