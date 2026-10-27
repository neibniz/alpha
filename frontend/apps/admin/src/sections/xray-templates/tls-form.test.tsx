/** @vitest-environment jsdom */
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import XrayTemplateForm from "./template-form";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (_key: string, fallback: string) => fallback,
    i18n: { language: "en-US" },
  }),
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

test.each([
  "inbound",
  "outbound",
] as const)("shows certificate and pinned hash controls for %s TLS", async (type) => {
  render(
    <XrayTemplateForm
      title="TLS template"
      trigger={<button type="button">Edit TLS</button>}
      initialValues={{
        type,
        name: "TLS",
        config: {
          tag: "tls",
          protocol: "vless",
          port: 443,
          settings: { decryption: "none" },
          streamSettings: { network: "raw", security: "tls" },
        },
      }}
      onSubmit={vi.fn()}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: "Edit TLS" }));
  await screen.findByText("TLS 证书与私钥");
  expect(screen.getByPlaceholderText("AA:BB:CC:...")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "添加证书" }));
  expect(screen.getByPlaceholderText("/etc/ssl/fullchain.pem")).toBeTruthy();
  expect(screen.getByPlaceholderText("/etc/ssl/privkey.pem")).toBeTruthy();
  expect(
    screen.getByPlaceholderText(/-----BEGIN CERTIFICATE-----/)
  ).toBeTruthy();
});
