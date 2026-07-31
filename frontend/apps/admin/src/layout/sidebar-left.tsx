import { Link, useLocation } from "@tanstack/react-router";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@workspace/ui/components/sidebar";
import { Badge } from "@workspace/ui/components/badge";
import { ChevronRight } from "lucide-react";
import type React from "react";
import { useTranslation } from "react-i18next";
import {
  findActiveNavGroup,
  isActiveNavUrl,
  type NavGroup,
  useNavs,
} from "./navs";

function DomainLinks({
  domain,
  pathname,
}: {
  domain: NavGroup;
  pathname: string;
}) {
  return (
    <SidebarMenu className="gap-1">
      {domain.items.map((item) => {
        const active = isActiveNavUrl(item.url, pathname);
        const Icon = item.icon;
        return (
          <SidebarMenuItem key={item.url}>
            <SidebarMenuButton
              asChild
              className="h-10"
              isActive={active}
              tooltip={item.title}
            >
              <Link to={item.url}>
                <Icon aria-hidden />
                <span className="truncate">{item.title}</span>
                {active ? (
                  <ChevronRight aria-hidden className="ml-auto" />
                ) : null}
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        );
      })}
    </SidebarMenu>
  );
}

export function SidebarLeft(props: React.ComponentProps<typeof Sidebar>) {
  const pathname = useLocation({ select: (location) => location.pathname });
  const navs = useNavs();
  const activeDomain = findActiveNavGroup(navs, pathname);
  const { isMobile } = useSidebar();
  const { t } = useTranslation("components");
  const DomainIcon = activeDomain.icon;

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader className="gap-3 px-4 py-4">
        <div className="flex items-center gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-sidebar-primary text-sidebar-primary-foreground">
            <DomainIcon aria-hidden />
          </div>
          <div className="min-w-0">
            <p className="truncate font-semibold text-sm">
              {activeDomain.title}
            </p>
            <p className="truncate text-muted-foreground text-xs">
              {activeDomain.description}
            </p>
          </div>
        </div>
      </SidebarHeader>
      <SidebarSeparator />
      <SidebarContent className="px-2 py-3">
        {isMobile ? (
          <SidebarGroup>
            <SidebarGroupLabel>
              {t("sidebar.workspaces", "Workspaces")}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu className="gap-1">
                {navs.map((domain) => {
                  const Icon = domain.icon;
                  const active = domain.title === activeDomain.title;
                  return (
                    <SidebarMenuItem key={domain.title}>
                      <SidebarMenuButton asChild isActive={active}>
                        <Link to={domain.items[0]?.url || "/dashboard"}>
                          <Icon aria-hidden />
                          <span>{domain.title}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ) : null}
        {isMobile ? <SidebarSeparator /> : null}
        <SidebarGroup>
          <SidebarGroupLabel>
            {t("sidebar.currentWorkspace", "Current workspace")}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <DomainLinks domain={activeDomain} pathname={pathname} />
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className="px-4 py-4">
        <div className="flex items-center justify-between gap-2 text-muted-foreground text-xs">
          <span>{t("site.adminConsole", "Operations console")}</span>
          <Badge variant="outline">Alpha</Badge>
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
