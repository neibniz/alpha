import type { Metadata } from "next";
import { MembershipCenter } from "@/components/membership/membership-center";
import { PageReveal } from "@/components/page-reveal";

export const metadata: Metadata = { title: "会员中心" };

export default function MembershipPage() {
  return (
    <PageReveal className="min-w-0">
      <MembershipCenter />
    </PageReveal>
  );
}
