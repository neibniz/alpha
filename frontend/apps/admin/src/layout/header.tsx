import { Link, useLocation } from "@tanstack/react-router";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@workspace/ui/components/breadcrumb";
import { Separator } from "@workspace/ui/components/separator";
import { SidebarTrigger } from "@workspace/ui/components/sidebar";
import { LanguageSwitch } from "@workspace/ui/composed/language-switch";
import { ThemeSwitch } from "@workspace/ui/composed/theme-switch";
import { useTranslation } from "react-i18next";
import { CommandMenu } from "./command-menu";
import { findActiveNavGroup, findNavByUrl, useNavs } from "./navs";
import TimezoneSwitch from "./timezone-switch";
import { UserNav } from "./user-nav";

export function Header() {
  const pathname = useLocation({ select: (location) => location.pathname });
  const navs = useNavs();
  const group = findActiveNavGroup(navs, pathname);
  const page = findNavByUrl(navs, pathname).at(-1);
  const { t } = useTranslation("components");

  return (
    <header className="admin-topbar">
      <div className="flex min-w-0 items-center gap-3">
        <SidebarTrigger
          aria-label={t("sidebar.toggle", "Toggle navigation")}
          className="shrink-0"
        />
        <Separator className="h-4" orientation="vertical" />
        <Breadcrumb className="min-w-0">
          <BreadcrumbList className="flex-nowrap">
            <BreadcrumbItem className="hidden shrink-0 md:block">
              <BreadcrumbLink asChild>
                <Link to={group.items[0]?.url || "/dashboard"}>
                  {group.title}
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem className="min-w-0">
              <BreadcrumbPage className="truncate font-medium">
                {page?.title || group.title}
              </BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <CommandMenu />
        <div className="hidden lg:block">
          <LanguageSwitch />
        </div>
        <div className="hidden xl:block">
          <TimezoneSwitch />
        </div>
        <ThemeSwitch />
        <UserNav />
      </div>
    </header>
  );
}
