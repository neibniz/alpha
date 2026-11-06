/** @vitest-environment jsdom */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import ServerXrayTemplateBindForm from "./server-bind-form";
import {
  queryServerXrayTemplateList,
  queryXrayTemplateCatalog,
} from "@workspace/ui/services/admin/server";

vi.mock("@workspace/ui/services/admin/server", () => ({
  queryXrayTemplateCatalog: vi.fn(async () => ({
    data: { data: { list: [] } },
  })),
  queryServerXrayTemplateList: vi.fn(async () => ({
    data: { data: { list: [] } },
  })),
  previewServerXrayConfig: vi.fn(async () => ({ data: { data: {} } })),
  bindServerXrayTemplates: vi.fn(),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
let client: QueryClient;
afterEach(() => {
  cleanup();
  client.clear();
});

test("uses a single narrow-screen scroll region and keeps the footer outside it", async () => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ServerXrayTemplateBindForm
        open
        server={{ id: 1, name: "Test server" } as API.Server}
      />
    </QueryClientProvider>
  );
  await screen.findByRole("heading", { name: /Test server/ });
  const body = screen.getByTestId("xray-binding-scroll");
  expect(body.className).toContain("min-h-0");
  expect(body.className).toContain("overflow-y-auto");
  expect(body.className.split(" ")).not.toContain("overflow-hidden");
  const sidebar = body.querySelector("aside")!;
  expect(sidebar.className.split(" ")).not.toContain("overflow-y-auto");
  expect(sidebar.className).toContain("lg:overflow-y-auto");
  expect(body.contains(screen.getByRole("button", { name: /^Save$/ }))).toBe(
    false
  );
  expect(body.closest("[data-slot=sheet-content]")?.className).toContain(
    "h-dvh"
  );
});

test("explains why a bound but globally disabled template is not deployed", async () => {
  vi.mocked(queryXrayTemplateCatalog).mockResolvedValueOnce({
    data: {
      data: {
        list: [
          {
            id: 2,
            name: "Paused outbound",
            type: "outbound",
            enabled: false,
            config: { tag: "paused", protocol: "freedom" },
            default_variables: {},
            subscription_meta: {},
          },
        ],
      },
    },
  } as Awaited<ReturnType<typeof queryXrayTemplateCatalog>>);
  vi.mocked(queryServerXrayTemplateList).mockResolvedValueOnce({
    data: {
      data: {
        list: [
          {
            template_id: 2,
            enabled: true,
            alias: "paused",
            variables: {},
            subscription_variables: {},
          },
        ],
        desired_config_hash: "same-hash",
        status: {
          status: "online",
          config_sync_status: "ok",
          config_apply_status: "ok",
          running_config_hash: "same-hash",
        },
      },
    },
  } as Awaited<ReturnType<typeof queryServerXrayTemplateList>>);
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ServerXrayTemplateBindForm
        open
        server={{ id: 1, name: "Test server" } as API.Server}
      />
    </QueryClientProvider>
  );
  await screen.findByText("模板已停用");
  fireEvent.mouseDown(screen.getByRole("tab", { name: "Outbound" }), {
    button: 0,
    ctrlKey: false,
  });
  await screen.findByText("全局模板已停用");
  expect(screen.getByText(/当前修改可以保存，但不会下发到节点/)).toBeTruthy();
  expect(
    screen
      .getByRole("switch", { name: "Paused outbound 服务器绑定" })
      .getAttribute("aria-checked")
  ).toBe("true");
});
