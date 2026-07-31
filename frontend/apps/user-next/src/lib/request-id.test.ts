import { describe, expect, test } from "bun:test";
import { createOrderRequestId } from "@/lib/request-id";

describe("createOrderRequestId", () => {
  test("uses randomUUID when the browser exposes it", () => {
    const expected = "f4a48f87-5d57-48c2-9783-229f2aa6879a";
    expect(
      createOrderRequestId({
        getRandomValues: (array) => array,
        randomUUID: () => expected,
      })
    ).toBe(expected);
  });

  test("creates an RFC 4122 v4 id on non-secure HTTP origins", () => {
    const id = createOrderRequestId({
      getRandomValues: (array) => {
        array.fill(0xab);
        return array;
      },
    });

    expect(id).toBe("abababab-abab-4bab-abab-abababababab");
    expect(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(
        id
      )
    ).toBe(true);
  });

  test("fails closed without a cryptographic random source", () => {
    let message = "";
    try {
      createOrderRequestId({ getRandomValues: undefined as never });
    } catch (error) {
      message = error instanceof Error ? error.message : String(error);
    }
    expect(message).toBe("当前浏览器无法生成安全的订单请求标识");
  });
});
