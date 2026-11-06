/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { RegistrationBadge } from "./registration-badge";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
afterEach(cleanup);
test.each([
  ["github", "GitHub 注册"],
  ["email", "邮箱注册"],
  ["admin", "后台创建"],
  [undefined, "来源未知"],
])("shows the original registration source %s", (method, label) => {
  render(<RegistrationBadge method={method} />);
  expect(screen.getByText(label!)).toBeTruthy();
});
