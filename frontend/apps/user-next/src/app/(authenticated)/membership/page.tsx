import type { Metadata } from "next";
import { MembershipCenter } from "@/components/membership/membership-center";
import { PageReveal } from "@/components/page-reveal";
import { getServerMembershipCard, getServerSiteConfig } from "@/lib/server/api";
import { ServerHydration } from "@/lib/server/hydration";

export const metadata: Metadata = { title: "会员中心" };

export default function MembershipPage() {
  return (
    <ServerHydration
      queries={[
        {
          queryFn: getServerMembershipCard,
          queryKey: ["alpha-membership-card"],
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
        <MembershipCenter />
      </PageReveal>
    </ServerHydration>
  );
}
