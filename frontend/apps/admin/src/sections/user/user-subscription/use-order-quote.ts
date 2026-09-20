import { useEffect, useState } from "react";
import { quoteUserSubscribeOrder } from "@workspace/ui/services/admin/user";

export function useOrderQuote({
  open,
  userId,
  subscribeId,
  quantity,
  minimum,
  maximum,
  amountMode,
}: {
  open: boolean;
  userId: number;
  subscribeId: number;
  quantity: number;
  minimum: number;
  maximum: number;
  amountMode: "complimentary" | "current_price";
}) {
  const [attempt, setAttempt] = useState(0);
  const valid =
    open &&
    subscribeId > 0 &&
    Number.isInteger(quantity) &&
    quantity >= minimum &&
    quantity <= maximum;
  const key = JSON.stringify([
    open,
    userId,
    subscribeId,
    quantity,
    minimum,
    maximum,
    amountMode,
    attempt,
  ]);
  const [result, setResult] = useState<{
    key: string;
    quote?: API.IssueUserSubscribeOrderQuote;
    failed?: boolean;
  }>();

  useEffect(() => {
    setResult(undefined);
    if (!valid) return;
    let active = true;
    const timer = window.setTimeout(async () => {
      try {
        const { data } = await quoteUserSubscribeOrder({
          user_id: userId,
          subscribe_id: subscribeId,
          quantity,
          amount_mode: amountMode,
        });
        if (active) setResult({ key, quote: data.data, failed: !data.data });
      } catch {
        if (active) setResult({ key, failed: true });
      }
    }, 250);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [key, valid, userId, subscribeId, quantity, amountMode]);

  // Never expose a quote for the previous form values, even during the debounce.
  const current = valid && result?.key === key ? result : undefined;
  return {
    quote: current?.quote,
    loading: valid && !current,
    failed: Boolean(current?.failed),
    retry: () => setAttempt((value) => value + 1),
  };
}
