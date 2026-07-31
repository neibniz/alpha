import { describe, expect, it } from "vitest";
import {
  canonicalPaymentPlatform,
  isHttpUrl,
  normalizePaymentFees,
  paymentConfigFieldIsRequired,
} from "./payment-form-data";

describe("payment form helpers", () => {
  it("normalizes legacy platform aliases", () => {
    expect(canonicalPaymentPlatform(" epay ")).toBe("EPay");
    expect(canonicalPaymentPlatform("alipay_f2f")).toBe("AlipayF2F");
    expect(canonicalPaymentPlatform("Custom")).toBe("Custom");
  });

  it("accepts only absolute HTTP callback URLs", () => {
    expect(isHttpUrl("https://pay.alpha.test")).toBe(true);
    expect(isHttpUrl("http://localhost:8080")).toBe(true);
    expect(isHttpUrl("javascript:alert(1)")).toBe(false);
    expect(isHttpUrl("/relative")).toBe(false);
  });

  it("keeps only fees used by the selected mode", () => {
    expect(
      normalizePaymentFees({ fee_mode: 0, fee_percent: 2, fee_amount: 3 })
    ).toEqual({
      fee_percent: undefined,
      fee_amount: undefined,
    });
    expect(
      normalizePaymentFees({ fee_mode: 3, fee_percent: 2, fee_amount: 3 })
    ).toEqual({
      fee_percent: 2,
      fee_amount: 3,
    });
  });

  it("does not require generated or boolean configuration fields", () => {
    expect(paymentConfigFieldIsRequired("secret_key")).toBe(true);
    expect(paymentConfigFieldIsRequired("webhook_secret")).toBe(false);
    expect(paymentConfigFieldIsRequired("sandbox")).toBe(false);
  });
});
