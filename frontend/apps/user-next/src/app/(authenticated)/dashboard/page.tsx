import type { Metadata } from "next";
import { PageReveal } from "@/components/page-reveal";
import { SubscriptionWorkspace } from "@/components/dashboard/subscription-workspace";

export const metadata: Metadata = { title: "我的订阅" };

export default function DashboardPage() {
  return (
    <PageReveal className="min-w-0">
      <SubscriptionWorkspace />
    </PageReveal>
  );
}
