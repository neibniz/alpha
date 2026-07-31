import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PageReveal } from "@/components/page-reveal";
import { SettingsHub } from "@/components/settings/settings-hub";
import { settingsSections, type SettingsSection } from "@/lib/settings";
import {
  getServerAnnouncements,
  getServerDocuments,
  getServerInitialOrders,
  getServerOAuthMethods,
  getServerPaymentMethods,
  getServerSiteConfig,
  getServerTickets,
  getServerWalletLogs,
} from "@/lib/server/api";
import {
  ServerHydration,
  type ServerPrefetchQuery,
} from "@/lib/server/hydration";

type PageProps = { params: Promise<{ section: string }> };

export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { section } = await params;
  const labels: Record<string, string> = {
    account: "账户资料",
    orders: "订单",
    resources: "文档与公告",
    security: "安全与登录",
    support: "工单",
    wallet: "钱包",
  };
  return { title: labels[section] || "设置" };
}

export default async function SettingsSectionPage({ params }: PageProps) {
  const { section } = await params;
  if (!settingsSections.includes(section as SettingsSection)) notFound();
  const queries: ServerPrefetchQuery[] = [];
  if (section === "security") {
    queries.push(
      {
        queryFn: getServerOAuthMethods,
        queryKey: ["alpha-oauth-methods"],
      },
      {
        queryFn: getServerSiteConfig,
        queryKey: ["alpha-site-config"],
        staleTime: 300_000,
      }
    );
  } else if (section === "wallet") {
    queries.push(
      {
        queryFn: getServerWalletLogs,
        queryKey: ["alpha-wallet-logs"],
      },
      {
        queryFn: getServerPaymentMethods,
        queryKey: ["alpha-payment-methods"],
      }
    );
  } else if (section === "orders") {
    queries.push({
      queryFn: getServerInitialOrders,
      queryKey: ["alpha-orders", 1, 0, ""],
    });
  } else if (section === "support") {
    queries.push({
      queryFn: getServerTickets,
      queryKey: ["alpha-tickets"],
    });
  } else if (section === "resources") {
    queries.push(
      {
        queryFn: getServerDocuments,
        queryKey: ["alpha-documents"],
      },
      {
        queryFn: getServerAnnouncements,
        queryKey: ["alpha-announcements"],
      }
    );
  }
  return (
    <ServerHydration queries={queries}>
      <PageReveal className="min-w-0">
        <SettingsHub section={section as SettingsSection} />
      </PageReveal>
    </ServerHydration>
  );
}
