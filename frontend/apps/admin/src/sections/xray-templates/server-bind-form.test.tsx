/** @vitest-environment jsdom */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import ServerXrayTemplateBindForm from "./server-bind-form";

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
