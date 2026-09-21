/** @vitest-environment jsdom */
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import CouponForm from "./coupon-form";
import { defaultCouponFormValues } from "./coupon-form-data";

vi.mock("@/stores/subscribe", () => ({
  useSubscribe: () => ({ subscribes: [] }),
}));
vi.mock("@workspace/ui/composed/date-picker", () => ({
  DatePicker: () => <button type="button">Date</button>,
}));
vi.mock("@workspace/ui/composed/combobox", () => ({
  Combobox: () => <button type="button">Products</button>,
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe = vi.fn();
      unobserve = vi.fn();
      disconnect = vi.fn();
    }
  );
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const initialValues = {
  ...defaultCouponFormValues(),
  id: 1,
  name: "Summer",
  code: "SUMMER",
};
const base = {
  title: "Edit coupon",
  trigger: <button type="button">Edit</button>,
  initialValues,
  onSubmit: vi.fn(async () => true),
};

test("keeps failed detail loading open, blocks save, and retries", async () => {
  const load = vi
    .fn()
    .mockRejectedValueOnce(new Error("offline"))
    .mockResolvedValueOnce(initialValues);
  render(<CouponForm {...base} loadInitialValues={load} />);
  fireEvent.click(screen.getByRole("button", { name: /^Edit$/ }));
  await screen.findByRole("alert");
  expect(
    (screen.getByRole("button", { name: "Save coupon" }) as HTMLButtonElement)
      .disabled
  ).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  await screen.findByPlaceholderText("Enter Coupon Name");
  expect(screen.queryByRole("alert")).toBeNull();
  expect(load).toHaveBeenCalledTimes(2);
});

test("closing a loading sheet prevents the late result from replacing a reopened form", async () => {
  let resolveOld!: (value: typeof initialValues) => void;
  const load = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveOld = resolve;
        })
    )
    .mockResolvedValueOnce({ ...initialValues, name: "New data" });
  render(<CouponForm {...base} loadInitialValues={load} />);
  fireEvent.click(screen.getByRole("button", { name: /^Edit$/ }));
  fireEvent.click(screen.getByRole("button", { name: /^Cancel$/ }));
  fireEvent.click(screen.getByRole("button", { name: /^Edit$/ }));
  await screen.findByDisplayValue("New data");
  await act(async () => resolveOld(initialValues));
  expect(screen.queryByDisplayValue("Summer")).toBeNull();
});

test("supports sub-unit fixed discounts", async () => {
  const submit = vi.fn(async () => true);
  render(
    <CouponForm
      {...base}
      initialValues={{ ...initialValues, type: 2 }}
      onSubmit={submit}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: /^Edit$/ }));
  const amount = await screen.findByRole("spinbutton");
  fireEvent.change(amount, { target: { value: "0.50" } });
  expect((amount as HTMLInputElement).value).toBe("0.5");
  await waitFor(() =>
    expect(
      (screen.getByRole("button", { name: "Save coupon" }) as HTMLButtonElement)
        .disabled
    ).toBe(false)
  );
  fireEvent.click(screen.getByRole("button", { name: "Save coupon" }));
  await waitFor(() =>
    expect(submit).toHaveBeenCalledWith(
      expect.objectContaining({ discount: 50 })
    )
  );
});

test("shows unlimited usage limits explicitly as zero", async () => {
  render(<CouponForm {...base} />);
  fireEvent.click(screen.getByRole("button", { name: /^Edit$/ }));
  const limits = await screen.findByRole("tab", { name: "Validity & limits" });
  fireEvent.mouseDown(limits, { button: 0, ctrlKey: false });
  await waitFor(() =>
    expect(screen.getAllByRole("spinbutton")).toHaveLength(2)
  );
  for (const input of screen.getAllByRole("spinbutton")) {
    expect((input as HTMLInputElement).value).toBe("0");
  }
});
