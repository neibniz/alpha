import type { NextConfig } from "next";
import { buildSubscriptionProxyRewrites } from "./src/lib/subscription-proxy";

const isDevelopment = process.env.NODE_ENV === "development";

const securityHeaders = [
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "X-DNS-Prefetch-Control", value: "off" },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=()",
  },
  {
    key: "Content-Security-Policy",
    value: [
      "default-src 'self'",
      `script-src 'self' 'unsafe-inline'${isDevelopment ? " 'unsafe-eval'" : ""}`,
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob: https:",
      `connect-src 'self' https:${isDevelopment ? " ws:" : ""}`,
      "font-src 'self' data:",
      "frame-src https:",
      "worker-src 'self' blob:",
      "object-src 'none'",
      "base-uri 'self'",
      "frame-ancestors 'none'",
      "form-action 'self' https:",
    ].join("; "),
  },
];

const nextConfig: NextConfig = {
  allowedDevOrigins: isDevelopment ? ["192.168.31.192"] : undefined,
  output: "standalone",
  poweredByHeader: false,
  reactStrictMode: true,
  transpilePackages: ["@workspace/ui"],
  async rewrites() {
    return buildSubscriptionProxyRewrites(
      process.env.ALPHA_SUBSCRIPTION_PROXY_PATH
    );
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
