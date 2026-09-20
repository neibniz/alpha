/** @vitest-environment jsdom */
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { quoteUserSubscribeOrder } from "@workspace/ui/services/admin/user";
import { useOrderQuote } from "./use-order-quote";

vi.mock("@workspace/ui/services/admin/user", () => ({
  quoteUserSubscribeOrder: vi.fn(),
}));
const request = vi.mocked(quoteUserSubscribeOrder);
const defaults = {
  open: true,
  userId: 1,
  subscribeId: 2,
  quantity: 1,
  minimum: 1,
  maximum: 12,
  amountMode: "current_price" as const,
};
const response = (amount: number) =>
  ({
    data: { data: { recorded_amount: amount, user_enabled: true } },
  }) as Awaited<ReturnType<typeof quoteUserSubscribeOrder>>;
beforeEach(() => {
  vi.useFakeTimers();
  request.mockReset();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

test("invalidates the old quote immediately during a new debounce", async () => {
  request.mockResolvedValue(response(10));
  const { result, rerender } = renderHook(useOrderQuote, {
    initialProps: defaults,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  expect(result.current.quote?.recorded_amount).toBe(10);
  rerender({ ...defaults, quantity: 2 });
  expect(result.current.quote).toBeUndefined();
  expect(result.current.loading).toBe(true);
});

test("ignores older responses after changing the product", async () => {
  let resolveOld!: (value: ReturnType<typeof response>) => void;
  request.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve;
      })
  );
  request.mockResolvedValueOnce(response(20));
  const { result, rerender } = renderHook(useOrderQuote, {
    initialProps: defaults,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  rerender({ ...defaults, subscribeId: 3 });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  await act(async () => {
    resolveOld(response(10));
  });
  expect(result.current.quote?.recorded_amount).toBe(20);
});

test.each([
  0, 1.5, 13,
])("does not request a quote for invalid quantity %s", async (quantity) => {
  const { result } = renderHook(() => useOrderQuote({ ...defaults, quantity }));
  await act(async () => {
    await vi.advanceTimersByTimeAsync(300);
  });
  expect(request).not.toHaveBeenCalled();
  expect(result.current.quote).toBeUndefined();
});

test("does not reuse a quote after closing and reopening", async () => {
  request.mockResolvedValue(response(10));
  const { result, rerender } = renderHook(useOrderQuote, {
    initialProps: defaults,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  rerender({ ...defaults, open: false });
  rerender(defaults);
  expect(result.current.quote).toBeUndefined();
  expect(result.current.loading).toBe(true);
});

test("reports a failure and can retry without changing terms", async () => {
  request.mockRejectedValueOnce(new Error("private upstream details"));
  request.mockResolvedValueOnce(response(10));
  const { result } = renderHook(() => useOrderQuote(defaults));
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  expect(result.current.failed).toBe(true);
  act(() => result.current.retry());
  expect(result.current.failed).toBe(false);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  expect(result.current.quote?.recorded_amount).toBe(10);
});
