import Image from "next/image";
import Link from "next/link";

export function AlphaLogo({
  compact = false,
}: Readonly<{ compact?: boolean }>) {
  return (
    <Link
      aria-label="Alpha 首页"
      className="inline-flex items-center gap-3 font-semibold text-foreground text-xl leading-none"
      href="/dashboard"
    >
      <span className="grid size-11 shrink-0 place-items-center rounded-lg border bg-background">
        <Image alt="" height={26} priority src="/alpha-mark.png" width={26} />
      </span>
      {compact ? null : <span>Alpha</span>}
    </Link>
  );
}
