"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Field, FieldLabel } from "@workspace/ui/components/field";
import { Input } from "@workspace/ui/components/input";
import { Spinner } from "@workspace/ui/components/spinner";
import { Check, Github, KeyRound } from "lucide-react";
import { type FormEvent, useState } from "react";
import { toast } from "sonner";
import {
  getOAuthMethods,
  getSiteConfig,
  startOAuthBind,
  unbindOAuth,
  updatePassword,
} from "@/lib/client-api";
import { githubOAuthCallbackUrl } from "@/lib/oauth-callback";
import { errorMessage } from "./settings-common";

export function SecuritySection() {
  const queryClient = useQueryClient();
  const methods = useQuery({
    queryFn: getOAuthMethods,
    queryKey: ["alpha-oauth-methods"],
  });
  const config = useQuery({
    queryFn: getSiteConfig,
    queryKey: ["alpha-site-config"],
    staleTime: 300_000,
  });
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordBusy, setPasswordBusy] = useState(false);
  const [oauthBusy, setOauthBusy] = useState(false);
  const github = methods.data?.find((method) => method.type === "github");
  const githubEnabled = config.data?.oauthMethods.includes("github") ?? false;

  const changePassword = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error("两次输入的新密码不一致");
      return;
    }
    if (newPassword.length < 12) {
      toast.error("新密码至少需要 12 位");
      return;
    }
    setPasswordBusy(true);
    try {
      await updatePassword({
        current_password: currentPassword,
        password: newPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success("密码已更新");
    } catch (error) {
      toast.error(errorMessage(error, "密码更新失败"));
    } finally {
      setPasswordBusy(false);
    }
  };

  const bindGitHub = async () => {
    setOauthBusy(true);
    try {
      const redirect = await startOAuthBind(
        "github",
        githubOAuthCallbackUrl(window.location.origin, "bind")
      );
      window.location.assign(redirect);
    } catch (error) {
      toast.error(errorMessage(error, "GitHub 绑定失败"));
      setOauthBusy(false);
    }
  };

  const disconnectGitHub = async () => {
    setOauthBusy(true);
    try {
      await unbindOAuth("github");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["alpha-oauth-methods"] }),
        queryClient.invalidateQueries({ queryKey: ["alpha-current-user"] }),
      ]);
      toast.success("GitHub 已解除绑定");
    } catch (error) {
      toast.error(errorMessage(error, "无法解除 GitHub 绑定"));
    } finally {
      setOauthBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <form data-reveal onSubmit={changePassword}>
        <Card className="rounded-lg py-0 shadow-xs">
          <CardHeader className="border-b px-5 py-5">
            <CardTitle className="text-base">修改密码</CardTitle>
            <CardDescription>使用 12 至 72 位的独立密码。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 px-5 py-5 sm:grid-cols-2">
            <Field className="sm:col-span-2">
              <FieldLabel htmlFor="settings-current-password">
                当前密码
              </FieldLabel>
              <Input
                autoComplete="current-password"
                id="settings-current-password"
                maxLength={128}
                onChange={(event) => setCurrentPassword(event.target.value)}
                required
                type="password"
                value={currentPassword}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="settings-new-password">新密码</FieldLabel>
              <Input
                autoComplete="new-password"
                id="settings-new-password"
                maxLength={72}
                minLength={12}
                onChange={(event) => setNewPassword(event.target.value)}
                required
                type="password"
                value={newPassword}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="settings-confirm-password">
                确认新密码
              </FieldLabel>
              <Input
                autoComplete="new-password"
                id="settings-confirm-password"
                maxLength={72}
                minLength={12}
                onChange={(event) => setConfirmPassword(event.target.value)}
                required
                type="password"
                value={confirmPassword}
              />
            </Field>
          </CardContent>
          <CardFooter className="justify-end border-t px-5 py-4">
            <Button disabled={passwordBusy} type="submit">
              {passwordBusy ? <Spinner /> : <KeyRound />}
              更新密码
            </Button>
          </CardFooter>
        </Card>
      </form>

      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">第三方登录</CardTitle>
          <CardDescription>仅支持管理员已启用的 GitHub OAuth。</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border bg-muted/30">
            <Github />
          </div>
          <div className="min-w-0 flex-1">
            <strong className="block font-medium text-sm">GitHub</strong>
            <small className="block truncate text-muted-foreground">
              {github?.identifier ||
                (githubEnabled ? "尚未绑定" : "管理员未启用")}
            </small>
          </div>
          <Badge variant={github ? "secondary" : "outline"}>
            {github ? <Check /> : null}
            {github ? "已连接" : "未连接"}
          </Badge>
          <Button
            disabled={
              oauthBusy ||
              !(github || githubEnabled) ||
              Boolean(github && (methods.data?.length || 0) <= 1)
            }
            onClick={github ? disconnectGitHub : bindGitHub}
            size="sm"
            variant="outline"
          >
            {oauthBusy ? <Spinner /> : <Github />}
            {github ? "解除绑定" : "连接 GitHub"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
