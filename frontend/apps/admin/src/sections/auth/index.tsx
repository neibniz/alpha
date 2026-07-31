"use client";

import { Link } from "@tanstack/react-router";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@workspace/ui/components/avatar";
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
import { LanguageSwitch } from "@workspace/ui/composed/language-switch";
import { ThemeSwitch } from "@workspace/ui/composed/theme-switch";
import { LockKeyhole, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useGlobalStore } from "@/stores/global";
import EmailAuthForm from "./email/auth-form";
import type { EmailAuthMode } from "./email/types";

export default function Auth() {
  const { t } = useTranslation("auth");
  const common = useGlobalStore((state) => state.common);
  const { site } = common;
  const [mode, setMode] = useState<EmailAuthMode>("login");
  const authCopy =
    mode === "reset"
      ? {
          title: t("reset.title", "Reset Password"),
          description: t("reset.description", "Reset your password"),
        }
      : mode === "register"
        ? {
            title: t("register.title", "Register"),
            description: t(
              "register.description",
              "Create an administrator account"
            ),
          }
        : {
            title: t("signInTitle", "Sign in"),
            description: t("continueTo", "Continue to {{siteName}}", {
              siteName: site.site_name,
            }),
          };
  return (
    <div className="min-h-svh bg-muted/30 text-foreground">
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
          <Button asChild className="-ml-2 h-9 px-2" variant="ghost">
            <Link to="/">
              <Avatar className="size-7 rounded-md border">
                <AvatarImage
                  alt={site.site_name || "Alpha Panel"}
                  src={
                    site.site_logo || `${import.meta.env.BASE_URL}favicon.png`
                  }
                />
                <AvatarFallback className="rounded-md">A</AvatarFallback>
              </Avatar>
              <span className="font-semibold">
                {site.site_name || "Alpha Panel"}
              </span>
            </Link>
          </Button>
          <div className="flex items-center gap-1">
            <LanguageSwitch />
            <ThemeSwitch />
          </div>
        </div>
      </header>

      <main className="grid min-h-[calc(100svh-3.5rem)] place-items-center px-4 py-8 sm:px-6 sm:py-12">
        <div className="flex w-full max-w-md flex-col items-center gap-4">
          <Badge className="gap-1.5" variant="outline">
            <ShieldCheck data-icon="inline-start" />
            {t("consoleBadge", "Admin Console")}
          </Badge>

          <Card className="w-full">
            <CardHeader className="justify-items-center text-center">
              <div className="mb-1 flex size-11 items-center justify-center rounded-lg border bg-muted text-muted-foreground">
                <LockKeyhole />
              </div>
              <CardTitle>
                <h1 className="text-balance text-xl">{authCopy.title}</h1>
              </CardTitle>
              <CardDescription>{authCopy.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <EmailAuthForm mode={mode} onModeChange={setMode} />
            </CardContent>
            <CardFooter className="justify-center border-t text-center text-muted-foreground text-xs">
              <ShieldCheck className="mr-1.5 size-3.5" />
              {t(
                "authorizedOnly",
                "Access is restricted to authorized administrators."
              )}
            </CardFooter>
          </Card>

          <p className="text-center text-muted-foreground text-xs">
            {site.site_name || "Alpha Panel"}
          </p>
        </div>
      </main>
    </div>
  );
}
