/** @vitest-environment jsdom */
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  MASKED_LOG_VISIBILITY,
  SensitiveLogToggle,
  type SensitiveLogVisibility,
} from "./sensitive-log-toggle";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (
      _key: string,
      fallback: string | { defaultValue: string; count: number }
    ) =>
      typeof fallback === "string"
        ? fallback
        : fallback.defaultValue.replace("{{count}}", String(fallback.count)),
  }),
}));
beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn();
  Element.prototype.hasPointerCapture = vi.fn(() => false);
  Element.prototype.setPointerCapture = vi.fn();
  Element.prototype.releasePointerCapture = vi.fn();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
function Harness() {
  const [value, setValue] = useState<SensitiveLogVisibility>(
    MASKED_LOG_VISIBILITY
  );
  return <SensitiveLogToggle value={value} onChange={setValue} />;
}
test.each([
  5, 15, 30,
])("reveals logs for %i minutes, requires a reason, then masks on expiry", async (minutes) => {
  render(<Harness />);
  fireEvent.click(screen.getByRole("switch", { name: "完整信息" }));
  expect(
    (screen.getByRole("button", { name: "显示完整信息" }) as HTMLButtonElement)
      .disabled
  ).toBe(true);
  fireEvent.keyDown(screen.getByRole("combobox", { name: "访问原因" }), {
    key: "ArrowDown",
  });
  fireEvent.click(screen.getByRole("option", { name: "客户支持" }));
  fireEvent.keyDown(screen.getByRole("combobox", { name: "查看时限" }), {
    key: "ArrowDown",
  });
  fireEvent.click(screen.getByRole("option", { name: `${minutes} 分钟` }));
  fireEvent.click(screen.getByRole("button", { name: "显示完整信息" }));
  const toggle = screen.getByRole("switch", { name: "完整信息" });
  expect(toggle.getAttribute("aria-checked")).toBe("true");
  vi.useFakeTimers();
  const start = Date.now();
  await act(async () => {
    vi.setSystemTime(start + (minutes - 1) * 60_000);
    fireEvent.focus(window);
  });
  expect(toggle.getAttribute("aria-checked")).toBe("true");
  await act(async () => {
    vi.setSystemTime(start + minutes * 60_000 + 1000);
    fireEvent.focus(window);
  });
  expect(toggle.getAttribute("aria-checked")).toBe("false");
});
