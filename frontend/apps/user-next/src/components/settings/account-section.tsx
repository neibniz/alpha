"use client";

import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Avatar, AvatarFallback } from "@workspace/ui/components/avatar";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Card, CardContent } from "@workspace/ui/components/card";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Check, Copy, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { getCurrentUser } from "@/lib/client-api";
import { copyText } from "@/lib/copy-text";
import { formatDate } from "@/lib/format";
import { QueryFailure } from "./settings-common";

export function AccountSection() {
  const user = useQuery({
    queryFn: getCurrentUser,
    queryKey: ["alpha-current-user"],
  });

  if (user.isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }
  if (user.isError || !user.data) {
    return (
      <QueryFailure message="账户资料暂不可用" retry={() => user.refetch()} />
    );
  }

  const visibleAuthMethods = user.data.authMethods.filter(
    (method) => method.type === "email" || method.type === "github"
  );
  const identifier =
    visibleAuthMethods.find((method) => method.type === "email")?.identifier ||
    visibleAuthMethods[0]?.identifier ||
    `用户 #${user.data.id}`;
  const initial = identifier.trim().charAt(0).toUpperCase() || "A";

  return (
    <div className="grid gap-4">
      <Card className="rounded-lg py-0 shadow-none" data-reveal>
        <CardContent className="px-5 py-6 sm:px-6">
          <div className="flex items-center gap-4 border-b pb-6">
            <Avatar className="size-16">
              <AvatarFallback className="text-xl">{initial}</AvatarFallback>
            </Avatar>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <strong className="truncate font-medium text-xl">
                  {identifier}
                </strong>
                <Badge className="gap-1" variant="secondary">
                  <Check />
                  已验证
                </Badge>
              </div>
              <p className="mt-1 text-muted-foreground text-sm">
                用户 #{user.data.id} · 创建于 {formatDate(user.data.createdAt)}
              </p>
            </div>
          </div>
          <dl className="divide-y text-sm">
            {visibleAuthMethods.map((method) => (
              <div
                className="grid gap-1 py-4 sm:grid-cols-[160px_1fr]"
                key={method.type}
              >
                <dt className="text-muted-foreground">
                  {method.type === "email" ? "电子邮箱" : "GitHub"}
                </dt>
                <dd className="font-medium">{method.identifier || "已绑定"}</dd>
              </div>
            ))}
            <div className="grid gap-1 py-4 sm:grid-cols-[160px_1fr]">
              <dt className="text-muted-foreground">账户状态</dt>
              <dd className="flex items-center gap-2 font-medium">
                <i className="size-2 rounded-full bg-green-600" />
                正常
              </dd>
            </div>
            <div className="grid gap-1 py-4 sm:grid-cols-[160px_1fr_auto] sm:items-center">
              <dt className="text-muted-foreground">推荐码</dt>
              <dd className="font-medium">{user.data.referCode || "暂无"}</dd>
              <Button
                aria-label="复制推荐码"
                disabled={!user.data.referCode}
                onClick={async () => {
                  await copyText(user.data.referCode);
                  toast.success("推荐码已复制");
                }}
                size="icon"
                variant="outline"
              >
                <Copy />
              </Button>
            </div>
          </dl>
          <Alert>
            <ShieldCheck />
            <AlertTitle>隐私数据已脱敏</AlertTitle>
            <AlertDescription>敏感标识由服务端脱敏后返回。</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  );
}
