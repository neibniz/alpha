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

test("blocks invalid Settings JSON and saves a finalRules array without losing freedom fields", async () => {
  Element.prototype.scrollIntoView = vi.fn();
  Element.prototype.hasPointerCapture = vi.fn(() => false);
  Element.prototype.setPointerCapture = vi.fn();
  Element.prototype.releasePointerCapture = vi.fn();
  const onSubmit = vi.fn(async (_value: unknown) => false);
  render(
    <XrayTemplateForm
      title="Outbound template"
      trigger={<button type="button">Edit Outbound</button>}
      initialValues={{
        type: "outbound",
        name: "Interdiction",
        config: {
          tag: "interdiction-out",
          protocol: "freedom",
          settings: {
            domainStrategy: "AsIs",
            redirect: "127.0.0.1:5080",
            userLevel: 0,
          },
        },
      }}
      onSubmit={onSubmit}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: "Edit Outbound" }));
  fireEvent.click(await screen.findByRole("button", { name: "添加覆盖字段" }));
  fireEvent.change(screen.getByRole("textbox", { name: "字段名" }), {
    target: { value: "finalRules" },
  });
  fireEvent.keyDown(screen.getByRole("combobox", { name: "字段类型" }), {
    key: " ",
  });
  fireEvent.click(screen.getByRole("option", { name: "JSON" }));
  const input = screen.getByRole("textbox", { name: "finalRules JSON" });
  fireEvent.change(input, { target: { value: "[" } });
  fireEvent.click(screen.getByRole("button", { name: "Save template" }));
  await screen.findAllByText("Settings JSON 不是合法 JSON。");
  expect(onSubmit).not.toHaveBeenCalled();
  const rules = [
    { action: "allow", network: "tcp", ip: ["127.0.0.1/32"], port: "5080" },
  ];
  fireEvent.change(input, { target: { value: JSON.stringify(rules) } });
  fireEvent.click(screen.getByRole("button", { name: "Save template" }));
  await waitFor(() => expect(onSubmit).toHaveBeenCalledOnce());
  expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
    config: {
      tag: "interdiction-out",
      protocol: "freedom",
      settings: {
        domainStrategy: "AsIs",
        redirect: "127.0.0.1:5080",
        userLevel: 0,
        finalRules: rules,
      },
    },
  });
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
  expect(screen.getByRole("textbox", { name: /^ALPN$/ })).toBeTruthy();
  expect(screen.getByPlaceholderText("AA:BB:CC:...")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "添加证书" }));
  expect(screen.getByPlaceholderText("/etc/ssl/fullchain.pem")).toBeTruthy();
  expect(screen.getByPlaceholderText("/etc/ssl/privkey.pem")).toBeTruthy();
  expect(
    screen.getByPlaceholderText(/-----BEGIN CERTIFICATE-----/)
  ).toBeTruthy();
});

test.each([
  "inbound",
  "outbound",
] as const)("edits and saves HY2 ALPN values for %s TLS", async (type) => {
  const onSubmit = vi.fn();
  render(
    <XrayTemplateForm
      title="HY2 template"
      trigger={<button type="button">Edit HY2</button>}
      initialValues={{
        type,
        name: "HY2",
        config: {
          tag: "hy2",
          protocol: "hysteria",
          ...(type === "inbound" ? { port: 38_003 } : {}),
          settings:
            type === "inbound"
              ? { version: 2, users: [{ auth: "test-password" }] }
              : { version: 2, address: "edge.example.com", port: 38_003 },
          streamSettings: {
            network: "hysteria",
            security: "tls",
            hysteriaSettings: { version: 2, auth: "test-password" },
            tlsSettings: {
              alpn: ["h3"],
              minVersion: "1.3",
              certificates: [
                { certificateFile: "/cert.pem", keyFile: "/key.pem" },
              ],
            },
          },
        },
      }}
      onSubmit={onSubmit}
    />
  );
  fireEvent.click(screen.getByRole("button", { name: "Edit HY2" }));
  const alpn = await screen.findByRole("textbox", { name: /^ALPN$/ });
  expect((alpn as HTMLInputElement).value).toBe("h3");
  fireEvent.change(alpn, { target: { value: "h3, custom-protocol" } });
  const preview = screen.getByRole("complementary").querySelector("textarea")!;
  await waitFor(() =>
    expect(JSON.parse(preview.value).streamSettings.tlsSettings.alpn).toEqual([
      "h3",
      "custom-protocol",
    ])
  );
  fireEvent.change(alpn, { target: { value: "" } });
  await waitFor(() =>
    expect(
      JSON.parse(preview.value).streamSettings.tlsSettings
    ).not.toHaveProperty("alpn")
  );
  fireEvent.change(alpn, { target: { value: "h3, custom-protocol" } });
  fireEvent.click(screen.getByRole("button", { name: "Save template" }));
  await waitFor(() => expect(onSubmit).toHaveBeenCalled());
  expect(
    onSubmit.mock.calls[0]?.[0].config.streamSettings.tlsSettings
  ).toMatchObject({
    alpn: ["h3", "custom-protocol"],
    minVersion: "1.3",
  });
});
