/** @vitest-environment jsdom */
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import XrayTemplateForm from "./template-form";
vi.mock("./generate-tls-certificate", () => ({
  generateTLSCertificate: vi.fn(async () => ({
    certificate: ["TEST CERTIFICATE"],
    key: ["TEST PRIVATE KEY"],
    pinnedPeerCertSha256: "ab".repeat(32),
  })),
}));

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

test("appends the generated pair and fingerprint without overwriting existing certificate files", async () => {
  const oldPin = "cd".repeat(32);
  render(
    <XrayTemplateForm
      title="TLS template"
      trigger={<button type="button">Edit TLS</button>}
      initialValues={{
        type: "inbound",
        name: "TLS",
        config: {
          protocol: "vless",
          port: 443,
          streamSettings: {
            network: "raw",
            security: "tls",
            tlsSettings: {
              serverName: "example.com",
              pinnedPeerCertSha256: oldPin,
              certificates: [
                { certificateFile: "/cert.pem", keyFile: "/key.pem" },
              ],
            },
          },
        },
      }}
      onSubmit={vi.fn()}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: "Edit TLS" }));
  fireEvent.click(
    await screen.findByRole("button", { name: "生成自签名证书" })
  );
  expect(
    (screen.getByRole("textbox", { name: "域名或 IP" }) as HTMLTextAreaElement)
      .value
  ).toBe("example.com");
  fireEvent.click(screen.getByRole("button", { name: "生成并填入" }));
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "生成并填入" })).toBeNull()
  );
  const preview = screen.getByRole("complementary").querySelector("textarea");
  expect(preview).toBeTruthy();
  const config = JSON.parse(preview?.value || "{}");
  expect(config.streamSettings.tlsSettings.certificates).toEqual([
    { certificateFile: "/cert.pem", keyFile: "/key.pem" },
    {
      certificate: ["TEST CERTIFICATE"],
      key: ["TEST PRIVATE KEY"],
      usage: "encipherment",
    },
  ]);
  expect(config.streamSettings.tlsSettings.pinnedPeerCertSha256).toBe(
    `${oldPin},${"ab".repeat(32)}`
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
