import { Link, useLocation } from "@tanstack/react-router";
import { Button } from "@workspace/ui/components/button";
import { Separator } from "@workspace/ui/components/separator";
import { SidebarTrigger } from "@workspace/ui/components/sidebar";
import { LanguageSwitch } from "@workspace/ui/composed/language-switch";
import { ThemeSwitch } from "@workspace/ui/composed/theme-switch";
import { cn } from "@workspace/ui/lib/utils";
import { useGlobalStore } from "@/stores/global";
import { CommandMenu } from "./command-menu";
import { findActiveNavGroup, useNavs } from "./navs";
import TimezoneSwitch from "./timezone-switch";
import { UserNav } from "./user-nav";

export function Header() {
  const pathname = useLocation({ select: (location) => location.pathname });
  const navs = useNavs();
  const activeDomain = findActiveNavGroup(navs, pathname);
  const { common } = useGlobalStore();
  const site = common.site;

  return (
    <header className="admin-topbar">
      <div className="flex min-w-0 items-center gap-2">
        <SidebarTrigger className="shrink-0" />
        <Link className="admin-brand" to="/dashboard">
          <img
            alt="logo"
            height={28}
            src={site.site_logo || `${import.meta.env.BASE_URL}favicon.png`}
            width={28}
          />
          <span className="hidden truncate font-semibold sm:block">
            {site.site_name || "Alpha Panel"}
          </span>
        </Link>
        <Separator
          className="mx-1 hidden h-5 md:block"
          orientation="vertical"
        />
        <nav aria-label="Primary" className="admin-domain-nav">
          {navs.map((group) => {
            const Icon = group.icon;
            const active = group.title === activeDomain.title;
            return (
              <Button
                asChild
                className={cn("admin-domain-link", active && "is-active")}
                key={group.title}
                size="sm"
                variant="ghost"
              >
                <Link to={group.items[0]?.url || "/dashboard"}>
                  <Icon data-icon="inline-start" />
                  <span>{group.title}</span>
                </Link>
              </Button>
            );
          })}
        </nav>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <CommandMenu />
        <div className="hidden xl:block">
          <LanguageSwitch />
        </div>
        <div className="hidden 2xl:block">
          <TimezoneSwitch />
        </div>
        <ThemeSwitch />
        <UserNav />
      </div>
    </header>
  );
}
