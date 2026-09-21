import type { ErrorComponentProps } from "@tanstack/react-router";
import { useRouter } from "@tanstack/react-router";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { RefreshCw, TriangleAlert } from "lucide-react";
import { useState } from "react";
import { classifyRouteError, routeErrorDescriptions } from "@/lib/route-error";

export function AdminRouteError({ error, reset }: ErrorComponentProps) {
  const router = useRouter();
  const [retrying, setRetrying] = useState(false);
  const [retryFailed, setRetryFailed] = useState(false);
  const kind = classifyRouteError(error);

  const retry = async () => {
    if (retrying) return;
    setRetrying(true);
    setRetryFailed(false);
    try {
      await router.invalidate();
      reset();
    } catch {
      setRetryFailed(true);
    } finally {
      setRetrying(false);
    }
  };

  return (
    <div className="grid min-h-[45vh] place-items-center p-6">
      <Alert className="max-w-lg" variant="destructive">
        <TriangleAlert />
        <AlertTitle>页面暂时无法加载</AlertTitle>
        <AlertDescription className="flex flex-col gap-4">
          <p>{routeErrorDescriptions[kind]}</p>
          <p>尚未保存的内容不会自动提交；刷新页面会丢失未保存的修改。</p>
          {retryFailed ? (
            <p role="status">重试未成功，请稍后再试或刷新页面。</p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            {kind !== "resource" ? (
              <Button
                disabled={retrying}
                onClick={retry}
                size="sm"
                variant="outline"
              >
                <RefreshCw className={retrying ? "animate-spin" : undefined} />
                重试
              </Button>
            ) : null}
            <Button
              disabled={retrying}
              onClick={() => window.location.reload()}
              size="sm"
              variant="outline"
            >
              <RefreshCw />
              刷新页面
            </Button>
          </div>
        </AlertDescription>
      </Alert>
    </div>
  );
}
