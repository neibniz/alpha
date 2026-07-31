"use client";

import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Spinner } from "@workspace/ui/components/spinner";
import { CircleAlert } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { AlphaLogo } from "@/components/alpha-logo";
import { isClientDemoMode } from "@/lib/runtime";

export function OAuthCallback({
  code,
  state,
}: Readonly<{ code?: string; state?: string }>) {
  const router = useRouter();
  const [error, setError] = useState("");

  useEffect(() => {
    if (isClientDemoMode()) {
      router.replace("/dashboard");
      return;
    }
    if (!(code && state)) {
      setError("GitHub 回调参数不完整");
      return;
    }
    let active = true;
    fetch("/api/auth/oauth/callback", {
      body: JSON.stringify({ code, state }),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    })
      .then(async (response) => {
        const payload = (await response.json()) as { message?: string };
        if (!response.ok) throw new Error(payload.message || "GitHub 登录失败");
        if (active) {
          router.replace("/dashboard");
          router.refresh();
        }
      })
      .catch((reason: unknown) => {
        if (active)
          setError(
            reason instanceof Error ? reason.message : "GitHub 登录失败"
          );
      });
    return () => {
      active = false;
    };
  }, [code, router, state]);

  return (
    <main className="grid min-h-screen place-items-center bg-background px-4 py-10">
      <div className="w-full max-w-md">
        <div className="mb-6">
          <AlphaLogo />
        </div>
        <Card className="rounded-lg py-0 shadow-sm">
          <CardHeader className="items-center border-b px-6 py-6 text-center">
            <span
              className={`flex size-12 items-center justify-center rounded-full ${error ? "bg-destructive/10 text-destructive" : "bg-muted"}`}
            >
              {error ? <CircleAlert /> : <Spinner />}
            </span>
            <CardTitle className="mt-2 text-lg">
              {error ? "登录未完成" : "正在完成 GitHub 登录"}
            </CardTitle>
            <CardDescription>
              {error || "验证成功后将自动进入订阅工作台。"}
            </CardDescription>
          </CardHeader>
          {error ? (
            <CardContent className="px-6 py-5">
              <Button
                className="w-full"
                onClick={() => router.replace("/login")}
              >
                返回登录
              </Button>
            </CardContent>
          ) : null}
        </Card>
      </div>
    </main>
  );
}
