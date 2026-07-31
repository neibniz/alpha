"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@workspace/ui/components/empty";
import { Progress } from "@workspace/ui/components/progress";
import { Separator } from "@workspace/ui/components/separator";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  CalendarDays,
  Check,
  CircleAlert,
  Crown,
  Gauge,
  Gift,
  ReceiptText,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { OrderCheckoutStatus } from "@/components/orders/order-checkout-status";
import { PaymentMethodSelector } from "@/components/orders/payment-method-selector";
import {
  checkoutOrder,
  createMembershipOrder,
  createOrderRequestId,
  getMembershipCard,
  getMembershipQuote,
  getPaymentMethods,
  getSiteConfig,
} from "@/lib/client-api";
import { formatCurrency, formatDate } from "@/lib/format";
import {
  membershipDurationLabel,
  membershipUsagePercent,
  safeMembershipReturnPath,
} from "@/lib/membership";
import type {
  CheckoutResult,
  MembershipCard,
  MembershipQuote,
} from "@/lib/models";

function money(cents: number, symbol: string) {
  return formatCurrency(cents / 100, symbol);
}

function MembershipOrderSummary({
  card,
  quote,
  symbol,
}: Readonly<{
  card: MembershipCard;
  quote?: MembershipQuote;
  symbol: string;
}>) {
  return (
    <dl className="space-y-3 rounded-lg border bg-muted/25 p-4 text-sm">
      <div className="flex justify-between gap-4">
        <dt className="text-muted-foreground">会员方案</dt>
        <dd className="text-right font-medium">{card.plan.name}</dd>
      </div>
      <div className="flex justify-between gap-4">
        <dt className="text-muted-foreground">有效期</dt>
        <dd className="font-medium">
          {membershipDurationLabel(
            card.plan.durationValue,
            card.plan.durationUnit
          )}
        </dd>
      </div>
      <div className="flex justify-between gap-4">
        <dt className="text-muted-foreground">方案价格</dt>
        <dd className="font-medium">
          {money(quote?.price ?? card.plan.unitPrice, symbol)}
        </dd>
      </div>
      {quote?.commissionDeduction ? (
        <div className="flex justify-between gap-4 text-teal-700">
          <dt>佣金余额抵扣</dt>
          <dd className="font-medium">
            -{money(quote.commissionDeduction, symbol)}
          </dd>
        </div>
      ) : null}
      {quote?.giftAmount ? (
        <div className="flex justify-between gap-4 text-teal-700">
          <dt>赠送余额抵扣</dt>
          <dd className="font-medium">-{money(quote.giftAmount, symbol)}</dd>
        </div>
      ) : null}
      {quote?.feeAmount ? (
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">支付手续费</dt>
          <dd className="font-medium">+{money(quote.feeAmount, symbol)}</dd>
        </div>
      ) : null}
      <Separator />
      <div className="flex items-end justify-between gap-4">
        <dt className="font-medium">应付</dt>
        <dd className="font-semibold text-xl tabular-nums">
          {quote ? money(quote.amount, symbol) : "计算中"}
        </dd>
      </div>
      <p className="text-muted-foreground text-xs">
        服务端依次使用佣金余额、赠送余额，剩余金额再交由所选支付方式处理。
      </p>
    </dl>
  );
}

