import { Outlet } from "@tanstack/react-router";
import {
  SidebarInset,
  SidebarProvider,
} from "@workspace/ui/components/sidebar";
import { getCookie } from "@workspace/ui/lib/cookies";
import { type CSSProperties, useEffect, useState } from "react";
import { Header } from "@/layout/header";
import { SidebarLeft } from "./sidebar-left";

export default function DashboardLayout() {
  const [open, setOpen] = useState(true);

  useEffect(() => {
    const sidebarState = getCookie("sidebar_state");
    if (sidebarState !== undefined) {
      setOpen(sidebarState === "true");
    }
  }, []);

  return (
    <SidebarProvider
      className="admin-shell"
      defaultOpen={open}
      style={
        {
          "--sidebar-width": "14rem",
          "--sidebar-width-icon": "3.25rem",
        } as CSSProperties
      }
    >
      <SidebarLeft />
      <SidebarInset className="admin-workspace relative min-w-0 flex-grow overflow-hidden">
        <Header />
        <main className="admin-main flex-grow overflow-auto">
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
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
