"use client";

import type React from "react";

interface IpLinkProps {
  ip: string;
  children?: React.ReactNode;
  className?: string;
}

interface IpLocationProps {
  className?: string;
  location?: string;
}

export function IpLink({ ip, children, className = "" }: IpLinkProps) {
  const cleanIP = ip.trim();

  if (!cleanIP) {
    return <span className="text-muted-foreground">-</span>;
  }

  return (
    <span className={`font-mono text-foreground ${className}`}>
      {children || cleanIP}
    </span>
  );
}

export function IpLocation({ className = "", location }: IpLocationProps) {
  const value = location?.trim();
  return (
    <span
      className={`inline-block max-w-44 truncate text-muted-foreground ${className}`}
      title={value || undefined}
    >
      {value || "-"}
    </span>
  );
}
