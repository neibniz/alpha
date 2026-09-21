/** @vitest-environment jsdom */
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { filterTrafficLogDetails } from "@workspace/ui/services/admin/log";
import TrafficDetailsPage from "./index";

const route = vi.hoisted(() => ({
  search: { date: "2026-09-01", user_id: "1", subscribe_id: "10" },
}));
vi.mock("@tanstack/react-router", () => ({ useSearch: () => route.search }));
vi.mock("@workspace/ui/services/admin/log", () => ({
  filterTrafficLogDetails: vi.fn(),
}));
vi.mock("@/stores/server", () => ({
  useServer: () => ({ getServerName: () => "Server" }),
}));
vi.mock("@/sections/user/user-detail", () => ({
  UserDetail: () => null,
  UserSubscribeDetail: () => null,
}));
vi.mock("ahooks", () => ({ useSize: () => ({ height: 600, width: 1024 }) }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}));
const request = vi.mocked(filterTrafficLogDetails);
beforeEach(() => {
  route.search = { date: "2026-09-01", user_id: "1", subscribe_id: "10" };
  request.mockReset();
  request.mockResolvedValue({
    data: { data: { list: [], total: 0 } },
  } as Awaited<ReturnType<typeof filterTrafficLogDetails>>);
});
afterEach(cleanup);

test("updates the request and visible filters when route parameters change", async () => {
  const view = render(<TrafficDetailsPage />);
  await waitFor(() =>
    expect(request).toHaveBeenLastCalledWith(
      expect.objectContaining({ user_id: 1, subscribe_id: 10 })
    )
  );
  route.search = { date: "2026-09-02", user_id: "2", subscribe_id: "20" };
  view.rerender(<TrafficDetailsPage />);
  await waitFor(() =>
    expect(request).toHaveBeenLastCalledWith(
      expect.objectContaining({
        date: "2026-09-02",
        user_id: 2,
        subscribe_id: 20,
        page: 1,
      })
    )
  );
  expect(
    (screen.getByRole("textbox", { name: "User ID" }) as HTMLInputElement).value
  ).toBe("2");
});

test("preserves manually edited filters during an ordinary rerender", async () => {
  const view = render(<TrafficDetailsPage />);
  await waitFor(() => expect(request).toHaveBeenCalledTimes(1));
  fireEvent.change(screen.getByRole("textbox", { name: "User ID" }), {
    target: { value: "7" },
  });
  await waitFor(() =>
    expect(request).toHaveBeenLastCalledWith(
      expect.objectContaining({ user_id: "7" })
    )
  );
  view.rerender(<TrafficDetailsPage />);
  expect(
    (screen.getByRole("textbox", { name: "User ID" }) as HTMLInputElement).value
  ).toBe("7");
  expect(request).toHaveBeenCalledTimes(2);
});
