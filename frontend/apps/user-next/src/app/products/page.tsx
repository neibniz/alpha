import type { Metadata } from "next";
import { cookies } from "next/headers";
import { Suspense } from "react";
import { AppShell } from "@/components/app-shell";
import { PageReveal } from "@/components/page-reveal";
import { ProductStorefront } from "@/components/products/product-storefront";
import { WorkspaceLoading } from "@/components/workspace-loading";
import { authCookieName, isServerDemoMode } from "@/lib/runtime";
import { getServerProducts } from "@/lib/server/api";
import { ServerHydration } from "@/lib/server/hydration";

export const metadata: Metadata = { title: "购买套餐" };

export default async function ProductsPage() {
  const cookieStore = await cookies();
  const authenticated =
    isServerDemoMode() || Boolean(cookieStore.get(authCookieName())?.value);
  return (
    <AppShell authenticated={authenticated}>
      <Suspense fallback={<WorkspaceLoading />}>
        <ServerHydration
          queries={[
            {
              queryFn: () => getServerProducts(authenticated),
              queryKey: ["alpha-products", authenticated ? "user" : "guest"],
              staleTime: authenticated ? 15_000 : 60_000,
            },
          ]}
        >
          <PageReveal className="min-w-0">
            <ProductStorefront authenticated={authenticated} />
          </PageReveal>
        </ServerHydration>
      </Suspense>
    </AppShell>
  );
}
