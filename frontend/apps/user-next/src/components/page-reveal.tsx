"use client";

import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import type { ReactNode } from "react";
import { useRef } from "react";

gsap.registerPlugin(useGSAP);

export function PageReveal({
  children,
  className = "",
}: Readonly<{ children: ReactNode; className?: string }>) {
  const root = useRef<HTMLDivElement>(null);

  useGSAP(() => {
    if (!root.current) return;
    const media = gsap.matchMedia();
    media.add("(prefers-reduced-motion: no-preference)", () => {
      gsap.fromTo(
        root.current,
        { autoAlpha: 0, y: 14 },
        {
          autoAlpha: 1,
          duration: 0.62,
          ease: "power3.out",
          y: 0,
        }
      );
    });
    return () => media.revert();
  });

  return (
    <div className={className} ref={root}>
      {children}
    </div>
  );
}
