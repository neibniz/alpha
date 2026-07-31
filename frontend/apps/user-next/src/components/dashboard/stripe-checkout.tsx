"use client";

import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from "@stripe/react-stripe-js";
import { loadStripe } from "@stripe/stripe-js";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { Spinner } from "@workspace/ui/components/spinner";
import { CircleAlert, LockKeyhole } from "lucide-react";
import { useMemo, useState } from "react";
import type { StripeCheckout } from "@/lib/models";

function StripePaymentForm({ onSuccess }: Readonly<{ onSuccess: () => void }>) {
  const stripe = useStripe();
  const elements = useElements();
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState("");

  return (
    <form
      className="space-y-4"
      onSubmit={async (event) => {
        event.preventDefault();
        if (!(stripe && elements)) return;
        setProcessing(true);
        setError("");
        const result = await stripe.confirmPayment({
          confirmParams: { return_url: window.location.href },
          elements,
          redirect: "if_required",
        });
        if (result.error) {
          setError(result.error.message || "支付未完成，请检查支付信息");
          setProcessing(false);
          return;
        }
        if (result.paymentIntent?.status === "succeeded") onSuccess();
        else setError("支付正在处理，请稍后在订单中查看状态");
        setProcessing(false);
      }}
    >
      <PaymentElement options={{ layout: "tabs" }} />
      {error ? (
        <Alert variant="destructive">
          <CircleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      <Button
        className="w-full"
        disabled={processing || !(stripe && elements)}
        type="submit"
      >
        {processing ? <Spinner /> : <LockKeyhole />}
        安全支付
      </Button>
    </form>
  );
}

export default function StripeCheckoutPanel({
  checkout,
  onSuccess,
}: Readonly<{
  checkout: StripeCheckout;
  onSuccess: () => void;
}>) {
  const stripe = useMemo(
    () => loadStripe(checkout.publishableKey),
    [checkout.publishableKey]
  );
  return (
    <Elements
      options={{
        appearance: {
          theme: "stripe",
          variables: {
            borderRadius: "4px",
            colorPrimary: "#12bfd0",
            fontFamily: "Inter, system-ui, sans-serif",
          },
        },
        clientSecret: checkout.clientSecret,
      }}
      stripe={stripe}
    >
      <StripePaymentForm onSuccess={onSuccess} />
    </Elements>
  );
}
