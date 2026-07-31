"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { ButtonGroup } from "@workspace/ui/components/button-group";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import {
  Field,
  FieldDescription,
  FieldLabel,
} from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Input } from "@workspace/ui/components/input";
import { Separator } from "@workspace/ui/components/separator";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import {
  CircleAlert,
  Crown,
  Gift,
  Minus,
  Plus,
  RefreshCcw,
  RotateCcw,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { OrderCheckoutStatus } from "@/components/orders/order-checkout-status";
import { PaymentMethodSelector } from "@/components/orders/payment-method-selector";
import {
  checkoutOrder,
  createOrderRequestId,
  createRenewalOrder,
  createResetTrafficOrder,
  getPaymentMethods,
  getRenewalQuote,
  getResetTrafficQuote,
  isMembershipRequiredError,
} from "@/lib/client-api";
import {
  formatBytes,
  formatCurrency,
  formatSpeedLimitMbps,
} from "@/lib/format";
import type {
  CheckoutResult,
  RenewalQuote,
  ResetTrafficQuote,
  SiteConfig,
  Subscription,
} from "@/lib/models";

export type SubscriptionActionMode = "renewal" | "reset";

function money(cents: number, symbol: string) {
  return formatCurrency(cents / 100, symbol);
}

function durationLabel(quantity: number, unit: string) {
  const labels: Record<string, string> = {
    day: "天",
    month: "个月",
    year: "年",
  };
  return `${quantity} ${labels[unit.toLowerCase()] || unit}`;
}

function BillingSummary({
  quote,
  symbol,
}: Readonly<{
  quote: RenewalQuote | ResetTrafficQuote;
  symbol: string;
}>) {
  const renewal = "discount" in quote;
  const adjustments = [
    renewal
      ? { label: "套餐优惠", negative: true, value: quote.discount }
      : undefined,
    renewal
      ? { label: "优惠券", negative: true, value: quote.couponDiscount }
      : undefined,
    {
      label: "佣金余额抵扣",
      negative: true,
      value: quote.commissionDeduction,
    },
    { label: "赠送余额抵扣", negative: true, value: quote.giftAmount },
    { label: "支付手续费", negative: false, value: quote.feeAmount },
  ].filter(
    (item): item is { label: string; negative: boolean; value: number } =>
      Boolean(item?.value)
  );
  return (
    <dl className="space-y-3 text-sm">
      <div className="flex justify-between gap-4">
        <dt className="text-muted-foreground">原价</dt>
        <dd className="font-medium tabular-nums">
          {money(quote.price, symbol)}
        </dd>
      </div>
      {adjustments.map((item) => (
        <div
          className={`flex justify-between gap-4 ${item.negative ? "text-teal-700" : ""}`}
          key={item.label}
        >
          <dt>{item.label}</dt>
          <dd className="font-medium tabular-nums">
            {item.negative ? "-" : "+"}
            {money(item.value, symbol)}
          </dd>
        </div>
      ))}
      <Separator />
      <div className="flex items-end justify-between gap-4">
        <dt className="font-medium">应付金额</dt>
        <dd className="font-semibold text-xl tabular-nums">
          {money(quote.amount, symbol)}
        </dd>
      </div>
    </dl>
  );
}

export function SubscriptionActionsDialog({
  config,
  mode,
  onOpenChange,
  open,
  subscription,
}: Readonly<{
  config?: SiteConfig;
  mode: SubscriptionActionMode;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  subscription: Subscription;
}>) {
  const queryClient = useQueryClient();
  const router = useRouter();
  const [quantity, setQuantity] = useState(0);
  const [payment, setPayment] = useState(0);
  const [coupon, setCoupon] = useState("");
  const [appliedCoupon, setAppliedCoupon] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [checkout, setCheckout] = useState<CheckoutResult>();
  const [orderNo, setOrderNo] = useState("");
  const [requestId, setRequestId] = useState("");
  const symbol = config?.currencySymbol || "¥";

  useEffect(() => {
    if (!open) return;
    setQuantity(0);
    setPayment(0);
    setCoupon("");
    setAppliedCoupon("");
    setCheckout(undefined);
    setOrderNo("");
    setRequestId(createOrderRequestId());
  }, [mode, open, subscription.id]);

  useEffect(() => {
    if (!open || orderNo) return;
    setRequestId(createOrderRequestId());
  }, [appliedCoupon, mode, open, orderNo, payment, quantity, subscription.id]);

  const payments = useQuery({
    enabled: open,
    queryFn: getPaymentMethods,
    queryKey: ["alpha-payment-methods"],
    staleTime: 60_000,
  });

  useEffect(() => {
    if (!open || payment || !payments.data?.length) return;
    setPayment(payments.data[0]?.id || 0);
  }, [open, payment, payments.data]);

  const renewalQuote = useQuery({
    enabled: open && mode === "renewal" && !checkout,
    queryFn: () =>
      getRenewalQuote({
        coupon: appliedCoupon,
        payment,
        quantity,
        user_subscribe_id: subscription.id,
      }),
    queryKey: [
      "alpha-renewal-quote",
      subscription.id,
      quantity,
      payment,
      appliedCoupon,
    ],
    retry: false,
  });

  const resetQuote = useQuery({
    enabled: open && mode === "reset" && !checkout,
    queryFn: () =>
      getResetTrafficQuote({
        payment,
        user_subscribe_id: subscription.id,
      }),
    queryKey: ["alpha-reset-quote", subscription.id, payment],
    retry: false,
  });

  const quote = mode === "renewal" ? renewalQuote.data : resetQuote.data;
  const quoteLoading =
    mode === "renewal" ? renewalQuote.isFetching : resetQuote.isFetching;
  const quoteError = mode === "renewal" ? renewalQuote.error : resetQuote.error;

  useEffect(() => {
    if (mode !== "renewal" || !renewalQuote.data?.quantity) return;
    if (quantity === renewalQuote.data.quantity) return;
    setQuantity(renewalQuote.data.quantity);
  }, [mode, quantity, renewalQuote.data?.quantity]);

  const durationOptions = useMemo(() => {
    if (mode !== "renewal") return [];
    const product = renewalQuote.data?.product;
    const minimum = Math.max(1, product?.minQuantity || 1);
    const maximum = product?.maxQuantity || 0;
    const values = new Set<number>([minimum]);
    for (const item of product?.discounts || subscription.discounts) {
      if (item.quantity < minimum) continue;
      if (maximum > 0 && item.quantity > maximum) continue;
      values.add(item.quantity);
    }
    if (quantity >= minimum && (maximum <= 0 || quantity <= maximum)) {
      values.add(quantity);
    }
    return [...values].sort((a, b) => a - b);
  }, [mode, quantity, renewalQuote.data?.product, subscription.discounts]);

  const paymentRequired = Boolean(quote && quote.amount > 0);
  const canSubmit =
    Boolean(quote) &&
    Boolean(requestId) &&
    !quoteLoading &&
    !quoteError &&
    !submitting &&
    (!paymentRequired || payment !== 0);

  const openMembership = () => {
    onOpenChange(false);
    const next = encodeURIComponent(
      `/dashboard?subscription=${subscription.id}`
    );
    router.push(`/membership?next=${next}`);
  };

  const submit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const createdOrderNo =
        mode === "renewal"
          ? await createRenewalOrder({
              coupon: appliedCoupon,
              payment,
              quantity: renewalQuote.data?.quantity || quantity,
              request_id: requestId,
              user_subscribe_id: subscription.id,
            })
          : await createResetTrafficOrder({
              payment,
              request_id: requestId,
              user_subscribe_id: subscription.id,
            });
      const checkoutResult = await checkoutOrder(createdOrderNo);
      setOrderNo(createdOrderNo);
      setCheckout(checkoutResult);
    } catch (error) {
      if (mode === "renewal" && isMembershipRequiredError(error)) {
        openMembership();
        toast.info("续费前请先开通会员");
      } else {
        toast.error(error instanceof Error ? error.message : "订单提交失败");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const close = () => {
    onOpenChange(false);
  };

  const complete = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["alpha-subscriptions"] }),
      queryClient.invalidateQueries({
        queryKey: ["alpha-subscription-scopes"],
      }),
      queryClient.invalidateQueries({ queryKey: ["alpha-traffic"] }),
      queryClient.invalidateQueries({ queryKey: ["alpha-orders"] }),
      queryClient.invalidateQueries({
        queryKey: ["alpha-membership-card"],
      }),
    ]);
  };

  const used = subscription.uploadBytes + subscription.downloadBytes;
  const remaining = Math.max(0, subscription.trafficBytes - used);
  const currentProduct = quote?.product;
  const quoteUnavailable =
    quoteError instanceof Error ? quoteError.message : "报价暂不可用";
  const membershipRequired =
    mode === "renewal" && isMembershipRequiredError(quoteError);

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-4xl">
        {checkout && orderNo ? (
          <OrderCheckoutStatus
            checkout={checkout}
            kind={mode}
            onClose={close}
            onComplete={complete}
            orderNo={orderNo}
          />
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>
                {mode === "renewal" ? "续费订阅" : "重置流量"}
              </DialogTitle>
              <DialogDescription>
                {mode === "renewal"
                  ? "按商品当前价格、周期和优惠规则计算，原订单价格不会沿用。"
                  : "清零当前已用流量，不改变套餐容量与到期时间。"}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-6 md:grid-cols-[0.9fr_1.1fr]">
              <section className="space-y-5 rounded-lg border bg-muted/20 p-5">
                <header>
                  <Badge variant="secondary">
                    {mode === "renewal" ? "续费套餐" : "重置对象"}
                  </Badge>
                  <h3 className="mt-3 font-semibold text-lg">
                    {currentProduct?.name || subscription.planName}
                  </h3>
                </header>
                <dl className="grid grid-cols-3 divide-x rounded-lg border bg-background py-3 text-center">
                  <div className="min-w-0 px-2">
                    <dt className="text-muted-foreground text-xs">套餐流量</dt>
                    <dd className="mt-1 truncate font-medium text-sm">
                      {formatBytes(
                        currentProduct?.trafficBytes ||
                          subscription.trafficBytes
                      )}
                    </dd>
                  </div>
                  <div className="min-w-0 px-2">
                    <dt className="text-muted-foreground text-xs">速度上限</dt>
                    <dd className="mt-1 truncate font-medium text-sm">
                      {formatSpeedLimitMbps(
                        currentProduct?.speedLimitMbps ||
                          subscription.speedLimitMbps
                      )}
                    </dd>
                  </div>
                  <div className="min-w-0 px-2">
                    <dt className="text-muted-foreground text-xs">设备数</dt>
                    <dd className="mt-1 truncate font-medium text-sm">
                      {currentProduct?.deviceLimit ||
                        subscription.deviceLimit ||
                        "不限"}{" "}
                      台
                    </dd>
                  </div>
                </dl>

                {mode === "reset" ? (
                  <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 rounded-lg border bg-background p-4">
                    <div className="min-w-0">
                      <span className="block text-muted-foreground text-xs">
                        当前已用
                      </span>
                      <strong className="mt-1 block truncate font-medium text-sm">
                        {formatBytes(used, 1)}
                      </strong>
                      <small className="block truncate text-muted-foreground text-xs">
                        剩余 {formatBytes(remaining, 1)}
                      </small>
                    </div>
                    <RefreshCcw
                      aria-hidden="true"
                      className="text-muted-foreground"
                    />
                    <div className="text-right">
                      <span className="block text-muted-foreground text-xs">
                        重置后已用
                      </span>
                      <strong className="mt-1 block font-medium text-teal-700">
                        0 B
                      </strong>
                    </div>
                  </div>
                ) : null}

                {quote ? (
                  <BillingSummary quote={quote} symbol={symbol} />
                ) : quoteError ? (
                  <Alert variant="destructive">
                    {membershipRequired ? <Crown /> : <CircleAlert />}
                    <AlertTitle>
                      {membershipRequired
                        ? "续费前需要有效会员"
                        : "报价暂不可用"}
                    </AlertTitle>
                    <AlertDescription className="flex flex-col items-start gap-2">
                      <span>
                        {membershipRequired
                          ? "开通会员后即可继续为当前订阅续费。"
                          : quoteUnavailable}
                      </span>
                      {membershipRequired ? (
                        <Button
                          className="h-auto px-0 py-0"
                          onClick={openMembership}
                          type="button"
                          variant="link"
                        >
                          前往会员中心
                        </Button>
                      ) : null}
                    </AlertDescription>
                  </Alert>
                ) : (
                  <div className="space-y-3">
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-4/5" />
                    <Skeleton className="h-8 w-full" />
                  </div>
                )}
              </section>

              <section className="space-y-5">
                {mode === "renewal" ? (
                  <>
                    <Field>
                      <div className="flex items-center justify-between gap-3">
                        <FieldLabel>续费周期</FieldLabel>
                        <FieldDescription>
                          {durationLabel(
                            renewalQuote.data?.quantity || quantity || 1,
                            currentProduct?.unitTime || subscription.unitTime
                          )}
                        </FieldDescription>
                      </div>
                      <ToggleGroup
                        className="flex w-full flex-wrap"
                        onValueChange={(value) => {
                          if (value) setQuantity(Number(value));
                        }}
                        type="single"
                        value={String(quantity)}
                        variant="outline"
                      >
                        {durationOptions.map((value) => {
                          const discount = (
                            currentProduct?.discounts || subscription.discounts
                          ).find((item) => item.quantity === value)?.discount;
                          return (
                            <ToggleGroupItem
                              className="h-auto flex-1 flex-col items-start gap-0.5 py-2"
                              key={value}
                              value={String(value)}
                            >
                              <strong className="font-medium text-xs">
                                {durationLabel(
                                  value,
                                  currentProduct?.unitTime ||
                                    subscription.unitTime
                                )}
                              </strong>
                              <small className="font-normal text-[11px] text-muted-foreground">
                                {discount && discount < 100
                                  ? `优惠 ${100 - discount}%`
                                  : "标准价格"}
                              </small>
                            </ToggleGroupItem>
                          );
                        })}
                      </ToggleGroup>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="renewal-quantity">
                        周期数量
                      </FieldLabel>
                      <ButtonGroup>
                        <Button
                          aria-label="减少一个周期"
                          disabled={
                            quantity <= (currentProduct?.minQuantity || 1)
                          }
                          onClick={() =>
                            setQuantity((value) => Math.max(1, value - 1))
                          }
                          title="减少一个周期"
                          type="button"
                          size="icon"
                          variant="outline"
                        >
                          <Minus aria-hidden="true" />
                        </Button>
                        <Input
                          aria-label="续费周期数量"
                          className="w-20 text-center tabular-nums"
                          id="renewal-quantity"
                          max={currentProduct?.maxQuantity || 1000}
                          min={currentProduct?.minQuantity || 1}
                          onChange={(event) =>
                            setQuantity(
                              Math.max(
                                currentProduct?.minQuantity || 1,
                                Number(event.target.value) || 1
                              )
                            )
                          }
                          step={1}
                          type="number"
                          value={quantity || 1}
                        />
                        <Button
                          aria-label="增加一个周期"
                          disabled={
                            Boolean(currentProduct?.maxQuantity) &&
                            quantity >= (currentProduct?.maxQuantity || 0)
                          }
                          onClick={() =>
                            setQuantity((value) => (value || 1) + 1)
                          }
                          title="增加一个周期"
                          type="button"
                          size="icon"
                          variant="outline"
                        >
                          <Plus aria-hidden="true" />
                        </Button>
                      </ButtonGroup>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="renewal-coupon">优惠券</FieldLabel>
                      <InputGroup>
                        <InputGroupInput
                          id="renewal-coupon"
                          onChange={(event) => setCoupon(event.target.value)}
                          placeholder="选填"
                          value={coupon}
                        />
                        <InputGroupAddon align="inline-end">
                          <InputGroupButton
                            disabled={coupon.trim() === appliedCoupon}
                            onClick={() => setAppliedCoupon(coupon.trim())}
                            size="sm"
                          >
                            应用
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </Field>
                  </>
                ) : (
                  <Alert>
                    <RefreshCcw />
                    <AlertTitle>仅清零流量计数</AlertTitle>
                    <AlertDescription>
                      不会延长有效期，也不会改变速度或设备限制。
                    </AlertDescription>
                  </Alert>
                )}

                {paymentRequired ? (
                  payments.isLoading ? (
                    <div className="flex items-center gap-2 rounded-lg border p-4 text-muted-foreground text-sm">
                      <Spinner />
                      正在加载支付方式
                    </div>
                  ) : payments.isError || !payments.data?.length ? (
                    <Alert variant="destructive">
                      <CircleAlert />
                      <AlertTitle>暂无可用支付方式</AlertTitle>
                      <AlertDescription>
                        <Button
                          className="h-auto px-0 py-0"
                          onClick={() => payments.refetch()}
                          type="button"
                          variant="link"
                        >
                          重新加载
                        </Button>
                      </AlertDescription>
                    </Alert>
                  ) : (
                    <PaymentMethodSelector
                      idPrefix="subscription-payment"
                      methods={payments.data}
                      onChange={setPayment}
                      value={payment}
                    />
                  )
                ) : quote ? (
                  <Alert>
                    <Gift />
                    <AlertTitle>本次无需支付</AlertTitle>
                    <AlertDescription>确认后立即处理。</AlertDescription>
                  </Alert>
                ) : null}
              </section>
            </div>

            <DialogFooter>
              <Button onClick={close} variant="outline">
                取消
              </Button>
              <Button disabled={!canSubmit} onClick={submit}>
                {submitting || quoteLoading ? (
                  <Spinner />
                ) : mode === "renewal" ? (
                  <RotateCcw />
                ) : (
                  <RefreshCcw />
                )}
                {mode === "renewal" ? "确认续费" : "确认重置"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
