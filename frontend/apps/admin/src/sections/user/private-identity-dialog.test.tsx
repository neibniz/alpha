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
import { revealUserPrivateIdentity } from "@workspace/ui/services/admin/user";
import { PrivateIdentityDialog } from "./private-identity-dialog";

vi.mock("@workspace/ui/services/admin/user", () => ({
  revealUserPrivateIdentity: vi.fn(),
}));
vi.mock("@/utils/common", () => ({
  formatDate: (value: number) => String(value),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (_key: string, fallback: unknown) =>
      typeof fallback === "string"
        ? fallback
        : (fallback as { defaultValue: string }).defaultValue,
  }),
}));
const request = vi.mocked(revealUserPrivateIdentity);
const response = () =>
  ({
    data: {
      data: {
        user_id: 9,
        email: "full@example.test",
        auth_methods: [],
        revealed_at: Date.now(),
        expires_at: Date.now() + 15 * 60_000,
      },
    },
  }) as Awaited<ReturnType<typeof revealUserPrivateIdentity>>;
beforeEach(() => {
  request.mockReset();
  Element.prototype.scrollIntoView = vi.fn();
  Element.prototype.hasPointerCapture = vi.fn(() => false);
  Element.prototype.setPointerCapture = vi.fn();
  Element.prototype.releasePointerCapture = vi.fn();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
function openAndAuthorize() {
  render(
    <PrivateIdentityDialog
      userId={9}
      trigger={<button type="button">View email</button>}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: "View email" }));
  fireEvent.keyDown(screen.getByRole("combobox", { name: "Access reason" }), {
    key: "ArrowDown",
  });
  fireEvent.click(screen.getByRole("option", { name: "Account verification" }));
  fireEvent.click(screen.getByRole("button", { name: "Reveal private data" }));
}
test("shows full email for the selected duration and clears it on expiry", async () => {
  request.mockResolvedValueOnce(response());
  openAndAuthorize();
  await screen.findByText("full@example.test");
  expect(request).toHaveBeenCalledWith({
    user_id: 9,
    reason: "account_verification",
    duration_minutes: 15,
  });
  vi.useFakeTimers();
  await act(async () => {
    vi.setSystemTime(Date.now() + 16 * 60_000);
    fireEvent.focus(window);
  });
  expect(screen.queryByText("full@example.test")).toBeNull();
});
test("discards a response arriving after the dialog is closed", async () => {
  let resolve!: (
    value: Awaited<ReturnType<typeof revealUserPrivateIdentity>>
  ) => void;
  request.mockImplementationOnce(
    () =>
      new Promise((done) => {
        resolve = done;
      })
  );
  openAndAuthorize();
  await waitFor(() => expect(request).toHaveBeenCalledTimes(1));
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  await act(async () => {
    resolve(response());
  });
  fireEvent.click(screen.getByRole("button", { name: "View email" }));
  expect(screen.queryByText("full@example.test")).toBeNull();
});
