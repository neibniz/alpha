"use client";

import { useQuery } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@workspace/ui/components/avatar";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { Progress } from "@workspace/ui/components/progress";
import { Separator } from "@workspace/ui/components/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import {
  Bell,
  ChevronDown,
  CircleHelp,
  FileText,
  Layers3,
  LogOut,
  Menu,
  MoreVertical,
  PanelLeftClose,
  PanelLeftOpen,
  ReceiptText,
  Settings,
  ShoppingBag,
  UserRound,
  WalletCards,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { type ReactNode, useState } from "react";
import { toast } from "sonner";
import { getCurrentUser, getSubscriptions } from "@/lib/client-api";
import { formatBytes, percentOf } from "@/lib/format";
import type { Subscription } from "@/lib/models";
import { AlphaLogo } from "./alpha-logo";

const primaryNavigation = [
  { href: "/dashboard", label: "我的订阅", icon: Layers3 },
  { href: "/products", label: "购买套餐", icon: ShoppingBag },
];

const secondaryNavigation = [
  { href: "/settings/wallet", label: "钱包", icon: WalletCards },
  { href: "/settings/orders", label: "订单", icon: ReceiptText },
  { href: "/settings/support", label: "工单", icon: CircleHelp },
  { href: "/settings/resources", label: "文档与公告", icon: FileText },
  { href: "/settings/account", label: "设置", icon: Settings },
];

const routeLabels: Record<string, string> = {
  "/dashboard": "我的订阅",
  "/products": "购买套餐",
  "/settings/account": "账户资料",
  "/settings/orders": "订单",
  "/settings/resources": "文档与公告",
  "/settings/security": "安全与登录",
  "/settings/support": "工单",
  "/settings/wallet": "钱包",
};

function isActivePath(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(`${href}/`);
}

function SubscriptionLink({
  active,
  compact,
  onNavigate,
  subscription,
}: Readonly<{
  active: boolean;
  compact: boolean;
  onNavigate?: () => void;
  subscription: Subscription;
}>) {
  const used = subscription.uploadBytes + subscription.downloadBytes;
  const usage = percentOf(used, subscription.trafficBytes);
  const label = (
    <Link
      aria-current={active ? "page" : undefined}
      className={`group block rounded-md px-3 py-2.5 transition-colors ${
        active
          ? "bg-sidebar-accent text-foreground"
          : "hover:bg-sidebar-accent/70"
      }`}
      href={`/dashboard?subscription=${subscription.id}`}
      onClick={onNavigate}
    >
      {compact ? (
        <span className="flex h-7 items-center justify-center font-semibold text-xs">
          {subscription.name.slice(0, 2)}
        </span>
      ) : (
        <span className="grid gap-2">
          <span className="flex items-center justify-between gap-3 text-sm">
            <strong className="truncate font-medium">
              {subscription.name}
            </strong>
            <span className="flex shrink-0 items-center gap-1.5 text-muted-foreground text-xs">
              <i
                aria-hidden="true"
                className={`size-1.5 rounded-full ${
                  subscription.status === "active"
                    ? "bg-blue-600"
                    : "bg-zinc-400"
                }`}
              />
              {subscription.status === "active" ? "正常" : "受限"}
            </span>
          </span>
          <span className="text-muted-foreground text-xs tabular-nums">
            已用 {formatBytes(used)} / {formatBytes(subscription.trafficBytes)}
          </span>
          <Progress className="h-1" value={usage} />
        </span>
      )}
    </Link>
  );

  if (!compact) return label;
  return (
    <Tooltip>
      <TooltipTrigger asChild>{label}</TooltipTrigger>
      <TooltipContent side="right">{subscription.name}</TooltipContent>
    </Tooltip>
  );
}

