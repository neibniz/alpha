import { Skeleton } from "@workspace/ui/components/skeleton";

export function WorkspaceLoading() {
  return (
    <main
      aria-busy="true"
      aria-label="正在加载页面"
      className="mx-auto flex w-full max-w-[1500px] flex-col gap-6 px-4 py-6 sm:px-6 lg:px-7 lg:py-8"
    >
      <div className="space-y-3">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-full max-w-lg" />
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        <Skeleton className="h-44 rounded-lg lg:col-span-2" />
        <Skeleton className="h-44 rounded-lg" />
      </div>
      <Skeleton className="h-96 rounded-lg" />
    </main>
  );
}
