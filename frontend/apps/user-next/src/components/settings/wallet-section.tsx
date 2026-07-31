"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertTitle } from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Dialog, DialogContent } from "@workspace/ui/components/dialog";
import {
  Field,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import {
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import { CircleAlert, CreditCard } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { OrderCheckoutStatus } from "@/components/orders/order-checkout-status";
import {
  checkoutOrder,
  createOrderRequestId,
  createRechargeOrder,
  getCurrentUser,
  getPaymentMethods,
  getWalletLogs,
} from "@/lib/client-api";
import { formatCurrency, formatDate } from "@/lib/format";
import type { CheckoutResult, PaymentMethod } from "@/lib/models";
import { errorMessage } from "./settings-common";

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
          const id = `recharge-payment-${method.id}`;
          return (
            <FieldLabel
              className="flex cursor-pointer items-center gap-3 rounded-lg border p-3"
              htmlFor={id}
              key={method.id}
            >
              <RadioGroupItem id={id} value={String(method.id)} />
              <CreditCard className="size-4 text-muted-foreground" />
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

export function WalletSection() {
  const queryClient = useQueryClient();
  const user = useQuery({
    queryFn: getCurrentUser,
    queryKey: ["alpha-current-user"],
  });
  const logs = useQuery({
    queryFn: getWalletLogs,
    queryKey: ["alpha-wallet-logs"],
  });
  const methods = useQuery({
    queryFn: getPaymentMethods,
    queryKey: ["alpha-payment-methods"],
  });
  const externalMethods = (methods.data || []).filter(
    (method) => method.platform.toLowerCase() !== "balance"
  );
  const [amount, setAmount] = useState("100");
  const [payment, setPayment] = useState(0);
  const [requestId, setRequestId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [checkout, setCheckout] = useState<CheckoutResult>();
  const [orderNo, setOrderNo] = useState("");

  useEffect(() => {
    if (!payment && externalMethods[0]) setPayment(externalMethods[0].id);
  }, [externalMethods, payment]);
  useEffect(() => {
    if (!orderNo) setRequestId(createOrderRequestId());
  }, [amount, orderNo, payment]);

  const amountCents = Math.round(Number(amount) * 100);
  const submit = async () => {
    if (!(amountCents > 0 && payment && requestId)) return;
    setSubmitting(true);
    try {
      const createdOrderNo = await createRechargeOrder({
        amount: amountCents,
        payment,
        request_id: requestId,
      });
      const checkoutResult = await checkoutOrder(createdOrderNo);
      setOrderNo(createdOrderNo);
      setCheckout(checkoutResult);
    } catch (error) {
      toast.error(errorMessage(error, "充值订单创建失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="grid gap-4">
      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardDescription>可用余额</CardDescription>
          <CardTitle className="font-semibold text-3xl tabular-nums">
            {formatCurrency((user.data?.balance || 0) / 100)}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 px-5 py-5 sm:grid-cols-2">
          <div>
            <span className="text-muted-foreground text-xs">赠送余额</span>
            <strong className="mt-1 block font-medium tabular-nums">
              {formatCurrency((user.data?.giftAmount || 0) / 100)}
            </strong>
          </div>
          <div>
            <span className="text-muted-foreground text-xs">佣金</span>
            <strong className="mt-1 block font-medium tabular-nums">
              {formatCurrency((user.data?.commission || 0) / 100)}
            </strong>
          </div>
        </CardContent>
        <CardFooter className="border-t px-5 py-3 text-muted-foreground text-xs">
          结算时依次抵扣佣金、赠送余额，再调用支付方式。
        </CardFooter>
      </Card>

      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">账户充值</CardTitle>
          <CardDescription>
            充值只能使用外部支付方式，不能使用账户余额。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5 px-5 py-5">
          <Field>
            <FieldLabel>充值金额</FieldLabel>
            <ToggleGroup
              className="grid w-full grid-cols-4"
              onValueChange={(next) => {
                if (next) setAmount(next);
              }}
              type="single"
              value={amount}
              variant="outline"
            >
              {[50, 100, 200, 500].map((value) => (
                <ToggleGroupItem key={value} value={String(value)}>
                  ¥{value}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </Field>
          <Field>
            <FieldLabel htmlFor="recharge-amount">自定义金额</FieldLabel>
            <InputGroup>
              <InputGroupInput
                id="recharge-amount"
                inputMode="decimal"
                max="20000000"
                min="1"
                onChange={(event) => setAmount(event.target.value)}
                step="0.01"
                type="number"
                value={amount}
              />
              <InputGroupAddon align="inline-end">元</InputGroupAddon>
            </InputGroup>
          </Field>
          {externalMethods.length ? (
            <PaymentSelector
              methods={externalMethods}
              onChange={setPayment}
              value={payment}
            />
          ) : methods.isLoading ? (
            <Skeleton className="h-20 rounded-lg" />
          ) : (
            <Alert variant="destructive">
              <CircleAlert />
              <AlertTitle>暂无可用的外部支付方式</AlertTitle>
            </Alert>
          )}
        </CardContent>
        <CardFooter className="justify-end border-t px-5 py-4">
          <Button
            disabled={submitting || !(amountCents > 0 && payment)}
            onClick={submit}
          >
            {submitting ? <Spinner /> : <CreditCard />}
            继续支付
          </Button>
        </CardFooter>
      </Card>

      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">资产明细</CardTitle>
          <CardDescription>
            最近 30 条余额、赠送余额和佣金变动。
          </CardDescription>
        </CardHeader>
        <CardContent className="divide-y px-5 py-0">
          {(logs.data || []).map((log) => (
            <div
              className="flex items-center justify-between gap-4 py-4"
              key={`${log.asset}-${log.id}`}
            >
              <span className="min-w-0">
                <strong className="block font-medium text-sm">
                  {log.asset === "balance"
                    ? "余额"
                    : log.asset === "gift"
                      ? "赠送余额"
                      : "佣金"}
                </strong>
                <small className="block truncate text-muted-foreground">
                  {formatDate(log.timestamp)}
                  {log.orderNo ? ` · ${log.orderNo}` : ""}
                </small>
              </span>
              <b
                className={`font-medium tabular-nums ${log.amount >= 0 ? "text-teal-700" : "text-foreground"}`}
              >
                {log.amount >= 0 ? "+" : ""}
                {formatCurrency(log.amount / 100)}
              </b>
            </div>
          ))}
          {!(logs.isLoading || logs.data?.length) ? (
            <p className="py-8 text-center text-muted-foreground text-sm">
              暂无资产变动。
            </p>
          ) : null}
        </CardContent>
      </Card>
      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            setCheckout(undefined);
            setOrderNo("");
          }
        }}
        open={Boolean(checkout && orderNo)}
      >
        <DialogContent className="sm:max-w-lg">
          {checkout && orderNo ? (
            <OrderCheckoutStatus
              checkout={checkout}
              kind="recharge"
              onClose={() => {
                setCheckout(undefined);
                setOrderNo("");
              }}
              onComplete={async () => {
                await Promise.all([
                  queryClient.invalidateQueries({
                    queryKey: ["alpha-current-user"],
                  }),
                  queryClient.invalidateQueries({
                    queryKey: ["alpha-wallet-logs"],
                  }),
                  queryClient.invalidateQueries({ queryKey: ["alpha-orders"] }),
                ]);
              }}
              orderNo={orderNo}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