function SidebarContent({
  authenticated,
  compact,
  identifier,
  initial,
  logout,
  onNavigate,
  pathname,
  selectedSubscriptionId,
  subscriptions,
}: Readonly<{
  authenticated: boolean;
  compact: boolean;
  identifier: string;
  initial: string;
  logout: () => void;
  onNavigate?: () => void;
  pathname: string;
  selectedSubscriptionId: number;
  subscriptions: Subscription[];
}>) {
  const navigationLink = (
    item: (typeof primaryNavigation)[number],
    visible: boolean
  ) => {
    if (!visible) return null;
    const active = isActivePath(pathname, item.href);
    return (
      <Button
        asChild
        className={`w-full ${compact ? "justify-center px-0" : "justify-start"}`}
        key={item.href}
        variant={active ? "secondary" : "ghost"}
      >
        <Link href={item.href} onClick={onNavigate}>
          <item.icon aria-hidden="true" />
          {compact ? <span className="sr-only">{item.label}</span> : item.label}
        </Link>
      </Button>
    );
  };

  return (
    <div className="flex h-full min-h-0 flex-col bg-sidebar text-sidebar-foreground">
      <div
        className={`flex h-19 items-center ${compact ? "justify-center px-2" : "px-6"}`}
      >
        <AlphaLogo compact={compact} />
      </div>
      <div className={`px-4 pb-3 ${compact ? "px-2" : ""}`}>
        <Button asChild className="w-full" size="lg">
          <Link href="/products" onClick={onNavigate}>
            <ShoppingBag aria-hidden="true" />
            {compact ? (
              <span className="sr-only">购买新套餐</span>
            ) : (
              "购买新套餐"
            )}
          </Link>
        </Button>
      </div>
      <nav
        aria-label="主要导航"
        className={`grid gap-1 px-4 py-3 ${compact ? "px-2" : ""}`}
      >
        {primaryNavigation.map((item) =>
          navigationLink(item, authenticated || item.href === "/products")
        )}
      </nav>

      {authenticated ? (
        <>
          <Separator />
          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
            {compact ? null : (
              <p className="mb-2 px-3 font-medium text-muted-foreground text-xs">
                我的订阅
              </p>
            )}
            <div className="grid gap-1">
              {subscriptions.map((subscription, index) => (
                <SubscriptionLink
                  active={
                    pathname === "/dashboard" &&
                    (selectedSubscriptionId
                      ? subscription.id === selectedSubscriptionId
                      : index === 0)
                  }
                  compact={compact}
                  key={subscription.id}
                  onNavigate={onNavigate}
                  subscription={subscription}
                />
              ))}
            </div>
          </div>
          <Separator />
          <nav
            aria-label="账户导航"
            className={`grid gap-1 px-4 py-3 ${compact ? "px-2" : ""}`}
          >
            {secondaryNavigation.map((item) => {
              const active = isActivePath(pathname, item.href);
              return (
                <Button
                  asChild
                  className={`w-full ${compact ? "justify-center px-0" : "justify-start"}`}
                  key={item.href}
                  variant={active ? "secondary" : "ghost"}
                >
                  <Link href={item.href} onClick={onNavigate}>
                    <item.icon aria-hidden="true" />
                    {compact ? (
                      <span className="sr-only">{item.label}</span>
                    ) : (
                      item.label
                    )}
                  </Link>
                </Button>
              );
            })}
          </nav>
          <Separator />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                className={`m-3 h-auto ${compact ? "w-[calc(100%-1.5rem)] justify-center px-0 py-2" : "w-[calc(100%-1.5rem)] justify-start px-2 py-2"}`}
                variant="ghost"
              >
                <Avatar className="size-9">
                  <AvatarFallback>{initial}</AvatarFallback>
                </Avatar>
                {compact ? null : (
                  <>
                    <span className="min-w-0 flex-1 truncate text-left text-sm">
                      {identifier}
                    </span>
                    <MoreVertical aria-hidden="true" />
                  </>
                )}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-60">
              <DropdownMenuLabel>账户中心</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild>
                <Link href="/settings/account" onClick={onNavigate}>
                  <UserRound />
                  账户资料
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={logout}>
                <LogOut />
                退出登录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </>
      ) : (
        <div className="mt-auto p-4">
          <Button asChild className="w-full" variant="outline">
            <Link href="/login?next=%2Fproducts">登录</Link>
          </Button>
        </div>
      )}
    </div>
  );
}

