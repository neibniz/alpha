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
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
import {
  Field,
  FieldDescription,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Input } from "@workspace/ui/components/input";
import {
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import { Separator } from "@workspace/ui/components/separator";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import {
  Check,
  CircleAlert,
  CircleGauge,
  CreditCard,
  Gauge,
  Gift,
  HardDrive,
  Minus,
  Plus,
  ShieldCheck,
  ShoppingBag,
  Smartphone,
  Sparkles,
  WalletCards,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { OrderCheckoutStatus } from "@/components/orders/order-checkout-status";
import {
  checkoutOrder,
  createOrderRequestId,
  createPurchaseOrder,
  getPaymentMethods,
  getProducts,
  getPurchaseQuote,
  getSiteConfig,
} from "@/lib/client-api";
import {
  formatBytes,
  formatCurrency,
  formatSpeedLimitMbps,
} from "@/lib/format";
import type {
  CheckoutResult,
  PaymentMethod,
  Product,
  PurchaseQuote,
} from "@/lib/models";

const durationOptions = [
  { label: "1 个月", value: 1 },
  { label: "3 个月", value: 3 },
  { label: "12 个月", value: 12 },
];

function normalizedQuantity(product: Product, quantity: number) {
  const minimum = Math.max(1, product.minQuantity || 1);
  const maximum = product.maxQuantity || Number.POSITIVE_INFINITY;
  return Math.min(maximum, Math.max(minimum, quantity));
}

function pricing(product: Product, requestedQuantity: number) {
  const quantity = normalizedQuantity(product, requestedQuantity);
  const percentage =
    product.discounts.find((item) => item.quantity === quantity)?.discount ||
    100;
  const original = (product.unitPrice / 100) * quantity;
  return {
    original,
    price: original * (percentage / 100),
    quantity,
    saved: 100 - percentage,
  };
}

function money(cents: number, symbol: string) {
  return formatCurrency(cents / 100, symbol);
}

function ProductCard({
  duration,
  onPurchase,
  product,
  symbol,
}: Readonly<{
  duration: number;
  onPurchase: (product: Product) => void;
  product: Product;
  symbol: string;
}>) {
  const price = pricing(product, duration);
  const appearance =
    product.template === "popular"
      ? "border-foreground ring-1 ring-foreground/10"
      : "border-border";
  return (
    <Card
      className={`h-full gap-0 rounded-lg py-0 shadow-none ${appearance}`}
      data-reveal
    >
      <CardHeader className="gap-4 px-5 pt-5 pb-4 sm:px-6 sm:pt-6">
        <div className="flex items-center justify-between gap-3">
          <Badge
            variant={product.template === "popular" ? "default" : "outline"}
          >
            {product.badge || "订阅套餐"}
          </Badge>
        </div>
        <div>
          <CardTitle className="text-xl">{product.name}</CardTitle>
          <CardDescription className="mt-2 min-h-10 leading-relaxed">
            {product.description}
          </CardDescription>
        </div>
        <div>
          <strong className="inline-block font-semibold text-3xl tabular-nums">
            {formatCurrency(price.price, symbol)}
          </strong>
          <span className="ml-1 text-muted-foreground text-sm">
            / {price.quantity} 个月
          </span>
          {price.saved > 0 ? (
            <del className="ml-2 text-muted-foreground text-xs">
              {formatCurrency(price.original, symbol)}
            </del>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-5 px-5 py-4 sm:px-6">
        <dl className="grid divide-y border-y text-sm">
          <div className="flex items-center gap-3 py-3">
            <HardDrive className="text-muted-foreground" />
            <dt className="text-muted-foreground">流量</dt>
            <dd className="ml-auto font-medium">
              {formatBytes(product.trafficBytes)}
            </dd>
          </div>
          <div className="flex items-center gap-3 py-3">
            <Gauge className="text-muted-foreground" />
            <dt className="text-muted-foreground">速度</dt>
            <dd className="ml-auto font-medium">
              {formatSpeedLimitMbps(product.speedLimitMbps)}
            </dd>
          </div>
          <div className="flex items-center gap-3 py-3">
            <Smartphone className="text-muted-foreground" />
            <dt className="text-muted-foreground">设备数</dt>
            <dd className="ml-auto font-medium">
              {product.deviceLimit || "不限"} 台
            </dd>
          </div>
        </dl>
        <ul className="grid gap-3 text-sm">
          <li className="flex items-center gap-2">
            <Check className="text-foreground" />
            全部可用协议与节点
          </li>
          <li className="flex items-center gap-2">
            <Check className="text-foreground" />
            实时流量与安全提醒
          </li>
          <li className="flex items-center gap-2">
            <Check className="text-foreground" />
            可自定义节点名称
          </li>
        </ul>
      </CardContent>
      <CardFooter className="px-5 pt-0 pb-5 sm:px-6 sm:pb-6">
        <Button
          className="w-full"
          disabled={!product.available}
          onClick={() => onPurchase(product)}
          type="button"
          variant={product.template === "popular" ? "default" : "outline"}
        >
          {product.available ? "选择此套餐" : "暂不可购买"}
        </Button>
      </CardFooter>
    </Card>
  );
}

function PaymentSelector({
  methods,
  onChange,
  value,
}: Readonly<{
  methods: PaymentMethod[];
  onChange: (value: number) => void;
  value: number;
}>) {
  return (
    <FieldSet>
      <FieldLegend>支付方式</FieldLegend>
      <RadioGroup
        className="grid gap-2 sm:grid-cols-2"
        onValueChange={(next) => onChange(Number(next))}
        value={String(value)}
      >
        {methods.map((method) => {
          const Icon =
            method.platform.toLowerCase() === "balance"
              ? WalletCards
              : CreditCard;
          const id = `purchase-payment-${method.id}`;
          return (
            <FieldLabel
              className="flex cursor-pointer items-center gap-3 rounded-lg border p-3"
              htmlFor={id}
              key={method.id}
            >
              <RadioGroupItem id={id} value={String(method.id)} />
              <Icon
                aria-hidden="true"
                className="size-4 text-muted-foreground"
              />
              <span className="min-w-0">
                <strong className="block font-medium text-sm">
                  {method.name}
                </strong>
                <small className="block truncate font-normal text-muted-foreground text-xs">
                  {method.description || method.platform}
                </small>
              </span>
            </FieldLabel>
          );
        })}
      </RadioGroup>
    </FieldSet>
  );
}

function PurchaseSummary({
  product,
  quantity,
  quote,
  symbol,
}: Readonly<{
  product: Product;
  quantity: number;
  quote?: PurchaseQuote;
  symbol: string;
}>) {
  const rows = [
    ["套餐", product.name],
    ["周期", `${quantity} 个月`],
    ["流量", formatBytes(product.trafficBytes)],
  ];
  return (
    <dl className="space-y-3 rounded-lg border bg-muted/25 p-4 text-sm">
      {rows.map(([label, value]) => (
        <div className="flex justify-between gap-4" key={label}>
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="text-right font-medium">{value}</dd>
        </div>
      ))}
      {quote?.discount ? (
        <div className="flex justify-between gap-4 text-teal-700">
          <dt>套餐优惠</dt>
          <dd className="font-medium">-{money(quote.discount, symbol)}</dd>
        </div>
      ) : null}
      {quote?.couponDiscount ? (
        <div className="flex justify-between gap-4 text-teal-700">
          <dt>优惠券</dt>
          <dd className="font-medium">
            -{money(quote.couponDiscount, symbol)}
          </dd>
        </div>
      ) : null}
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
    </dl>
  );
}

function CheckoutDialog({
  duration,
  onOpenChange,
  open,
  product,
  symbol,
}: Readonly<{
  duration: number;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  product?: Product;
  symbol: string;
}>) {
  const queryClient = useQueryClient();
  const [quantity, setQuantity] = useState(1);
  const [payment, setPayment] = useState(0);
  const [coupon, setCoupon] = useState("");
  const [appliedCoupon, setAppliedCoupon] = useState("");
  const [requestId, setRequestId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [checkout, setCheckout] = useState<CheckoutResult>();
  const [orderNo, setOrderNo] = useState("");

  const payments = useQuery({
    enabled: open,
    queryFn: getPaymentMethods,
    queryKey: ["alpha-payment-methods"],
    staleTime: 60_000,
  });

  useEffect(() => {
    if (!(open && product)) return;
    setQuantity(normalizedQuantity(product, duration));
    setPayment(0);
    setCoupon("");
    setAppliedCoupon("");
    setCheckout(undefined);
    setOrderNo("");
  }, [duration, open, product]);

  useEffect(() => {
    if (!open || payment || !payments.data?.length) return;
    setPayment(payments.data[0]?.id || 0);
  }, [open, payment, payments.data]);

  useEffect(() => {
    if (!open || orderNo) return;
    setRequestId(createOrderRequestId());
  }, [appliedCoupon, open, orderNo, payment, product?.id, quantity]);

  const quote = useQuery({
    enabled: Boolean(open && product && quantity),
    queryFn: () =>
      getPurchaseQuote({
        coupon: appliedCoupon,
        payment,
        quantity,
        subscribe_id: product?.id || 0,
      }),
    queryKey: [
      "alpha-purchase-quote",
      product?.id,
      quantity,
      payment,
      appliedCoupon,
    ],
    retry: false,
  });

  if (!product) return null;
  const minimum = Math.max(1, product.minQuantity || 1);
  const maximum = product.maxQuantity || 0;
  const paymentRequired = Boolean(quote.data && quote.data.amount > 0);
  const canSubmit =
    Boolean(quote.data && requestId) &&
    !quote.isFetching &&
    !submitting &&
    (!paymentRequired || payment !== 0);

  const close = () => onOpenChange(false);
  const submit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const createdOrderNo = await createPurchaseOrder({
        coupon: appliedCoupon,
        payment,
        quantity,
        request_id: requestId,
        subscribe_id: product.id,
      });
      const checkoutResult = await checkoutOrder(createdOrderNo);
      setOrderNo(createdOrderNo);
      setCheckout(checkoutResult);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "订单创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        {checkout && orderNo ? (
          <OrderCheckoutStatus
            checkout={checkout}
            kind="purchase"
            onClose={close}
            onComplete={async () => {
              await Promise.all([
                queryClient.invalidateQueries({
                  queryKey: ["alpha-subscriptions"],
                }),
                queryClient.invalidateQueries({ queryKey: ["alpha-orders"] }),
                queryClient.invalidateQueries({
                  queryKey: ["alpha-current-user"],
                }),
              ]);
            }}
            orderNo={orderNo}
          />
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>确认订阅</DialogTitle>
              <DialogDescription>
                当前商品价格与优惠会写入订单；后续续费按届时商品价格计算。
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-6 md:grid-cols-[0.9fr_1.1fr]">
              <PurchaseSummary
                product={product}
                quantity={quantity}
                quote={quote.data}
                symbol={symbol}
              />
              <div className="space-y-5">
                <Field>
                  <div className="flex items-center justify-between gap-3">
                    <FieldLabel htmlFor="purchase-quantity">
                      购买周期
                    </FieldLabel>
                    <FieldDescription>
                      {minimum} 至 {maximum || "不限"} 个月
                    </FieldDescription>
                  </div>
                  <ButtonGroup>
                    <Button
                      aria-label="减少一个周期"
                      disabled={quantity <= minimum}
                      onClick={() =>
                        setQuantity((value) => Math.max(minimum, value - 1))
                      }
                      size="icon"
                      type="button"
                      variant="outline"
                    >
                      <Minus aria-hidden="true" />
                    </Button>
                    <Input
                      aria-label="购买周期数量"
                      className="w-20 text-center tabular-nums"
                      id="purchase-quantity"
                      max={maximum || 1000}
                      min={minimum}
                      onChange={(event) =>
                        setQuantity(
                          normalizedQuantity(
                            product,
                            Number(event.target.value) || minimum
                          )
                        )
                      }
                      type="number"
                      value={quantity}
                    />
                    <Button
                      aria-label="增加一个周期"
                      disabled={Boolean(maximum) && quantity >= maximum}
                      onClick={() =>
                        setQuantity((value) =>
                          normalizedQuantity(product, value + 1)
                        )
                      }
                      size="icon"
                      type="button"
                      variant="outline"
                    >
                      <Plus aria-hidden="true" />
                    </Button>
                  </ButtonGroup>
                </Field>
                <Field>
                  <FieldLabel htmlFor="purchase-coupon">优惠券</FieldLabel>
                  <InputGroup>
                    <InputGroupInput
                      id="purchase-coupon"
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
                    <PaymentSelector
                      methods={payments.data}
                      onChange={setPayment}
                      value={payment}
                    />
                  )
                ) : quote.data ? (
                  <Alert>
                    <Gift />
                    <AlertTitle>本次无需支付</AlertTitle>
                    <AlertDescription>
                      提交后由服务端直接完成开通。
                    </AlertDescription>
                  </Alert>
                ) : (
                  <div className="flex items-center gap-2 rounded-lg border p-4 text-muted-foreground text-sm">
                    <Spinner />
                    正在计算当前价格
                  </div>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button onClick={close} variant="outline">
                取消
              </Button>
              <Button disabled={!canSubmit} onClick={submit}>
                {submitting ? <Spinner /> : null}
                创建订单
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

export function ProductStorefront({
  authenticated,
}: Readonly<{ authenticated: boolean }>) {
  const router = useRouter();
  const [duration, setDuration] = useState(12);
  const [selected, setSelected] = useState<Product>();
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const products = useQuery({
    queryFn: () => getProducts({ authenticated }),
    queryKey: ["alpha-products", authenticated ? "user" : "guest"],
    staleTime: authenticated ? 15_000 : 60_000,
  });
  const config = useQuery({
    queryFn: getSiteConfig,
    queryKey: ["alpha-site-config"],
    staleTime: 300_000,
  });

  const displayedProducts = useMemo(() => products.data || [], [products.data]);

  if (products.isLoading) {
    return (
      <div className="mx-auto w-full max-w-[1500px] px-4 py-8 sm:px-6 lg:px-7">
        <div className="mb-8 flex items-end justify-between gap-4">
          <div className="space-y-3">
            <Skeleton className="h-8 w-44" />
            <Skeleton className="h-4 w-80 max-w-full" />
          </div>
          <Skeleton className="hidden h-9 w-60 sm:block" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2].map((item) => (
            <Skeleton className="h-[460px] rounded-lg" key={item} />
          ))}
        </div>
      </div>
    );
  }

  if (products.isError) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6">
        <Alert variant="destructive">
          <CircleAlert />
          <AlertTitle>套餐暂时无法加载</AlertTitle>
          <AlertDescription>
            <Button
              className="h-auto px-0 py-0"
              onClick={() => products.refetch()}
              variant="link"
            >
              重新加载
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-[1500px] px-4 py-8 sm:px-6 lg:px-7 lg:py-9">
      <section
        className="mb-8 flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between"
        data-reveal
      >
        <div>
          <h1 className="font-semibold text-3xl tracking-tight">
            选择适合你的套餐
          </h1>
          <p className="mt-2 max-w-2xl text-muted-foreground text-sm leading-relaxed">
            每次购买都会创建一个独立订阅，可分别管理流量、接入方式与续费。
          </p>
        </div>
        <ToggleGroup
          aria-label="购买周期"
          className="w-full sm:w-auto"
          onValueChange={(value) => {
            if (value) setDuration(Number(value));
          }}
          type="single"
          value={String(duration)}
          variant="outline"
        >
          {durationOptions.map((option) => (
            <ToggleGroupItem
              className="flex-1 sm:flex-none"
              key={option.value}
              value={String(option.value)}
            >
              {option.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </section>

      {displayedProducts.length ? (
        <section className="grid items-stretch gap-4 md:grid-cols-2 xl:grid-cols-3">
          {displayedProducts.map((product) => (
            <ProductCard
              duration={duration}
              key={product.id}
              onPurchase={(item) => {
                if (!authenticated) {
                  router.push("/login?next=%2Fproducts");
                  return;
                }
                setSelected(item);
                setCheckoutOpen(true);
              }}
              product={product}
              symbol={config.data?.currencySymbol || "¥"}
            />
          ))}
        </section>
      ) : (
        <Empty className="min-h-80 border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ShoppingBag aria-hidden="true" />
            </EmptyMedia>
            <EmptyTitle>当前没有可购买的套餐</EmptyTitle>
            <EmptyDescription>套餐上架后会显示在这里。</EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}

      <section
        className="mt-7 grid divide-y rounded-lg border text-sm sm:grid-cols-3 sm:divide-x sm:divide-y-0"
        data-reveal
      >
        <div className="flex gap-3 p-5">
          <ShieldCheck className="shrink-0" />
          <span>
            <strong className="block font-medium">订阅链接保持私密</strong>
            <small className="mt-1 block text-muted-foreground">
              仅在创建和主动复制时展示。
            </small>
          </span>
        </div>
        <div className="flex gap-3 p-5">
          <CircleGauge className="shrink-0" />
          <span>
            <strong className="block font-medium">订阅独立管理</strong>
            <small className="mt-1 block text-muted-foreground">
              流量、到期时间与预设互不影响。
            </small>
          </span>
        </div>
        <div className="flex gap-3 p-5">
          <Sparkles className="shrink-0" />
          <span>
            <strong className="block font-medium">按当前价格续费</strong>
            <small className="mt-1 block text-muted-foreground">
              到期前可随时在订阅中处理。
            </small>
          </span>
        </div>
      </section>

      {authenticated ? (
        <CheckoutDialog
          duration={duration}
          onOpenChange={setCheckoutOpen}
          open={checkoutOpen}
          product={selected}
          symbol={config.data?.currencySymbol || "¥"}
        />
      ) : null}
    </div>
  );
}