function MembershipCheckoutDialog({
  card,
  onOpenChange,
  open,
  returnPath,
  symbol,
}: Readonly<{
  card: MembershipCard;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  returnPath: string;
  symbol: string;
}>) {
  const queryClient = useQueryClient();
  const router = useRouter();
  const [payment, setPayment] = useState(0);
  const [requestId, setRequestId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [checkout, setCheckout] = useState<CheckoutResult>();
  const [orderNo, setOrderNo] = useState("");
  const [completed, setCompleted] = useState(false);

  const payments = useQuery({
    enabled: open,
    queryFn: getPaymentMethods,
    queryKey: ["alpha-payment-methods"],
    staleTime: 60_000,
  });

  useEffect(() => {
    if (!open) return;
    setPayment(0);
    setCheckout(undefined);
    setOrderNo("");
    setCompleted(false);
  }, [open]);

  useEffect(() => {
    if (!open || payment || !payments.data?.length) return;
    setPayment(payments.data[0]?.id || 0);
  }, [open, payment, payments.data]);

  useEffect(() => {
    if (!open || orderNo) return;
    setRequestId(createOrderRequestId());
  }, [open, orderNo, payment]);

  const quote = useQuery({
    enabled: open && card.enabled,
    queryFn: () => getMembershipQuote({ payment }),
    queryKey: ["alpha-membership-quote", card.plan.id, payment],
    retry: false,
  });

  const paymentRequired = Boolean(quote.data && quote.data.amount > 0);
  const canSubmit =
    Boolean(quote.data && requestId) &&
    !quote.isFetching &&
    !submitting &&
    (!paymentRequired || payment !== 0);

  const close = () => {
    onOpenChange(false);
    if (completed && returnPath !== "/membership") {
      router.push(returnPath);
    }
  };

  const submit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const createdOrderNo = await createMembershipOrder({
        payment,
        request_id: requestId,
      });
      const checkoutResult = await checkoutOrder(createdOrderNo);
      setOrderNo(createdOrderNo);
      setCheckout(checkoutResult);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "会员订单创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        {checkout && orderNo ? (
          <OrderCheckoutStatus
            checkout={checkout}
            kind="membership"
            onClose={close}
            onComplete={async () => {
              await Promise.all([
                queryClient.invalidateQueries({
                  queryKey: ["alpha-membership-card"],
                }),
                queryClient.invalidateQueries({
                  queryKey: ["alpha-current-user"],
                }),
                queryClient.invalidateQueries({ queryKey: ["alpha-orders"] }),
              ]);
              setCompleted(true);
            }}
            orderNo={orderNo}
          />
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>
                {card.isMember ? "续期会员" : "开通会员"}
              </DialogTitle>
              <DialogDescription>
                {card.isMember
                  ? "新有效期会从当前到期时间继续顺延。"
                  : "支付完成后即可购买和续费订阅套餐。"}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-6 sm:grid-cols-[0.95fr_1.05fr]">
              <MembershipOrderSummary
                card={card}
                quote={quote.data}
                symbol={symbol}
              />
              <div className="space-y-5">
                {quote.isError ? (
                  <Alert variant="destructive">
                    <CircleAlert />
                    <AlertTitle>报价暂不可用</AlertTitle>
                    <AlertDescription>
                      <span>
                        {quote.error instanceof Error
                          ? quote.error.message
                          : "请稍后重试"}
                      </span>
                      <Button
                        className="h-auto px-0 py-0"
                        onClick={() => quote.refetch()}
                        type="button"
                        variant="link"
                      >
                        重新计算
                      </Button>
                    </AlertDescription>
                  </Alert>
                ) : paymentRequired ? (
                  payments.isError || !payments.data?.length ? (
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
                      idPrefix="membership-payment"
                      methods={payments.data}
                      onChange={setPayment}
                      value={payment}
                    />
                  )
                ) : quote.data ? (
                  <Alert>
                    <Gift />
                    <AlertTitle>本次无需额外支付</AlertTitle>
                    <AlertDescription>
                      提交后由服务端直接完成会员开通。
                    </AlertDescription>
                  </Alert>
                ) : (
                  <div className="flex items-center gap-2 rounded-lg border p-4 text-muted-foreground text-sm">
                    <Spinner />
                    正在计算当前价格
                  </div>
                )}

                <div className="space-y-3 border-y py-4 text-sm">
                  <div className="flex items-start gap-3">
                    <Check className="mt-0.5 size-4 shrink-0" />
                    <span>支付完成后自动激活，无需人工处理。</span>
                  </div>
                  <div className="flex items-start gap-3">
                    <Check className="mt-0.5 size-4 shrink-0" />
                    <span>续期从当前有效期末尾顺延，不会损失剩余时间。</span>
                  </div>
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button onClick={close} variant="outline">
                取消
              </Button>
              <Button disabled={!canSubmit} onClick={submit}>
                {submitting ? <Spinner /> : null}
                {card.isMember ? "确认续期" : "确认开通"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function UsageItem({
  detail,
  icon: Icon,
  label,
  progress,
}: Readonly<{
  detail: string;
  icon: typeof Gauge;
  label: string;
  progress?: number;
}>) {
  return (
    <div className="flex min-h-36 flex-col gap-4 p-5 sm:p-6">
      <div className="flex items-center gap-3">
        <span className="flex size-9 items-center justify-center rounded-md border bg-background">
          <Icon className="size-4" />
        </span>
        <strong className="font-medium text-sm">{label}</strong>
      </div>
      <p className="text-muted-foreground text-sm">{detail}</p>
      {progress === undefined ? null : (
        <Progress className="mt-auto h-1.5" value={progress} />
      )}
    </div>
  );
}

export function MembershipCenter() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const autoOpened = useRef(false);
  const returnPath = safeMembershipReturnPath(searchParams.get("next"));
  const card = useQuery({
    queryFn: getMembershipCard,
    queryKey: ["alpha-membership-card"],
    staleTime: 30_000,
  });
  const config = useQuery({
    queryFn: getSiteConfig,
    queryKey: ["alpha-site-config"],
    staleTime: 300_000,
  });

  useEffect(() => {
    if (
      autoOpened.current ||
      !searchParams.get("next") ||
      !card.data?.enabled ||
      card.data.isMember
    ) {
      return;
    }
    autoOpened.current = true;
    setCheckoutOpen(true);
  }, [card.data, searchParams]);

  if (card.isLoading) {
    return (
      <div className="mx-auto grid w-full max-w-6xl gap-5 px-4 py-8 sm:px-6 lg:px-7">
        <Skeleton className="h-20 rounded-lg" />
        <div className="grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
          <Skeleton className="h-64 rounded-lg" />
          <Skeleton className="h-64 rounded-lg" />
        </div>
      </div>
    );
  }

  if (card.isError || !card.data) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6">
        <Alert variant="destructive">
          <CircleAlert />
          <AlertTitle>会员状态暂不可用</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>无法读取当前会员配置，请稍后重试。</span>
            <Button onClick={() => card.refetch()} size="sm" variant="outline">
              重新加载
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!card.data.enabled) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-10 sm:px-6 lg:px-7">
        <Empty className="min-h-[420px] border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Crown />
            </EmptyMedia>
            <EmptyTitle>当前不需要会员</EmptyTitle>
            <EmptyDescription>
              会员体系已由管理员关闭，你可以直接购买和续费订阅套餐。
            </EmptyDescription>
          </EmptyHeader>
          <Button onClick={() => router.push("/products")}>浏览套餐</Button>
        </Empty>
      </div>
    );
  }

  const membership = card.data;
  const policy = membership.plan.policy;
  const hasPolicy =
    policy.monthlyConsumptionLimitEnabled ||
    policy.monthlyOrderLimitEnabled ||
    policy.subscriptionExpiryLimitEnabled;
  const symbol = config.data?.currencySymbol || "¥";

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-7 lg:py-9">
      <header className="mb-7 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 text-muted-foreground text-xs">
            <Crown className="size-4" />
            账户权益
          </div>
          <h1 className="font-semibold text-3xl tracking-tight">会员中心</h1>
          <p className="mt-2 max-w-2xl text-muted-foreground text-sm leading-relaxed">
            查看会员有效期、订阅购买资格和本月权益使用情况。
          </p>
        </div>
        <Button onClick={() => setCheckoutOpen(true)}>
          <Sparkles />
          {membership.isMember ? "续期会员" : "开通会员"}
        </Button>
      </header>

      {!membership.isMember ? (
        <Alert className="mb-5">
          <CircleAlert />
          <AlertTitle>当前账户没有有效会员</AlertTitle>
          <AlertDescription>
            开通会员后即可购买新套餐或为现有订阅续费。
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
        <Card className="rounded-lg shadow-none">
          <CardHeader className="gap-4">
            <div className="flex items-start justify-between gap-4">
              <span className="flex size-11 items-center justify-center rounded-lg border bg-muted/40">
                <ShieldCheck />
              </span>
              <Badge variant={membership.isMember ? "default" : "outline"}>
                {membership.isMember ? "有效" : "未开通"}
              </Badge>
            </div>
            <div>
              <CardTitle>{membership.plan.name}</CardTitle>
              <CardDescription className="mt-2 leading-relaxed">
                {membership.plan.description || "用于解锁订阅购买和续费资格。"}
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent className="grid gap-4 border-t pt-5 sm:grid-cols-2">
            <div>
              <p className="text-muted-foreground text-xs">会员状态</p>
              <p className="mt-1 font-medium">
                {membership.isMember ? "正常使用中" : "等待开通"}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground text-xs">当前到期时间</p>
              <p className="mt-1 font-medium tabular-nums">
                {membership.expiredAt
                  ? formatDate(membership.expiredAt)
                  : "尚未开通"}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-lg shadow-none">
          <CardHeader>
            <CardDescription>当前方案</CardDescription>
            <CardTitle className="flex items-baseline gap-2">
              <span className="text-3xl tabular-nums">
                {money(membership.plan.unitPrice, symbol)}
              </span>
              <span className="font-normal text-muted-foreground text-sm">
                /{" "}
                {membershipDurationLabel(
                  membership.plan.durationValue,
                  membership.plan.durationUnit
                )}
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 border-t pt-5 text-sm">
            <div className="flex items-center gap-3">
              <CalendarDays className="size-4 text-muted-foreground" />
              <span>续期会从现有有效期结束后继续顺延</span>
            </div>
            <div className="flex items-center gap-3">
              <ReceiptText className="size-4 text-muted-foreground" />
              <span>价格与权益在创建订单时写入快照</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <section className="mt-7">
        <div className="mb-4 flex items-end justify-between gap-4">
          <div>
            <h2 className="font-semibold text-lg">本月权益</h2>
            <p className="mt-1 text-muted-foreground text-sm">
              数据与服务端实际下单限制保持一致。
            </p>
          </div>
          <Button
            aria-label="刷新会员权益"
            onClick={() =>
              queryClient.invalidateQueries({
                queryKey: ["alpha-membership-card"],
              })
            }
            size="icon"
            variant="outline"
          >
            <Gauge />
          </Button>
        </div>

        {hasPolicy ? (
          <div className="grid divide-y rounded-lg border sm:grid-cols-3 sm:divide-x sm:divide-y-0">
            {policy.monthlyConsumptionLimitEnabled ? (
              <UsageItem
                detail={`${money(
                  membership.usage.monthlyConsumption,
                  symbol
                )} / ${money(policy.monthlyConsumptionLimit, symbol)}`}
                icon={ReceiptText}
                label="消费额度"
                progress={membershipUsagePercent(
                  membership.usage.monthlyConsumption,
                  policy.monthlyConsumptionLimit
                )}
              />
            ) : null}
            {policy.monthlyOrderLimitEnabled ? (
              <UsageItem
                detail={`${membership.usage.monthlyOrderCount} / ${policy.monthlyOrderLimit} 笔`}
                icon={Gauge}
                label="订单数量"
                progress={membershipUsagePercent(
                  membership.usage.monthlyOrderCount,
                  policy.monthlyOrderLimit
                )}
              />
            ) : null}
            {policy.subscriptionExpiryLimitEnabled ? (
              <UsageItem
                detail={`订阅到期时间最多可超过会员到期时间 ${policy.subscriptionExpiryGraceDays} 天`}
                icon={CalendarDays}
                label="订阅有效期"
              />
            ) : null}
          </div>
        ) : (
          <Alert>
            <ShieldCheck />
            <AlertTitle>当前方案没有额外额度限制</AlertTitle>
            <AlertDescription>
              订阅购买仍遵循商品库存、购买数量和账户安全规则。
            </AlertDescription>
          </Alert>
        )}
      </section>

      <MembershipCheckoutDialog
        card={membership}
        onOpenChange={setCheckoutOpen}
        open={checkoutOpen}
        returnPath={returnPath}
        symbol={symbol}
      />
    </div>
  );
}