export function AppShell({
  authenticated = true,
  children,
}: Readonly<{ authenticated?: boolean; children: ReactNode }>) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const user = useQuery({
    enabled: authenticated,
    queryFn: getCurrentUser,
    queryKey: ["alpha-current-user"],
  });
  const subscriptions = useQuery({
    enabled: authenticated,
    queryFn: getSubscriptions,
    queryKey: ["alpha-subscriptions"],
    staleTime: 30_000,
  });
  const identifier =
    user.data?.authMethods.find((method) => method.type === "email")
      ?.identifier ||
    user.data?.authMethods[0]?.identifier ||
    (user.data ? `用户 #${user.data.id}` : "账户");
  const initial = identifier.trim().charAt(0).toUpperCase() || "A";
  const pageLabel = routeLabels[pathname] || "Alpha";
  const selectedSubscriptionId = Number(searchParams.get("subscription")) || 0;

  const logout = async () => {
    try {
      await fetch("/api/auth/logout", { method: "POST" });
      router.push("/login");
      router.refresh();
    } catch {
      toast.error("退出失败，请稍后重试");
    }
  };

  const sidebarProps = {
    authenticated,
    identifier,
    initial,
    logout,
    pathname,
    selectedSubscriptionId,
    subscriptions: subscriptions.data || [],
  };

  return (
    <div className="min-h-screen bg-background">
      <aside
        className={`fixed inset-y-0 left-0 z-40 hidden border-r bg-sidebar transition-[width] duration-200 lg:block ${
          collapsed ? "w-20" : "w-[292px]"
        }`}
      >
        <SidebarContent compact={collapsed} {...sidebarProps} />
      </aside>

      <Sheet onOpenChange={setMobileOpen} open={mobileOpen}>
        <SheetContent
          className="w-[292px] gap-0 p-0 sm:max-w-[292px]"
          side="left"
        >
          <SheetHeader className="sr-only">
            <SheetTitle>主导航</SheetTitle>
            <SheetDescription>浏览订阅、套餐和账户设置</SheetDescription>
          </SheetHeader>
          <SidebarContent
            compact={false}
            onNavigate={() => setMobileOpen(false)}
            {...sidebarProps}
          />
        </SheetContent>
      </Sheet>

      <div
        className={`min-h-screen transition-[padding] duration-200 ${
          collapsed ? "lg:pl-20" : "lg:pl-[292px]"
        }`}
      >
        <header className="sticky top-0 z-30 flex h-19 items-center justify-between border-b bg-background/96 px-4 backdrop-blur-sm sm:px-6 lg:px-7">
          <div className="flex min-w-0 items-center gap-4">
            <Button
              aria-label="打开主导航"
              className="lg:hidden"
              onClick={() => setMobileOpen(true)}
              size="icon"
              variant="outline"
            >
              <Menu aria-hidden="true" />
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  aria-label={collapsed ? "展开侧栏" : "收起侧栏"}
                  className="hidden lg:inline-flex"
                  onClick={() => setCollapsed((value) => !value)}
                  size="icon"
                  variant="outline"
                >
                  {collapsed ? <PanelLeftOpen /> : <PanelLeftClose />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {collapsed ? "展开侧栏" : "收起侧栏"}
              </TooltipContent>
            </Tooltip>
            <div className="flex min-w-0 items-center gap-3 text-sm">
              <span className="hidden text-muted-foreground sm:inline">
                Alpha
              </span>
              <span className="hidden text-muted-foreground sm:inline">/</span>
              <strong className="truncate font-medium">{pageLabel}</strong>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {authenticated ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    asChild
                    aria-label="查看通知"
                    size="icon"
                    variant="outline"
                  >
                    <Link href="/settings/resources">
                      <Bell aria-hidden="true" />
                      <i
                        aria-hidden="true"
                        className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-blue-600"
                      />
                    </Link>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>通知与公告</TooltipContent>
              </Tooltip>
            ) : (
              <Button asChild size="sm">
                <Link href="/login?next=%2Fproducts">登录</Link>
              </Button>
            )}
            {authenticated ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    className="hidden gap-2 px-2 sm:inline-flex"
                    variant="ghost"
                  >
                    <Avatar className="size-7">
                      <AvatarFallback>{initial}</AvatarFallback>
                    </Avatar>
                    <span className="max-w-28 truncate">{identifier}</span>
                    <ChevronDown aria-hidden="true" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuItem asChild>
                    <Link href="/settings/account">账户资料</Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={logout}>
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}
          </div>
        </header>
        <main className="min-h-[calc(100vh-4.75rem)]">{children}</main>
      </div>
    </div>
  );
}
