import type { MetadataRoute } from "next";
import { getServerSiteConfig } from "@/lib/server/api";
import {
  DEFAULT_SITE_DESCRIPTION,
  DEFAULT_SITE_NAME,
} from "@/lib/site-brand";

export const dynamic = "force-dynamic";

export default async function manifest(): Promise<MetadataRoute.Manifest> {
  const config = await getServerSiteConfig().catch(() => undefined);
  const siteName = config?.siteName || DEFAULT_SITE_NAME;
  const siteLogo = config?.siteLogo || "";

  return {
    background_color: "#f7f8f6",
    description: config?.siteDescription || DEFAULT_SITE_DESCRIPTION,
    display: "standalone",
    icons: siteLogo
      ? [{ purpose: "any", sizes: "any", src: siteLogo }]
      : [
          {
            purpose: "any",
            sizes: "192x192",
            src: "/pwa-192x192.png",
            type: "image/png",
          },
          {
            purpose: "any",
            sizes: "512x512",
            src: "/pwa-512x512.png",
            type: "image/png",
          },
          {
            purpose: "maskable",
            sizes: "192x192",
            src: "/pwa-maskable-192x192.png",
            type: "image/png",
          },
          {
            purpose: "maskable",
            sizes: "512x512",
            src: "/pwa-maskable-512x512.png",
            type: "image/png",
          },
        ],
    name: siteName,
    short_name: siteName,
    start_url: "/",
    theme_color: "#f7f8f6",
  };
}
