"use client";

import { useGSAP } from "@gsap/react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Spinner } from "@workspace/ui/components/spinner";
import gsap from "gsap";
import { Check, Eye, EyeOff, Github, LockKeyhole, Mail } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { AlphaLogo } from "@/components/alpha-logo";
import { getSiteConfig } from "@/lib/client-api";
import { isClientDemoMode } from "@/lib/runtime";
import {
  DEFAULT_SITE_DESCRIPTION,
  DEFAULT_SITE_NAME,
  SITE_CONFIG_QUERY_KEY,
  SITE_CONFIG_STALE_TIME,
} from "@/lib/site-brand";

type AuthMode = "login" | "register" | "reset";

async function jsonRequest(path: string, body: unknown) {
  const response = await fetch(path, {
    body: JSON.stringify(body),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  const payload = (await response.json()) as {
    authenticated?: boolean;
    message?: string;
    reset?: boolean;
  };
  if (!response.ok) throw new Error(payload.message || "操作失败");
  return payload;
}

export function AuthScreen() {
  const root = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const config = useQuery({
    queryFn: getSiteConfig,
    queryKey: SITE_CONFIG_QUERY_KEY,
    staleTime: SITE_CONFIG_STALE_TIME,
  });
  const [mode, setMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [invite, setInvite] = useState("");
  const [visible, setVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [githubLoading, setGithubLoading] = useState(false);
  const [cooldown, setCooldown] = useState(0);

  useGSAP(
    () => {
      const targets = root.current?.querySelectorAll("[data-auth-reveal]");
      if (!targets?.length) return;
      const media = gsap.matchMedia();
      media.add("(prefers-reduced-motion: no-preference)", () => {
        gsap.fromTo(
          targets,
          { autoAlpha: 0, y: 12 },
          {
            autoAlpha: 1,
            duration: 0.58,
            ease: "power3.out",
            stagger: 0.065,
            y: 0,
          }
        );
      });
      return () => media.revert();
    },
    { dependencies: [mode], revertOnUpdate: true }
  );

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = window.setInterval(
      () => setCooldown((current) => Math.max(0, current - 1)),
      1000
    );
    return () => window.clearInterval(timer);
  }, [cooldown]);

  const sendCode = async () => {
    if (!email.includes("@")) {
      toast.error("请先填写有效邮箱");
      return;
    }
    if (isClientDemoMode()) {
      setCooldown(60);
      toast.success("验证码已发送");
      return;
    }
    try {
      await jsonRequest("/api/auth/send-code", {
        email,
        type: mode === "register" ? 1 : 2,
      });
      setCooldown(60);
      toast.success("验证码已发送");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "发送失败");
    }
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (
      !email.includes("@") ||
      password.length < (mode === "login" ? 1 : 12) ||
      password.length > (mode === "login" ? 128 : 72)
    ) {
      toast.error(
        mode === "login" ? "请输入邮箱和密码" : "密码需要 12 至 72 位"
      );
      return;
    }
    setLoading(true);
    try {
      if (isClientDemoMode()) {
        toast.success(
          mode === "register"
            ? "账户已创建"
            : mode === "reset"
              ? "密码已重置"
              : "登录成功"
        );
        if (mode === "reset") setMode("login");
        else router.push("/dashboard");
        return;
      }
      const endpoint =
        mode === "login"
          ? "/api/auth/login"
          : mode === "register"
            ? "/api/auth/register"
            : "/api/auth/reset";
      const result = await jsonRequest(endpoint, {
        code,
        email,
        invite,
        password,
      });
      if (mode === "reset" && !result.authenticated) {
        toast.success("密码已重置，请重新登录");
        setMode("login");
      } else {
        toast.success(
          mode === "register"
            ? "注册成功"
            : mode === "reset"
              ? "密码已重置"
              : "登录成功"
        );
        router.push("/dashboard");
        router.refresh();
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "操作失败");
    } finally {
      setLoading(false);
    }
  };

  const githubLogin = async () => {
    setGithubLoading(true);
    try {
      if (isClientDemoMode()) {
        toast.success("GitHub 登录成功");
        router.push("/dashboard");
        return;
      }
      const response = await fetch("/api/auth/oauth/start", { method: "POST" });
      const payload = (await response.json()) as {
        message?: string;
        redirect?: string;
      };
      if (!(response.ok && payload.redirect))
        throw new Error(payload.message || "GitHub 登录失败");
      window.location.assign(payload.redirect);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "GitHub 登录失败");
      setGithubLoading(false);
    }
  };

  const githubEnabled = config.data?.oauthMethods.includes("github") ?? true;
  const siteName = config.data?.siteName || DEFAULT_SITE_NAME;
  const title =
    mode === "login"
      ? `登录 ${siteName}`
      : mode === "register"
        ? `创建 ${siteName} 账户`
        : "重置密码";
  const description =
    mode === "login"
      ? config.data?.siteDescription || DEFAULT_SITE_DESCRIPTION
      : mode === "register"
        ? "一个账户可以管理多个独立订阅。"
        : "验证邮箱后设置新的账户密码。";

  return (
    <div className="flex min-h-screen flex-col bg-muted/20" ref={root}>
      <main className="flex flex-1 flex-col items-center justify-center px-4 py-10 sm:py-14">
        <div className="mb-8" data-auth-reveal>
          <AlphaLogo />
        </div>
        <Card className="w-full max-w-[500px] gap-0 border-0 bg-transparent py-0 shadow-none">
          <CardHeader className="gap-2 px-6 pt-0 pb-7 text-center sm:px-9">
            <CardTitle className="text-3xl tracking-tight" data-auth-reveal>
              {title}
            </CardTitle>
            <CardDescription className="text-sm" data-auth-reveal>
              {description}
            </CardDescription>
          </CardHeader>
          <CardContent className="px-6 pb-8 sm:px-9">
            <FieldGroup className="gap-5">
              {githubEnabled && mode !== "reset" ? (
                <Button
                  className="h-11 w-full bg-background"
                  data-auth-reveal
                  disabled={githubLoading}
                  onClick={githubLogin}
                  type="button"
                  variant="outline"
                >
                  {githubLoading ? (
                    <Spinner />
                  ) : (
                    <Github data-icon="inline-start" />
                  )}
                  使用 GitHub 继续
                </Button>
              ) : null}

              {mode !== "reset" ? (
                <FieldSeparator data-auth-reveal>或使用邮箱</FieldSeparator>
              ) : null}

              <form onSubmit={submit}>
                <FieldGroup className="gap-5">
                  <Field data-auth-reveal>
                    <FieldLabel htmlFor="auth-email">电子邮箱</FieldLabel>
                    <InputGroup className="h-11">
                      <InputGroupInput
                        id="auth-email"
                        autoComplete="email"
                        onChange={(event) => setEmail(event.target.value)}
                        placeholder="name@example.com"
                        type="email"
                        value={email}
                      />
                      <InputGroupAddon>
                        <Mail />
                      </InputGroupAddon>
                    </InputGroup>
                  </Field>
                  {mode !== "login" ? (
                    <Field data-auth-reveal>
                      <FieldLabel htmlFor="auth-code">验证码</FieldLabel>
                      <InputGroup className="h-11">
                        <InputGroupInput
                          id="auth-code"
                          inputMode="numeric"
                          maxLength={8}
                          onChange={(event) =>
                            setCode(event.target.value.replace(/\D/g, ""))
                          }
                          pattern="[0-9]*"
                          placeholder="邮箱验证码"
                          value={code}
                        />
                        <InputGroupAddon>
                          <Check />
                        </InputGroupAddon>
                        <InputGroupAddon align="inline-end">
                          <InputGroupButton
                            disabled={cooldown > 0}
                            onClick={sendCode}
                            size="sm"
                          >
                            {cooldown > 0 ? `${cooldown}s` : "发送验证码"}
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </Field>
                  ) : null}
                  <Field data-auth-reveal>
                    <div className="flex items-center justify-between gap-3">
                      <FieldLabel htmlFor="auth-password">
                        {mode === "reset" ? "新密码" : "密码"}
                      </FieldLabel>
                      {mode === "login" ? (
                        <Button
                          className="h-auto px-0 py-0 font-normal"
                          onClick={() => setMode("reset")}
                          type="button"
                          variant="link"
                        >
                          忘记密码？
                        </Button>
                      ) : null}
                    </div>
                    <InputGroup className="h-11">
                      <InputGroupInput
                        id="auth-password"
                        autoComplete={
                          mode === "login" ? "current-password" : "new-password"
                        }
                        maxLength={mode === "login" ? 128 : 72}
                        minLength={mode === "login" ? 1 : 12}
                        onChange={(event) => setPassword(event.target.value)}
                        placeholder={
                          mode === "login" ? "输入密码" : "12 至 72 位密码"
                        }
                        type={visible ? "text" : "password"}
                        value={password}
                      />
                      <InputGroupAddon>
                        <LockKeyhole />
                      </InputGroupAddon>
                      <InputGroupAddon align="inline-end">
                        <InputGroupButton
                          aria-label={visible ? "隐藏密码" : "显示密码"}
                          onClick={() => setVisible((current) => !current)}
                          size="icon-sm"
                          title={visible ? "隐藏密码" : "显示密码"}
                        >
                          {visible ? <EyeOff /> : <Eye />}
                        </InputGroupButton>
                      </InputGroupAddon>
                    </InputGroup>
                    {mode !== "login" ? (
                      <FieldDescription>
                        密码长度需要为 12 至 72 位。
                      </FieldDescription>
                    ) : null}
                  </Field>
                  {mode === "register" ? (
                    <Field data-auth-reveal>
                      <FieldLabel htmlFor="auth-invite">
                        邀请码{" "}
                        <span className="text-muted-foreground">选填</span>
                      </FieldLabel>
                      <InputGroup className="h-11">
                        <InputGroupInput
                          id="auth-invite"
                          onChange={(event) => setInvite(event.target.value)}
                          placeholder="输入邀请码"
                          value={invite}
                        />
                        <InputGroupAddon>
                          <Check />
                        </InputGroupAddon>
                      </InputGroup>
                    </Field>
                  ) : null}
                  <Button
                    className="h-11 w-full"
                    data-auth-reveal
                    disabled={loading}
                    size="lg"
                    type="submit"
                  >
                    {loading ? <Spinner /> : null}
                    {mode === "login"
                      ? "登录"
                      : mode === "register"
                        ? "注册"
                        : "重置密码"}
                  </Button>
                </FieldGroup>
              </form>
            </FieldGroup>
          </CardContent>
          <CardFooter
            className="flex-col gap-2 px-6 py-2 text-center text-sm sm:px-9"
            data-auth-reveal
          >
            {mode === "login" ? (
              <div className="flex items-center justify-center gap-1 text-muted-foreground">
                <span>还没有账户？</span>
                <Button
                  className="h-auto px-1 py-0"
                  onClick={() => setMode("register")}
                  type="button"
                  variant="link"
                >
                  注册
                </Button>
              </div>
            ) : (
              <div className="flex items-center justify-center gap-1 text-muted-foreground">
                <span>已有账户？</span>
                <Button
                  className="h-auto px-1 py-0"
                  onClick={() => setMode("login")}
                  type="button"
                  variant="link"
                >
                  返回登录
                </Button>
              </div>
            )}
          </CardFooter>
        </Card>
      </main>
      <footer className="px-4 pb-7 text-center text-muted-foreground text-xs leading-5">
        继续操作即表示你同意 {siteName} 的服务条款和隐私政策。
      </footer>
    </div>
  );
}
