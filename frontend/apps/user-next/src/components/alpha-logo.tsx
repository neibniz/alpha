"use client";

import { useQuery } from "@tanstack/react-query";
import Image, { type ImageLoaderProps } from "next/image";
import Link from "next/link";
import { useState } from "react";
import { getSiteConfig } from "@/lib/client-api";
import {
  DEFAULT_SITE_LOGO,
  DEFAULT_SITE_NAME,
  SITE_CONFIG_QUERY_KEY,
  SITE_CONFIG_STALE_TIME,
} from "@/lib/site-brand";

function siteLogoLoader({ src }: ImageLoaderProps) {
  return src;
}

export function AlphaLogo({
  compact = false,
}: Readonly<{ compact?: boolean }>) {
  const config = useQuery({
    queryFn: getSiteConfig,
    queryKey: SITE_CONFIG_QUERY_KEY,
    staleTime: SITE_CONFIG_STALE_TIME,
  });
  const [failedLogo, setFailedLogo] = useState("");
  const siteName = config.data?.siteName || DEFAULT_SITE_NAME;
  const configuredLogo = config.data?.siteLogo || "";
  const logoSource =
    configuredLogo && configuredLogo !== failedLogo
      ? configuredLogo
      : DEFAULT_SITE_LOGO;

  return (
    <Link
      aria-label={`${siteName} 首页`}
      className="inline-flex min-w-0 items-center gap-3 font-semibold text-foreground text-xl leading-none"
      href="/dashboard"
      title={siteName}
    >
      <span
        className={`grid h-11 shrink-0 place-items-center rounded-lg border bg-background px-2 ${
          compact ? "w-11" : "w-16"
        }`}
      >
        <Image
          alt=""
          className="max-h-7 max-w-full object-contain"
          height={28}
          loader={siteLogoLoader}
          onError={() => {
            if (configuredLogo && logoSource === configuredLogo) {
              setFailedLogo(configuredLogo);
            }
          }}
          referrerPolicy="no-referrer"
          src={logoSource}
          unoptimized
          width={64}
        />
      </span>
      {compact ? null : <span className="max-w-44 truncate">{siteName}</span>}
    </Link>
  );
}
