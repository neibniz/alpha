/** @vitest-environment jsdom */
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AdminRouteError } from "./admin-route-error";

const invalidate = vi.hoisted(() => vi.fn());
vi.mock("@tanstack/react-router", () => ({
  useRouter: () => ({ invalidate }),
}));
beforeEach(() => {
  invalidate.mockReset();
});
afterEach(cleanup);
function props(error: Error, reset = vi.fn()): ErrorComponentProps {
  return { error, reset, info: { componentStack: "" } };
}

test("offers a full reload for failed chunks instead of an ineffective route retry", () => {
  render(
    <AdminRouteError
      {...props(new Error("Failed to fetch dynamically imported module"))}
    />
  );
  expect(screen.getByRole("button", { name: "刷新页面" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "重试" })).toBeNull();
  expect(invalidate).not.toHaveBeenCalled();
});

test("does not mislabel a rendering exception as a network failure or expose raw details", () => {
  render(<AdminRouteError {...props(new Error("private details"))} />);
  expect(screen.getByText(/不一定是网络问题/)).toBeTruthy();
  expect(screen.queryByText("private details")).toBeNull();
});

test("handles a failed retry without an unhandled rejection", async () => {
  invalidate.mockRejectedValue(new Error("offline"));
  const reset = vi.fn();
  render(<AdminRouteError {...props(new Error("Network Error"), reset)} />);
  fireEvent.click(screen.getByRole("button", { name: "重试" }));
  await screen.findByRole("status");
  expect(reset).not.toHaveBeenCalled();
  expect(
    (screen.getByRole("button", { name: "重试" }) as HTMLButtonElement).disabled
  ).toBe(false);
});

test("invalidates failed loaders before resetting the error boundary", async () => {
  const reset = vi.fn();
  invalidate.mockImplementation(async () => {
    expect(reset).not.toHaveBeenCalled();
  });
  render(<AdminRouteError {...props(new Error("Network Error"), reset)} />);
  fireEvent.click(screen.getByRole("button", { name: "重试" }));
  await waitFor(() => expect(reset).toHaveBeenCalledTimes(1));
});
