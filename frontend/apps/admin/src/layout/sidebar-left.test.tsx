/** @vitest-environment jsdom */

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
  SidebarProvider,
  SidebarTrigger,
} from "@workspace/ui/components/sidebar";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SidebarLeft } from "./sidebar-left";

const context = vi.hoisted(() => ({ pathname: "/dashboard", mobile: false }));
vi.mock("@workspace/ui/hooks/use-mobile", () => ({
  useIsMobile: () => context.mobile,
}));
vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    children,
    onClick,
    ...props
  }: ComponentProps<"a"> & { to: string }) => (
    <a
      {...props}
      href={to}
      onClick={(event) => {
        event.preventDefault();
        onClick?.(event);
      }}
    >
      {children}
    </a>
  ),
  useLocation: () => context.pathname,
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
vi.mock("@/stores/global", () => ({
  useGlobalStore: (selector: (store: unknown) => unknown) =>
    selector({ common: { site: { site_name: "Test Brand" } } }),
}));
afterEach(() => {
  cleanup();
  context.pathname = "/dashboard";
  context.mobile = false;
});

const Navigation = () => (
  <SidebarProvider>
    <SidebarLeft />
    <SidebarTrigger />
  </SidebarProvider>
);

describe("Admin navigation", () => {
  it("keeps all main workspaces directly reachable and uses the configured brand", () => {
    render(<Navigation />);
    expect(screen.getByText("Test Brand")).toBeTruthy();
    for (const name of [
      "Dashboard",
      "Server Management",
      "Node Management",
      "User Management",
      "Order Management",
      "Product Management",
      "Ticket Management",
    ]) {
      expect(screen.getByRole("link", { name })).toBeTruthy();
    }
    expect(
      screen
        .getByRole("link", { name: "Dashboard" })
        .getAttribute("aria-current")
    ).toBe("page");
  });

  it("expands the matching log group on direct navigation", () => {
    const view = render(<Navigation />);
    expect(screen.queryByRole("link", { name: "Proxy Access" })).toBeNull();
    context.pathname = "/dashboard/log/proxy-access";
    view.rerender(<Navigation />);
    expect(
      screen
        .getByRole("link", { name: "Proxy Access" })
        .getAttribute("aria-current")
    ).toBe("page");
  });

  it("closes the mobile drawer after selecting a page", () => {
    context.mobile = true;
    render(<Navigation />);
    fireEvent.click(screen.getByRole("button", { name: "Toggle Sidebar" }));
    expect(screen.getByRole("dialog")).toBeTruthy();
    fireEvent.click(screen.getByRole("link", { name: "Server Management" }));
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
