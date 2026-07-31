import {
  Activity,
  BadgeCheck,
  BellRing,
  BookOpenText,
  Boxes,
  ChartNoAxesCombined,
  CreditCard,
  FileCode2,
  Gift,
  HandCoins,
  KeyRound,
  LayoutDashboard,
  LifeBuoy,
  Link2,
  LogIn,
  Mail,
  Network,
  PackageOpen,
  PanelTop,
  ReceiptText,
  Route,
  RotateCcw,
  ServerCog,
  Settings,
  ShieldCheck,
  ShoppingBag,
  SlidersHorizontal,
  TicketPercent,
  UserPlus,
  UsersRound,
  WalletCards,
  type LucideIcon,
} from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

export interface NavLink {
  title: string;
  url: string;
  icon: LucideIcon;
  description?: string;
}

export interface NavGroup {
  title: string;
  icon: LucideIcon;
  items: NavLink[];
  description?: string;
}

export type NavItem = NavLink | NavGroup;

export function isNavGroup(item: NavItem): item is NavGroup {
  return "items" in item;
}

export function isNavLink(item: NavItem): item is NavLink {
  return "url" in item;
}

export function useNavs(): NavGroup[] {
  const { t } = useTranslation("menu");

  return useMemo(
    () => [
      {
        title: t("Overview", "Overview"),
        description: t("Overview Description", "Live operating status"),
        icon: LayoutDashboard,
        items: [
          {
            title: t("Dashboard", "Dashboard"),
            description: t(
              "Dashboard Description",
              "Health, traffic and business signals"
            ),
            url: "/dashboard",
            icon: LayoutDashboard,
          },
        ],
      },
      {
        title: t("Infrastructure", "Infrastructure"),
        description: t(
          "Infrastructure Description",
          "Compute, nodes and delivery configuration"
        ),
        icon: Boxes,
        items: [
          {
            title: t("Server Management", "Server Management"),
            url: "/dashboard/servers",
            icon: ServerCog,
          },
          {
            title: t("Node Management", "Node Management"),
            url: "/dashboard/nodes",
            icon: Network,
          },
          {
            title: t("Forwarder Processes", "Forwarder Processes"),
            url: "/dashboard/forwarders",
            icon: Route,
          },
          {
            title: t("Xray Templates", "Xray Templates"),
            url: "/dashboard/xray-templates",
            icon: FileCode2,
          },
          {
            title: t("Subscribe Config", "Subscribe Config"),
            url: "/dashboard/subscribe",
            icon: SlidersHorizontal,
          },
        ],
      },
      {
        title: t("Commerce", "Commerce"),
        description: t(
          "Commerce Description",
          "Catalog, orders and promotions"
        ),
        icon: ShoppingBag,
        items: [
          {
            title: t("Product Management", "Product Management"),
            url: "/dashboard/product",
            icon: PackageOpen,
          },
          {
            title: t("Order Management", "Order Management"),
            url: "/dashboard/order",
            icon: ReceiptText,
          },
          {
            title: t("Membership Management", "Membership Management"),
            url: "/dashboard/membership",
            icon: BadgeCheck,
          },
          {
            title: t("Coupon Management", "Coupon Management"),
            url: "/dashboard/coupon",
            icon: TicketPercent,
          },
        ],
      },
      {
        title: t("Customers", "Customers"),
        description: t(
          "Customers Description",
          "Users, service and communication"
        ),
        icon: UsersRound,
        items: [
          {
            title: t("User Management", "User Management"),
            url: "/dashboard/user",
            icon: UsersRound,
          },
          {
            title: t("Ticket Management", "Ticket Management"),
            url: "/dashboard/ticket",
            icon: LifeBuoy,
          },
          {
            title: t("Announcement Management", "Announcement Management"),
            url: "/dashboard/announcement",
            icon: BellRing,
          },
          {
            title: t("Document Management", "Document Management"),
            url: "/dashboard/document",
            icon: BookOpenText,
          },
        ],
      },
      {
        title: t("Analytics", "Analytics"),
        description: t(
          "Analytics Description",
          "Traffic, access and account activity"
        ),
        icon: ChartNoAxesCombined,
        items: [
          {
            title: t("Traffic Details", "Traffic Details"),
            url: "/dashboard/log/traffic-details",
            icon: Activity,
          },
          {
            title: t("Proxy Access", "Proxy Access"),
            url: "/dashboard/log/proxy-access",
            icon: ShieldCheck,
          },
          {
            title: t("Subscribe Traffic", "Subscribe Traffic"),
            url: "/dashboard/log/subscribe-traffic",
            icon: Link2,
          },
          {
            title: t("Server Traffic", "Server Traffic"),
            url: "/dashboard/log/server-traffic",
            icon: Network,
          },
          {
            title: t("Login", "Login"),
            url: "/dashboard/log/login",
            icon: LogIn,
          },
          {
            title: t("Register", "Register"),
            url: "/dashboard/log/register",
            icon: UserPlus,
          },
          {
            title: t("Email", "Email"),
            url: "/dashboard/log/email",
            icon: Mail,
          },
          {
            title: t("Subscribe", "Subscribe"),
            url: "/dashboard/log/subscribe",
            icon: Link2,
          },
          {
            title: t("Reset Subscribe", "Reset Subscribe"),
            url: "/dashboard/log/reset-subscribe",
            icon: RotateCcw,
          },
          {
            title: t("Balance", "Balance"),
            url: "/dashboard/log/balance",
            icon: WalletCards,
          },
          {
            title: t("Commission", "Commission"),
            url: "/dashboard/log/commission",
            icon: HandCoins,
          },
          {
            title: t("Gift", "Gift"),
            url: "/dashboard/log/gift",
            icon: Gift,
          },
        ],
      },
      {
        title: t("Settings", "Settings"),
        description: t(
          "Settings Description",
          "Platform, access and payment settings"
        ),
        icon: Settings,
        items: [
          {
            title: t("System Config", "System Config"),
            url: "/dashboard/system",
            icon: SlidersHorizontal,
          },
          {
            title: t("Auth Control", "Auth Control"),
            url: "/dashboard/auth-control",
            icon: KeyRound,
          },
          {
            title: t("Payment Config", "Payment Config"),
            url: "/dashboard/payment",
            icon: CreditCard,
          },
          {
            title: t("ADS Config", "ADS Config"),
            url: "/dashboard/ads",
            icon: PanelTop,
          },
        ],
      },
    ],
    [t]
  );
}

const normalizePath = (path: string) =>
  path.endsWith("/") && path !== "/" ? path.replace(/\/+$/, "") : path;

export function isActiveNavUrl(url: string, pathname: string) {
  const path = normalizePath(pathname);
  const target = normalizePath(url);
  if (target === "/dashboard") return path === target;
  return path === target || path.startsWith(`${target}/`);
}

export function findActiveNavGroup(navs: NavGroup[], pathname: string) {
  return (
    navs.find((group) =>
      group.items.some((item) => isActiveNavUrl(item.url, pathname))
    ) ?? navs[0]!
  );
}

export function findNavByUrl(navs: NavGroup[], pathname: string): NavItem[] {
  for (const group of navs) {
    const item = group.items.find((candidate) =>
      isActiveNavUrl(candidate.url, pathname)
    );
    if (item) return [group, item];
  }
  return [];
}
