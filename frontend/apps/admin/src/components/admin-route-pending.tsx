import { Skeleton } from "@workspace/ui/components/skeleton";

export function AdminRoutePending() {
  return (
    <div
      aria-busy="true"
      aria-label="正在加载页面"
      className="mx-auto flex min-h-[45vh] w-full max-w-6xl flex-col gap-5 p-6"
      role="status"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-2">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-64 max-w-[70vw]" />
        </div>
        <Skeleton className="size-9" />
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        <Skeleton className="h-28" />
        <Skeleton className="h-28" />
        <Skeleton className="h-28" />
      </div>
      <Skeleton className="h-64 w-full" />
    </div>
  );
}
