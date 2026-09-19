import { Outlet, useLocation } from "@tanstack/react-router";
import {
  SidebarInset,
  SidebarProvider,
} from "@workspace/ui/components/sidebar";
import { getCookie } from "@workspace/ui/lib/cookies";
import { type CSSProperties, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Header } from "@/layout/header";
import { SidebarLeft } from "./sidebar-left";

export default function DashboardLayout() {
  const [open, setOpen] = useState(
    () => getCookie("sidebar_state") !== "false"
  );
  const { t } = useTranslation("components");
  const pathname = useLocation({ select: (location) => location.pathname });
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (pathname) contentRef.current?.scrollTo({ top: 0, left: 0 });
  }, [pathname]);

  return (
    <SidebarProvider
      className="admin-shell"
      onOpenChange={setOpen}
      open={open}
      style={
        {
          "--sidebar-width": "15.5rem",
          "--sidebar-width-icon": "3.5rem",
        } as CSSProperties
      }
    >
      <button
        className="admin-skip-link"
        onClick={() => {
          contentRef.current?.focus();
        }}
        type="button"
      >
        {t("sidebar.skipToContent", "Skip to content")}
      </button>
      <SidebarLeft />
      <SidebarInset className="admin-workspace relative min-w-0 flex-grow overflow-hidden">
        <Header />
        <div
          className="admin-main flex-grow overflow-auto"
          id="admin-content"
          ref={contentRef}
          tabIndex={-1}
        >
          <div className="admin-page-stage" data-slot="admin-page-stage">
            <Outlet />
          </div>
          <footer className="px-6 pb-4 text-right text-[11px] text-muted-foreground">
            <a
              className="underline-offset-4 hover:text-foreground hover:underline"
              href="https://db-ip.com"
              rel="noreferrer"
              target="_blank"
            >
              IP Geolocation by DB-IP
            </a>
          </footer>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
