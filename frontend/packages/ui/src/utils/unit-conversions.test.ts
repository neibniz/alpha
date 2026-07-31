import { describe, expect, test } from "bun:test";
import { evaluateWithPrecision, unitConversion } from "./unit-conversions";

describe("unitConversion", () => {
  test("converts currency without floating-point residue", () => {
    expect(unitConversion("centsToDollars", 1234)).toBe(12.34);
    expect(unitConversion("dollarsToCents", 12.34)).toBe(1234);
  });

  test("converts traffic and speed units", () => {
    expect(unitConversion("bitsToMb", 300 * 1024 ** 2)).toBe(300);
    expect(unitConversion("mbToBits", 300)).toBe(300 * 1024 ** 2);
    expect(unitConversion("bytesToGb", 5 * 1024 ** 3)).toBe(5);
    expect(unitConversion("gbToBytes", 5)).toBe(5 * 1024 ** 3);
  });

  test("rounds calculations to two decimal places", () => {
    expect(evaluateWithPrecision(10 / 3)).toBe(3.33);
    expect(evaluateWithPrecision(Number.NaN)).toBe(0);
  });
});
