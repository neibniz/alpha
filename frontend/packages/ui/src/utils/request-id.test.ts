import { describe, expect, test } from "bun:test";
import { createRequestId, withRequestId } from "./request-id";

describe("request IDs", () => {
  test("creates RFC 4122 version 4 IDs", () => {
    expect(createRequestId()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
    );
  });

  test("preserves an existing request ID", () => {
    const body = withRequestId({
      amount: 100,
      request_id: "checkout-attempt-00000001",
    });

    expect(body).toEqual({
      amount: 100,
      request_id: "checkout-attempt-00000001",
    });
  });

  test("fills an absent or blank request ID", () => {
    expect(withRequestId({ amount: 100 }).request_id).toHaveLength(36);
    expect(
      withRequestId({ amount: 100, request_id: " " }).request_id
    ).toHaveLength(36);
  });
});
