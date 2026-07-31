import type { ErrorComponentProps } from "@tanstack/react-router";
import { useRouter } from "@tanstack/react-router";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { RefreshCw, TriangleAlert } from "lucide-react";

export function AdminRouteError({ reset }: ErrorComponentProps) {
  const router = useRouter();

  const retry = () => {
    reset();
    return router.invalidate();
  };

  return (
    <div className="grid min-h-[45vh] place-items-center p-6">
      <Alert className="max-w-lg" variant="destructive">
        <TriangleAlert />
        <AlertTitle>页面暂时无法加载</AlertTitle>
        <AlertDescription className="space-y-4">
          <p>请检查网络连接后重试。尚未保存的表单内容不会被提交。</p>
          <Button onClick={retry} size="sm" variant="outline">
            <RefreshCw />
            重试
          </Button>
        </AlertDescription>
      </Alert>
    </div>
  );
}
