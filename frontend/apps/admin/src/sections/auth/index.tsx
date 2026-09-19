"use client";

import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@workspace/ui/components/avatar";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { LanguageSwitch } from "@workspace/ui/composed/language-switch";
import { ThemeSwitch } from "@workspace/ui/composed/theme-switch";
import { ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useGlobalStore } from "@/stores/global";
import EmailAuthForm from "./email/auth-form";
import type { EmailAuthMode } from "./email/types";

export default function Auth() {
  const { t } = useTranslation("auth");
  const site = useGlobalStore((store) => store.common.site);
  const siteName = site.site_name || "Alpha";
  const [mode, setMode] = useState<EmailAuthMode>("login");
  const title =
    mode === "reset"
      ? t("reset.title", "Reset Password")
      : mode === "register"
        ? t("register.title", "Register")
        : t("adminSignIn", "Sign in to the console");
  const description =
    mode === "reset"
      ? t("reset.description", "Reset your password")
      : mode === "register"
        ? t("register.description", "Create an administrator account")
        : t("continueTo", "Continue to {{siteName}}", { siteName });

  return (
    <div className="admin-auth min-h-svh bg-background text-foreground">
      <header className="flex h-16 items-center justify-end gap-1 px-4 sm:px-8">
        <LanguageSwitch />
        <ThemeSwitch />
      </header>
      <main className="grid min-h-[calc(100svh-8rem)] place-items-center px-4 py-8">
        <div className="flex w-full max-w-[25rem] flex-col gap-7">
          <div className="flex items-center justify-center gap-3">
            <Avatar className="size-9 rounded-md">
              <AvatarImage
                alt=""
                className="object-contain"
                src={site.site_logo || `${import.meta.env.BASE_URL}favicon.png`}
              />
              <AvatarFallback className="rounded-md">
                {siteName.slice(0, 1)}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0">
              <p className="break-words font-semibold text-base">{siteName}</p>
              <p className="text-xs text-muted-foreground">
                {t("consoleBadge", "Admin Console")}
              </p>
            </div>
          </div>
          <Card className="admin-auth-card">
            <CardHeader className="gap-2">
              <CardTitle>
                <h1 className="text-xl font-semibold">{title}</h1>
              </CardTitle>
              <CardDescription>{description}</CardDescription>
            </CardHeader>
            <CardContent>
              <EmailAuthForm mode={mode} onModeChange={setMode} />
            </CardContent>
          </Card>
          <p className="flex items-start justify-center gap-2 text-center text-xs leading-5 text-muted-foreground">
            <ShieldCheck className="mt-0.5 size-3.5 shrink-0" />
            {t(
              "authorizedOnly",
              "Access is restricted to authorized administrators."
            )}
          </p>
        </div>
      </main>
    </div>
  );
}
