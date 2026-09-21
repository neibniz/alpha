import { describe, expect, it } from "vitest";
import { isUnauthorizedSessionError } from "./admin-bootstrap";

describe("administrator bootstrap", () => {
  it("recognizes direct and nested unauthorized responses", () => {
    expect(isUnauthorizedSessionError({ status: 401 })).toBe(true);
    expect(isUnauthorizedSessionError({ data: { code: 40_004 } })).toBe(true);
    expect(
      isUnauthorizedSessionError({
        response: { data: { code: 40_003 }, status: 200 },
      })
    ).toBe(true);
  });

  it("does not turn transient failures into logout events", () => {
    expect(
      isUnauthorizedSessionError({ status: 200, data: { code: 401 } })
    ).toBe(false);
    expect(
      isUnauthorizedSessionError({
        response: { status: 200, data: { code: 401 } },
      })
    ).toBe(false);
    expect(isUnauthorizedSessionError({ status: 500 })).toBe(false);
    expect(
      isUnauthorizedSessionError({
        response: { data: { code: 500 }, status: 503 },
      })
    ).toBe(false);
    expect(isUnauthorizedSessionError(new Error("Network unavailable"))).toBe(
      false
    );
  });
});
