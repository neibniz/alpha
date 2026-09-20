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
import { filterProxyAccessLog } from "@workspace/ui/services/admin/log";
import ProxyAccessLogPage from "./index";

vi.mock("@workspace/ui/services/admin/log", () => ({
  filterProxyAccessLog: vi.fn(),
}));
vi.mock("@/stores/server", () => ({
  useServer: () => ({ getServerName: () => "Server" }),
}));
vi.mock("@/sections/user/user-detail", () => ({
  UserDetail: () => null,
  UserSubscribeDetail: () => null,
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: unknown) =>
      typeof fallback === "string" ? fallback : key,
  }),
}));
const request = vi.mocked(filterProxyAccessLog);
const response = (destination: string) =>
  ({
    data: {
      data: {
        list: [
          {
            event_id: destination,
            timestamp: Date.now(),
            destination,
            network: "tcp",
            status: "accepted",
          },
        ],
        next_cursor: "",
      },
    },
  }) as Awaited<ReturnType<typeof filterProxyAccessLog>>;
beforeEach(() => request.mockReset());
afterEach(cleanup);

test("submits filters with the form and clears results from the previous query", async () => {
  request.mockResolvedValueOnce(response("old.example.com"));
  request.mockRejectedValueOnce(new Error("internal secret"));
  render(<ProxyAccessLogPage />);
  await screen.findByText("old.example.com");
  fireEvent.change(screen.getByLabelText("Destination"), {
    target: { value: "new.example.com" },
  });
  fireEvent.submit(screen.getByLabelText("Destination").closest("form")!);
  await screen.findByRole("alert");
  expect(screen.queryByText("old.example.com")).toBeNull();
  expect(screen.queryByText("internal secret")).toBeNull();
  expect(request.mock.lastCall?.[0]).toMatchObject({
    destination: "new.example.com",
  });
});

test("offers an inline retry after loading fails", async () => {
  request.mockRejectedValueOnce(new Error("failed"));
  request.mockResolvedValueOnce(response("recovered.example.com"));
  render(<ProxyAccessLogPage />);
  fireEvent.click(await screen.findByRole("button", { name: "Retry" }));
  await screen.findByText("recovered.example.com");
  expect(screen.queryByRole("alert")).toBeNull();
});

test("ignores a response belonging to an unmounted page", async () => {
  let finish!: (value: ReturnType<typeof response>) => void;
  request.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      })
  );
  const view = render(<ProxyAccessLogPage />);
  await waitFor(() => expect(request).toHaveBeenCalledTimes(1));
  view.unmount();
  request.mockResolvedValueOnce(response("current.example.com"));
  render(<ProxyAccessLogPage />);
  await screen.findByText("current.example.com");
  await act(async () => {
    finish(response("stale.example.com"));
  });
  expect(screen.queryByText("stale.example.com")).toBeNull();
});
