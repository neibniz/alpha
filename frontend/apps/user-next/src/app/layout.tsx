import type { Metadata } from "next";
import type { ReactNode } from "react";
import "@workspace/ui/globals.css";
import "./globals.css";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: {
    default: "Alpha",
    template: "%s | Alpha",
  },
  description: "Alpha subscription workspace",
  icons: {
    icon: [{ sizes: "256x256", type: "image/png", url: "/favicon.png" }],
    shortcut: "/favicon.ico",
    apple: [
      {
        sizes: "180x180",
        type: "image/png",
        url: "/apple-touch-icon.png",
      },
    ],
  },
  manifest: "/site.webmanifest",
};

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
