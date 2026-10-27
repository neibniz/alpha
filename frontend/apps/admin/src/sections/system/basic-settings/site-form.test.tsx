/** @vitest-environment jsdom */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import {
  getSiteConfig,
  updateSiteConfig,
} from "@workspace/ui/services/admin/system";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import SiteConfig from "./site-form";

vi.mock("@workspace/ui/services/admin/system", () => ({
  getSiteConfig: vi.fn(),
  updateSiteConfig: vi.fn(),
}));
vi.mock("@workspace/ui/composed/upload-image", () => ({
  UploadImage: () => null,
}));
vi.mock("@workspace/ui/composed/editor/json", () => ({
  JSONEditor: ({
    value,
    onChange,
  }: {
    value?: string;
    onChange?: (value: string) => void;
  }) => (
    <div>
      JSON editor
      <textarea
        aria-label="Custom JSON"
        value={value}
        onChange={(event) => onChange?.(event.target.value)}
      />
    </div>
  ),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
const request = vi.mocked(getSiteConfig);
let client: QueryClient;
beforeEach(() => {
  request.mockReset();
  vi.mocked(updateSiteConfig).mockReset();
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
});
afterEach(() => {
  cleanup();
  client.clear();
});
const response = {
  data: {
    data: {
      site_name: "Example",
      site_logo: "",
      site_desc: "",
      keywords: "",
      host: "example.com",
      custom_html: "",
      custom_data: "{}",
    },
  },
} as Awaited<ReturnType<typeof getSiteConfig>>;
function openForm() {
  render(
    <QueryClientProvider client={client}>
      <SiteConfig />
    </QueryClientProvider>
  );
  fireEvent.click(screen.getByRole("button", { name: /Site Configuration/ }));
}

test("shows load errors without an editable empty form and supports retry", async () => {
  request
    .mockRejectedValueOnce(new Error("offline"))
    .mockResolvedValueOnce(response);
  openForm();
  await screen.findByRole("alert");
  expect(screen.queryByRole("textbox", { name: "Site Name" })).toBeNull();
  expect(
    (screen.getByRole("button", { name: "Save Settings" }) as HTMLButtonElement)
      .disabled
  ).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  await screen.findByDisplayValue("Example");
});

test("preserves edits between tabs and mounts the code editor only on demand", async () => {
  request.mockResolvedValue(response);
  openForm();
  const name = await screen.findByDisplayValue("Example");
  expect(screen.queryByText("JSON editor")).toBeNull();
  fireEvent.change(name, { target: { value: "Updated name" } });
  fireEvent.mouseDown(
    screen.getByRole("tab", { name: "Domain & extensions" }),
    { button: 0, ctrlKey: false }
  );
  await screen.findByText("JSON editor");
  fireEvent.mouseDown(screen.getByRole("tab", { name: "Identity" }), {
    button: 0,
    ctrlKey: false,
  });
  await screen.findByDisplayValue("Updated name");
  expect(updateSiteConfig).not.toHaveBeenCalled();
});

test("switches back to the identity tab for a missing site name", async () => {
  request.mockResolvedValue(response);
  openForm();
  fireEvent.change(await screen.findByDisplayValue("Example"), {
    target: { value: "" },
  });
  fireEvent.mouseDown(
    screen.getByRole("tab", { name: "Domain & extensions" }),
    { button: 0, ctrlKey: false }
  );
  fireEvent.click(screen.getByRole("button", { name: "Save Settings" }));
  await waitFor(() =>
    expect(
      screen
        .getByRole("tab", { name: "Identity" })
        .getAttribute("aria-selected")
    ).toBe("true")
  );
  expect(updateSiteConfig).not.toHaveBeenCalled();
});

test.each([
  "",
  "{}",
])("saves JSON-only changes without waiting for blur (initial: %j)", async (customData) => {
  request.mockResolvedValue({
    ...response,
    data: {
      ...response.data,
      data: { ...response.data.data!, custom_data: customData },
    },
  });
  vi.mocked(updateSiteConfig).mockResolvedValue(
    {} as Awaited<ReturnType<typeof updateSiteConfig>>
  );
  openForm();
  await screen.findByDisplayValue("Example");
  fireEvent.mouseDown(
    screen.getByRole("tab", { name: "Domain & extensions" }),
    {
      button: 0,
      ctrlKey: false,
    }
  );
  const editor = await screen.findByRole("textbox", { name: "Custom JSON" });
  fireEvent.focus(editor);
  fireEvent.change(editor, {
    target: { value: '{"website":"https://example.org"}' },
  });
  const save = screen.getByRole("button", {
    name: "Save Settings",
  }) as HTMLButtonElement;
  await waitFor(() => expect(save.disabled).toBe(false));
  fireEvent.click(save);
  await waitFor(() =>
    expect(updateSiteConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        custom_data: '{"website":"https://example.org"}',
      })
    )
  );
});
