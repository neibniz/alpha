/** @vitest-environment jsdom */
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { getOrderDetail } from "@workspace/ui/services/admin/order";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { OrderDetailSheet } from "./order-detail-sheet";

vi.mock("@workspace/ui/services/admin/order", () => ({
  getOrderDetail: vi.fn(),
}));
vi.mock("../user/user-detail", () => ({ UserDetail: () => null }));
vi.mock("@/components/display", () => ({
  Display: ({ value }: { value: number }) => <span>{value}</span>,
}));
vi.mock("@/utils/common", () => ({ formatDate: (value: string) => value }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}));

const request = vi.mocked(getOrderDetail);
type Response = Awaited<ReturnType<typeof getOrderDetail>>;
function response(id: number, status = 1): Response {
  return {
    data: {
      data: {
        id,
        order_no: `ORDER-${id}`,
        type: 4,
        status,
        quantity: 1,
        amount: 20,
      },
    },
  } as Response;
}
const props = {
  open: true,
  orderId: 1,
  onOpenChange: vi.fn(),
  renderStatus: (
    order: Pick<API.OrderDetail, "id" | "status">,
    onUpdated: (status: number) => void
  ) => (
    <button onClick={() => onUpdated(2)} type="button">
      Status {order.status}
    </button>
  ),
};
beforeEach(() => {
  request.mockReset();
  props.onOpenChange.mockReset();
});
afterEach(cleanup);

test("clears the previous order while loading and ignores an obsolete response", async () => {
  let resolveOld!: (value: Response) => void;
  request.mockResolvedValueOnce(response(1));
  const view = render(<OrderDetailSheet {...props} />);
  await screen.findByText("ORDER-1");
  request.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve;
      })
  );
  fireEvent.click(screen.getByRole("button", { name: "Refresh detail" }));
  expect(screen.queryByText("ORDER-1")).toBeNull();
  request.mockResolvedValueOnce(response(2));
  view.rerender(<OrderDetailSheet {...props} orderId={2} />);
  await screen.findByText("ORDER-2");
  await act(async () => resolveOld(response(1)));
  expect(screen.queryByText("ORDER-1")).toBeNull();
  expect(screen.getByText("ORDER-2")).toBeTruthy();
});

test("reloads authoritative detail after a status update", async () => {
  request
    .mockResolvedValueOnce(response(1))
    .mockResolvedValueOnce(response(1, 5));
  render(<OrderDetailSheet {...props} />);
  fireEvent.click(await screen.findByRole("button", { name: "Status 1" }));
  await screen.findByRole("button", { name: "Status 5" });
  expect(request).toHaveBeenCalledTimes(2);
});

test.each([{}, { id: 2, order_no: "OTHER-ORDER" }])(
  "rejects incomplete or mismatched order detail: %j",
  async (data) => {
    request.mockResolvedValue({ data: { data } } as Response);
    render(<OrderDetailSheet {...props} />);
    await screen.findByRole("alert");
    expect(screen.queryByText("OTHER-ORDER")).toBeNull();
    expect(screen.queryByText("Financial breakdown")).toBeNull();
  }
);

test("shows an error instead of a loading title and supports retry and close", async () => {
  request
    .mockRejectedValueOnce(new Error("offline"))
    .mockResolvedValueOnce(response(1));
  render(<OrderDetailSheet {...props} />);
  await screen.findByRole("alert");
  expect(screen.queryByText("Loading order data")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  await screen.findByText("ORDER-1");
  const closeButton = screen.getAllByRole("button", { name: /^Close$/ })[0];
  expect(closeButton).toBeDefined();
  if (closeButton) fireEvent.click(closeButton);
  await waitFor(() => expect(props.onOpenChange).toHaveBeenCalledWith(false));
});
