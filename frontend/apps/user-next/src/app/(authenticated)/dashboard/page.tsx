import type { Metadata } from "next";
import { PageReveal } from "@/components/page-reveal";
import { SubscriptionWorkspace } from "@/components/dashboard/subscription-workspace";
import {
  getServerSiteConfig,
  getServerSubscriptionScopes,
} from "@/lib/server/api";
import { ServerHydration } from "@/lib/server/hydration";

export const metadata: Metadata = { title: "我的订阅" };

export default function DashboardPage() {
  return (
    <ServerHydration
      queries={[
        {
          queryFn: getServerSubscriptionScopes,
          queryKey: ["alpha-subscription-scopes"],
          staleTime: 30_000,
        },
        {
          queryFn: getServerSiteConfig,
          queryKey: ["alpha-site-config"],
          staleTime: 300_000,
        },
      ]}
    >
      <PageReveal className="min-w-0">
        <SubscriptionWorkspace />
      </PageReveal>
    </ServerHydration>
  );
}
