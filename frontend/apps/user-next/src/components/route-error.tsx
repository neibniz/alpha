"use client";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { CircleAlert, RefreshCw } from "lucide-react";
import { useEffect } from "react";

export function RouteError({
  error,
  reset,
}: Readonly<{ error: Error & { digest?: string }; reset: () => void }>) {
  useEffect(() => {
    console.error("Route rendering failed", {
      digest: error.digest,
      message: error.message,
    });
  }, [error]);

  return (
    <main className="mx-auto flex min-h-[60vh] w-full max-w-3xl items-center px-4 py-12 sm:px-6">
      <Alert variant="destructive">
        <CircleAlert aria-hidden="true" />
        <AlertTitle>页面暂时无法加载</AlertTitle>
        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
          <span>服务可能正在更新，请稍后重试。</span>
          <Button onClick={reset} size="sm" type="button" variant="outline">
            <RefreshCw aria-hidden="true" />
            重新加载
          </Button>
        </AlertDescription>
      </Alert>
    </main>
  );
}
