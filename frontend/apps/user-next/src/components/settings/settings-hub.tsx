"use client";

import { Button } from "@workspace/ui/components/button";
import { Skeleton } from "@workspace/ui/components/skeleton";
import {
  BookOpen,
  CircleHelp,
  ReceiptText,
  ShieldCheck,
  UserRound,
  WalletCards,
} from "lucide-react";
import Link from "next/link";
import dynamic from "next/dynamic";
import type { SettingsSection } from "@/lib/settings";

function SectionLoading() {
  return (
    <div aria-busy="true" className="grid gap-3">
      <Skeleton className="h-36 rounded-lg" />
      <Skeleton className="h-64 rounded-lg" />
    </div>
  );
}

const AccountSection = dynamic(
  () => import("./account-section").then((module) => module.AccountSection),
  { loading: SectionLoading }
);
const OrdersSection = dynamic(
  () => import("./orders-section").then((module) => module.OrdersSection),
  { loading: SectionLoading }
);
const ResourcesSection = dynamic(
  () => import("./resources-section").then((module) => module.ResourcesSection),
  { loading: SectionLoading }
);
const SecuritySection = dynamic(
  () => import("./security-section").then((module) => module.SecuritySection),
  { loading: SectionLoading }
);
const SupportSection = dynamic(
  () => import("./support-section").then((module) => module.SupportSection),
  { loading: SectionLoading }
);
const WalletSection = dynamic(
  () => import("./wallet-section").then((module) => module.WalletSection),
  { loading: SectionLoading }
);

const navigation: Array<{
  href: SettingsSection;
  icon: typeof UserRound;
  label: string;
}> = [
  { href: "account", icon: UserRound, label: "账户资料" },
  { href: "security", icon: ShieldCheck, label: "安全与登录" },
  { href: "wallet", icon: WalletCards, label: "钱包" },
  { href: "orders", icon: ReceiptText, label: "订单" },
  { href: "support", icon: CircleHelp, label: "工单" },
  { href: "resources", icon: BookOpen, label: "文档与公告" },
];

const sectionTitles: Record<
  SettingsSection,
  { description: string; title: string }
> = {
  account: { description: "查看当前账户的登录标识。", title: "账户资料" },
  security: {
    description: "管理密码与 GitHub 登录。",
    title: "安全与登录",
  },
  wallet: { description: "查看资产明细并完成账户充值。", title: "钱包" },
  orders: { description: "查询购买、续费、重置与充值记录。", title: "订单" },
  support: { description: "提交问题并跟踪处理进度。", title: "工单" },
  resources: { description: "产品文档、公告和邀请信息。", title: "文档与公告" },
};

function SectionContent({ section }: Readonly<{ section: SettingsSection }>) {
  if (section === "account") return <AccountSection />;
  if (section === "security") return <SecuritySection />;
  if (section === "wallet") return <WalletSection />;
  if (section === "orders") return <OrdersSection />;
  if (section === "support") return <SupportSection />;
  return <ResourcesSection />;
}

export function SettingsHub({
  section,
}: Readonly<{ section: SettingsSection }>) {
  return (
    <div className="mx-auto flex w-full max-w-[1500px] flex-col gap-6 px-4 py-8 sm:px-6 lg:px-7 lg:py-9">
      <header data-reveal>
        <h1 className="font-semibold text-3xl tracking-tight">账户设置</h1>
        <p className="mt-2 text-muted-foreground text-sm">
          管理账户、资产与支持服务。
        </p>
      </header>
      <nav
        aria-label="设置导航"
        className="flex gap-1 overflow-x-auto border-b pb-0"
        data-reveal
      >
        {navigation.map((item) => (
          <Button
            asChild
            className={`shrink-0 rounded-b-none border-b-2 px-4 ${
              item.href === section
                ? "border-blue-600 bg-transparent text-foreground hover:bg-muted/50"
                : "border-transparent text-muted-foreground"
            }`}
            key={item.href}
            variant="ghost"
          >
            <Link href={`/settings/${item.href}`}>
              <item.icon />
              <span>{item.label}</span>
            </Link>
          </Button>
        ))}
      </nav>
      <section className="min-w-0">
        <header className="mb-5">
          <h2 className="font-semibold text-xl">
            {sectionTitles[section].title}
          </h2>
          <p className="mt-1 text-muted-foreground text-sm">
            {sectionTitles[section].description}
          </p>
        </header>
        <SectionContent section={section} />
      </section>
    </div>
  );
}
