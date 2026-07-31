import { Button } from "@workspace/ui/components/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import {
  Card,
  CardContent,
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
  DialogTrigger,
} from "@workspace/ui/components/dialog";
import { Input } from "@workspace/ui/components/input";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from "@workspace/ui/components/field";
import {
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  issueUserSubscribeOrder,
  quoteUserSubscribeOrder,
} from "@workspace/ui/services/admin/user";
import {
  CalendarDays,
  CircleAlert,
  Gauge,
  Gift,
  LoaderCircle,
  Plus,
  ReceiptText,
  Smartphone,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Display } from "@/components/display";
import { useSubscribe } from "@/stores/subscribe";
import { formatDate } from "@/utils/common";

type AmountMode = "complimentary" | "current_price";

interface IssueSubscriptionDialogProps {
  userId: number;
  onIssued: () => void;
}

export function IssueSubscriptionDialog({
  userId,
  onIssued,
}: IssueSubscriptionDialogProps) {
  const { t } = useTranslation("user");
  const { subscribes } = useSubscribe();
  const [open, setOpen] = useState(false);
  const [subscribeId, setSubscribeId] = useState(0);
  const [quantity, setQuantity] = useState(1);
  const [amountMode, setAmountMode] = useState<AmountMode>("current_price");
  const [tradeNo, setTradeNo] = useState("");
  const [quote, setQuote] = useState<API.IssueUserSubscribeOrderQuote>();
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const quoteSequence = useRef(0);

  const selectedProduct = useMemo(
    () => subscribes?.find((item) => item.id === subscribeId),
    [subscribeId, subscribes]
  );
  const minimum = Math.max(1, selectedProduct?.min_quantity || 1);
  const maximum = Math.max(minimum, selectedProduct?.max_quantity || 1000);

  useEffect(() => {
    if (!open || subscribeId > 0 || !subscribes?.length) return;
    const first = subscribes[0];
    if (!first) return;
    setSubscribeId(first.id || 0);
    setQuantity(Math.max(1, first.min_quantity || 1));
  }, [open, subscribeId, subscribes]);

  useEffect(() => {
    if (!(open && subscribeId > 0 && quantity >= minimum)) {
      setQuote(undefined);
      return;
    }
    const sequence = ++quoteSequence.current;
    const timer = window.setTimeout(async () => {
      setQuoteLoading(true);
      try {
        const { data } = await quoteUserSubscribeOrder({
          user_id: userId,
          subscribe_id: subscribeId,
          quantity,
          amount_mode: amountMode,
        });
        if (sequence === quoteSequence.current) setQuote(data.data);
      } catch {
        if (sequence === quoteSequence.current) setQuote(undefined);
      } finally {
        if (sequence === quoteSequence.current) setQuoteLoading(false);
      }
    }, 250);
    return () => window.clearTimeout(timer);
  }, [amountMode, minimum, open, quantity, subscribeId, userId]);

  const issue = async () => {
    if (!quote) return;
    setSubmitting(true);
    try {
      const { data } = await issueUserSubscribeOrder({
        user_id: userId,
        subscribe_id: subscribeId,
        quantity,
        amount_mode: amountMode,
        trade_no: tradeNo.trim() || undefined,
      });
      toast.success(
        t("subscriptionIssued", "Order {{order}} queued for activation", {
          order: data.data?.order_no || "",
        })
      );
      setOpen(false);
      setTradeNo("");
      onIssued();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog onOpenChange={setOpen} open={open}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="size-4" />
          {t("issueSubscription", "Issue subscription")}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[min(760px,calc(100dvh-2rem))] overflow-y-auto p-0 sm:max-w-2xl">
        <DialogHeader className="border-b px-6 py-5">
          <DialogTitle className="flex items-center gap-2">
            <ReceiptText className="size-5" />
            {t("issueSubscription", "Issue subscription")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "issueSubscriptionDescription",
              "Create a traceable paid order from the current product terms and activate it automatically."
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup className="gap-5 px-6 py-5">
          <div className="grid gap-4 sm:grid-cols-[1fr_140px]">
            <Field>
              <FieldLabel htmlFor="issue-product">
                {t("subscriptionProduct", "Subscription product")}
              </FieldLabel>
              <Select
                onValueChange={(value) => {
                  const id = Number(value);
                  const product = subscribes?.find((item) => item.id === id);
                  setSubscribeId(id);
                  setQuantity(Math.max(1, product?.min_quantity || 1));
                }}
                value={subscribeId ? String(subscribeId) : undefined}
              >
                <SelectTrigger id="issue-product">
                  <SelectValue
                    placeholder={t("selectSubscription", "Select subscription")}
                  />
                </SelectTrigger>
                <SelectContent>
                  {subscribes?.map((item) => (
                    <SelectItem key={item.id} value={String(item.id)}>
                      {item.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel htmlFor="issue-quantity">
                {t("billingPeriods", "Billing periods")}
              </FieldLabel>
              <Input
                id="issue-quantity"
                max={maximum}
                min={minimum}
                onChange={(event) => {
                  const next = Number(event.target.value);
                  setQuantity(Number.isFinite(next) ? next : minimum);
                }}
                type="number"
                value={quantity}
              />
            </Field>
          </div>

          <Field>
            <FieldLabel>{t("orderAmount", "Order amount")}</FieldLabel>
            <RadioGroup
              className="grid gap-2 sm:grid-cols-2"
              onValueChange={(value) => setAmountMode(value as AmountMode)}
              value={amountMode}
            >
              <FieldLabel htmlFor="amount-current">
                <Field orientation="horizontal">
                  <RadioGroupItem id="amount-current" value="current_price" />
                  <FieldContent>
                    <FieldTitle>
                      {t("recordCurrentPrice", "Record current price")}
                    </FieldTitle>
                    <FieldDescription>
                      {t(
                        "recordCurrentPriceDescription",
                        "Use the product price and active volume discount."
                      )}
                    </FieldDescription>
                  </FieldContent>
                </Field>
              </FieldLabel>
              <FieldLabel htmlFor="amount-free">
                <Field orientation="horizontal">
                  <RadioGroupItem id="amount-free" value="complimentary" />
                  <FieldContent>
                    <FieldTitle>
                      <Gift />
                      {t("complimentaryOrder", "Complimentary")}
                    </FieldTitle>
                    <FieldDescription>
                      {t(
                        "complimentaryOrderDescription",
                        "Record a zero-value administrative grant."
                      )}
                    </FieldDescription>
                  </FieldContent>
                </Field>
              </FieldLabel>
            </RadioGroup>
          </Field>

          {amountMode === "current_price" ? (
            <Field>
              <FieldLabel htmlFor="issue-trade-no">
                {t("externalReference", "External reference")}
              </FieldLabel>
              <Input
                id="issue-trade-no"
                maxLength={255}
                onChange={(event) => setTradeNo(event.target.value)}
                placeholder={t(
                  "externalReferencePlaceholder",
                  "Optional offline payment or ticket reference"
                )}
                value={tradeNo}
              />
            </Field>
          ) : null}

          <Card className="gap-0 overflow-hidden py-0">
            <CardHeader className="flex-row items-center justify-between border-b bg-muted/30 px-4 py-3">
              <CardTitle className="text-sm">
                {t("orderPreview", "Order preview")}
              </CardTitle>
              {quoteLoading ? (
                <LoaderCircle className="size-4 animate-spin text-muted-foreground" />
              ) : null}
            </CardHeader>
            {quote ? (
              <CardContent className="grid grid-cols-2 p-0 sm:grid-cols-4">
                <QuoteMetric
                  icon={ReceiptText}
                  label={t("recordedAmount", "Recorded amount")}
                  value={
                    <Display type="currency" value={quote.recorded_amount} />
                  }
                />
                <QuoteMetric
                  icon={Gauge}
                  label={t("trafficLimit", "Traffic limit")}
                  value={
                    <Display type="traffic" unlimited value={quote.traffic} />
                  }
                />
                <QuoteMetric
                  icon={Smartphone}
                  label={t("deviceLimit", "Device limit")}
                  value={
                    <Display
                      type="number"
                      unlimited
                      value={quote.device_limit}
                    />
                  }
                />
                <QuoteMetric
                  icon={CalendarDays}
                  label={t("expireTime", "Expires")}
                  value={
                    quote.expire_time
                      ? formatDate(quote.expire_time, false)
                      : t("permanent", "Permanent")
                  }
                />
              </CardContent>
            ) : (
              <CardContent className="px-4 py-8 text-center text-muted-foreground text-sm">
                {t(
                  "orderPreviewUnavailable",
                  "Choose valid terms to preview the order."
                )}
              </CardContent>
            )}
          </Card>

          {quote && !quote.user_enabled ? (
            <Alert variant="destructive">
              <CircleAlert />
              <AlertTitle>{t("issueBlocked", "Issuing is blocked")}</AlertTitle>
              <AlertDescription>
                {t(
                  "disabledUserIssueBlocked",
                  "This account is disabled or deleted. Enable it before issuing a subscription."
                )}
              </AlertDescription>
            </Alert>
          ) : null}
        </FieldGroup>

        <DialogFooter className="border-t px-6 py-4">
          <Button onClick={() => setOpen(false)} variant="outline">
            {t("cancel", "Cancel")}
          </Button>
          <Button
            disabled={submitting || quoteLoading || !quote?.user_enabled}
            onClick={issue}
          >
            {submitting ? (
              <LoaderCircle className="size-4 animate-spin" />
            ) : null}
            {t("createAndActivate", "Create and activate")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function QuoteMetric({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof ReceiptText;
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="border-border/70 border-b p-3 odd:border-r sm:border-b-0 sm:border-r sm:last:border-r-0">
      <div className="flex items-center gap-1.5 text-muted-foreground text-[11px]">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className="mt-1 truncate font-medium text-sm tabular-nums">
        {value}
      </div>
    </div>
  );
}
