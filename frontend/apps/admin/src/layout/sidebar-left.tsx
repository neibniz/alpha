import { Link, useLocation } from "@tanstack/react-router";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@workspace/ui/components/avatar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@workspace/ui/components/collapsible";
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
  SidebarRail,
  useSidebar,
} from "@workspace/ui/components/sidebar";
import { ChevronDown } from "lucide-react";
import { useEffect, useState, type ComponentProps } from "react";
import { useTranslation } from "react-i18next";
import { useGlobalStore } from "@/stores/global";
import { isActiveNavUrl, type NavGroup, useNavs } from "./navs";

function NavigationGroup({
  group,
  pathname,
}: {
  group: NavGroup;
  pathname: string;
}) {
  const { isMobile, setOpenMobile, state } = useSidebar();
  const active = group.items.some((item) => isActiveNavUrl(item.url, pathname));
  const compact =
    group.items.length > 6 || group.items[0]?.url === "/dashboard/system";
  const [open, setOpen] = useState(active || !compact);
  const collapsed = state === "collapsed" && !isMobile;

  useEffect(() => {
    if (active) setOpen(true);
  }, [active]);

  return (
    <Collapsible asChild onOpenChange={setOpen} open={collapsed || open}>
      <SidebarGroup className="py-1.5">
        {group.items.length > 1 ? (
          <SidebarGroupLabel asChild>
            <CollapsibleTrigger className="group/nav-heading mb-1 w-full justify-between">
              <span>{group.title}</span>
              <ChevronDown className="size-3 transition-transform group-data-[state=closed]/nav-heading:-rotate-90" />
            </CollapsibleTrigger>
          </SidebarGroupLabel>
        ) : null}
        <CollapsibleContent>
          <SidebarGroupContent>
            <SidebarMenu className="gap-0.5">
              {group.items.map((item) => {
                const selected = isActiveNavUrl(item.url, pathname);
                const Icon = item.icon;
                return (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      asChild
                      className="h-9"
                      isActive={selected}
                      tooltip={item.title}
                    >
                      <Link
                        aria-current={selected ? "page" : undefined}
                        onClick={() => {
                          if (isMobile) setOpenMobile(false);
                        }}
                        to={item.url}
                      >
                        <Icon aria-hidden />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}

export function SidebarLeft(props: ComponentProps<typeof Sidebar>) {
  const pathname = useLocation({ select: (location) => location.pathname });
  const navs = useNavs();
  const site = useGlobalStore((store) => store.common.site);
  const { isMobile, setOpenMobile } = useSidebar();
  const { t } = useTranslation("components");
  const settings = navs.at(-1);
  const siteName = site.site_name || "Alpha";

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="h-16 justify-center border-b px-4 group-data-[collapsible=icon]:px-2">
        <Link
          className="admin-brand"
          onClick={() => {
            if (isMobile) setOpenMobile(false);
          }}
          to="/dashboard"
        >
          <Avatar className="size-8 shrink-0 rounded-md">
            <AvatarImage
              alt=""
              className="object-contain"
              src={site.site_logo || `${import.meta.env.BASE_URL}favicon.png`}
            />
            <AvatarFallback className="rounded-md">
              {siteName.slice(0, 1)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 group-data-[collapsible=icon]:hidden">
            <p className="truncate font-semibold text-sm">{siteName}</p>
            <p className="truncate text-xs text-muted-foreground">
              {t("site.adminConsole", "Operations console")}
            </p>
          </div>
        </Link>
      </SidebarHeader>
      <SidebarContent className="gap-1 px-2 py-3">
        <nav aria-label={t("sidebar.navigation", "Main navigation")}>
          {navs.slice(0, -1).map((group) => (
            <NavigationGroup
              group={group}
              key={group.items[0]?.url}
              pathname={pathname}
            />
          ))}
        </nav>
      </SidebarContent>
      <SidebarFooter className="border-t px-2 py-2">
        {settings ? (
          <NavigationGroup group={settings} pathname={pathname} />
        ) : null}
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
