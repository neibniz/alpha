import { describe, expect, test } from "bun:test";
import { formatCurrencyAmount } from "./formatting";

describe("formatCurrencyAmount", () => {
  test("places the sign before the currency symbol", () => {
    expect(formatCurrencyAmount(-2800, "$")).toBe("-$28.00");
    expect(formatCurrencyAmount(2800, "$")).toBe("$28.00");
  });

  test("formats amounts without a configured symbol", () => {
    expect(formatCurrencyAmount(-1)).toBe("-0.01");
  });
});
