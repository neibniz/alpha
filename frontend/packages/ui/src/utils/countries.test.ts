// @ts-nocheck
import { describe, expect, test } from "bun:test";
import { getCountry } from "./countries.js";

describe("country utilities", () => {
  test("looks up countries by locale with module level index", () => {
    expect(getCountry("en-US")?.alpha2).toBe("US");
    expect(getCountry("en-US")?.lang).toBe("en");
  });

  test("returns undefined for missing or malformed locales", () => {
    expect(getCountry("en")).toBeUndefined();
    expect(getCountry("zz-ZZ")).toBeUndefined();
  });
});
