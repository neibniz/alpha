import { describe, expect, it } from "vitest";
import { isInternalRedirect } from "./common";

describe("administrator redirect validation", () => {
  it("accepts application routes", () => {
    expect(isInternalRedirect("/dashboard")).toBe(true);
    expect(isInternalRedirect("/dashboard/order?page=2")).toBe(true);
  });

  it("rejects external and malformed routes", () => {
    expect(isInternalRedirect("//example.com")).toBe(false);
    expect(isInternalRedirect("https://example.com")).toBe(false);
    expect(isInternalRedirect("dashboard")).toBe(false);
    expect(isInternalRedirect()).toBe(false);
  });
});
