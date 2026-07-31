import { describe, expect, test } from "bun:test";
import {
  isDisallowedIntegerKey,
  isIntegerNumberInput,
  normalizeIntegerInputValue,
} from "./numeric-input";

describe("integer number inputs", () => {
  test("identifies numeric inputs with an integer step", () => {
    expect(isIntegerNumberInput("number", 1)).toBe(true);
    expect(isIntegerNumberInput("number", "1")).toBe(true);
    expect(isIntegerNumberInput("number", 0.01)).toBe(false);
    expect(isIntegerNumberInput("text", 1)).toBe(false);
  });

  test("normalizes pasted decimal and exponent values", () => {
    expect(normalizeIntegerInputValue("12.9", false)).toBe("12");
    expect(normalizeIntegerInputValue("1e3", false)).toBe("1000");
    expect(normalizeIntegerInputValue("invalid", false)).toBe("");
  });

  test("preserves empty and valid integer drafts", () => {
    expect(normalizeIntegerInputValue("", false)).toBe("");
    expect(normalizeIntegerInputValue("0042", false)).toBe("0042");
    expect(normalizeIntegerInputValue("-", true)).toBe("-");
    expect(normalizeIntegerInputValue("-3", true)).toBe("-3");
    expect(normalizeIntegerInputValue("-3", false)).toBe("");
  });

  test("blocks decimal and exponent keys for integer fields", () => {
    expect(isDisallowedIntegerKey(".", false)).toBe(true);
    expect(isDisallowedIntegerKey("e", false)).toBe(true);
    expect(isDisallowedIntegerKey("-", false)).toBe(true);
    expect(isDisallowedIntegerKey("-", true)).toBe(false);
    expect(isDisallowedIntegerKey("5", false)).toBe(false);
  });
});
