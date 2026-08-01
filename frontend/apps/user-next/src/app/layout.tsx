import type { Metadata } from "next";
import type { ReactNode } from "react";
import { getServerSiteConfig } from "@/lib/server/api";
import {
  DEFAULT_SITE_DESCRIPTION,
  DEFAULT_SITE_NAME,
} from "@/lib/site-brand";
import "@workspace/ui/globals.css";
import "./globals.css";
import { Providers } from "./providers";

export const dynamic = "force-dynamic";

export async function generateMetadata(): Promise<Metadata> {
  const config = await getServerSiteConfig().catch(() => undefined);
  const siteName = config?.siteName || DEFAULT_SITE_NAME;
  const description = config?.siteDescription || DEFAULT_SITE_DESCRIPTION;
  const siteLogo = config?.siteLogo || "";
  const keywords = (config?.keywords || "")
    .split(/[,，]/)
    .map((keyword) => keyword.trim())
    .filter(Boolean);
  const icons = siteLogo
    ? { apple: siteLogo, icon: siteLogo, shortcut: siteLogo }
    : {
        apple: [
          {
            sizes: "180x180",
            type: "image/png",
            url: "/apple-touch-icon.png",
          },
        ],
        icon: [
          { sizes: "256x256", type: "image/png", url: "/favicon.png" },
        ],
        shortcut: "/favicon.ico",
      };

  return {
    applicationName: siteName,
    description,
    icons,
    keywords: keywords.length ? keywords : undefined,
    manifest: "/manifest.webmanifest",
    title: {
      default: siteName,
      template: `%s | ${siteName}`,
    },
  };
}

export default function RootLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="zh-CN">
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
