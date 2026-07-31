export const PAYMENT_NAME_MAX_LENGTH = 100;
export const PAYMENT_TEXT_MAX_LENGTH = 5000;

const PLATFORM_ALIASES: Record<string, string> = {
  stripe: "Stripe",
  epay: "EPay",
  alipayf2f: "AlipayF2F",
  alipay_f2f: "AlipayF2F",
  cryptosaas: "CryptoSaaS",
  crypto_saas: "CryptoSaaS",
  balance: "balance",
};

export function canonicalPaymentPlatform(value?: string): string {
  const trimmed = value?.trim() || "";
  return PLATFORM_ALIASES[trimmed.toLowerCase()] || trimmed;
}

export function isHttpUrl(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

export function paymentConfigFieldIsRequired(field: string): boolean {
  return field !== "webhook_secret" && field !== "sandbox";
}

export function normalizePaymentFees(values: {
  fee_mode: number;
  fee_percent?: number;
  fee_amount?: number;
}) {
  if (values.fee_mode === 0) {
    return { fee_percent: undefined, fee_amount: undefined };
  }
  if (values.fee_mode === 1) {
    return { fee_percent: values.fee_percent, fee_amount: undefined };
  }
  if (values.fee_mode === 2) {
    return { fee_percent: undefined, fee_amount: values.fee_amount };
  }
  return {
    fee_percent: values.fee_percent,
    fee_amount: values.fee_amount,
  };
}
